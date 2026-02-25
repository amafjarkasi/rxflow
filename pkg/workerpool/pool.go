// Package workerpool provides a bounded worker pool for controlled concurrency.
// Designed for high-throughput prescription processing at 5,000+ TPS.
package workerpool

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// Task represents a unit of work to be processed
type Task struct {
	ID      string
	Payload interface{}
	Context context.Context
}

// Result represents the outcome of task processing
type Result struct {
	TaskID  string
	Success bool
	Error   error
	Data    interface{}
}

// WorkerFunc is the function signature for task processing
type WorkerFunc func(ctx context.Context, task *Task) *Result

// Config holds worker pool configuration
type Config struct {
	// Workers is the number of concurrent workers
	Workers int
	// QueueSize is the size of the task queue
	QueueSize int
	// MaxRetries is the maximum number of retries for failed tasks
	MaxRetries int
	// RetryDelay is the delay between retries
	RetryDelay time.Duration
	// GracefulShutdownTimeout is the timeout for graceful shutdown
	GracefulShutdownTimeout time.Duration
}

// DefaultConfig returns sensible defaults for high-throughput processing
func DefaultConfig() Config {
	return Config{
		Workers:                 100,
		QueueSize:               10000,
		MaxRetries:              3,
		RetryDelay:              100 * time.Millisecond,
		GracefulShutdownTimeout: 30 * time.Second,
	}
}

// Pool manages a pool of workers for concurrent task processing
type Pool struct {
	config     Config
	workerFunc WorkerFunc
	logger     *zap.Logger

	taskChan   chan *Task
	resultChan chan *Result
	wg         sync.WaitGroup

	ctx    context.Context
	cancel context.CancelFunc

	// Metrics
	tasksSubmitted int64
	tasksCompleted int64
	tasksFailed    int64
	tasksRetried   int64
	activeWorkers  int64
	queueDepth     int64
}

// New creates a new worker pool
func New(cfg Config, fn WorkerFunc, logger *zap.Logger) (*Pool, error) {
	if fn == nil {
		return nil, fmt.Errorf("worker function is required")
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	if cfg.Workers <= 0 {
		cfg.Workers = DefaultConfig().Workers
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = DefaultConfig().QueueSize
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool := &Pool{
		config:     cfg,
		workerFunc: fn,
		logger:     logger,
		taskChan:   make(chan *Task, cfg.QueueSize),
		resultChan: make(chan *Result, cfg.QueueSize),
		ctx:        ctx,
		cancel:     cancel,
	}

	return pool, nil
}

// Start launches all workers
func (p *Pool) Start() {
	for i := 0; i < p.config.Workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
	p.logger.Info("worker pool started",
		zap.Int("workers", p.config.Workers),
		zap.Int("queue_size", p.config.QueueSize))
}

// Submit adds a task to the queue
func (p *Pool) Submit(task *Task) error {
	select {
	case <-p.ctx.Done():
		return fmt.Errorf("pool is shutting down")
	default:
	}

	select {
	case p.taskChan <- task:
		atomic.AddInt64(&p.tasksSubmitted, 1)
		atomic.AddInt64(&p.queueDepth, 1)
		return nil
	default:
		return fmt.Errorf("task queue is full")
	}
}

// SubmitWait adds a task and waits for completion
func (p *Pool) SubmitWait(ctx context.Context, task *Task) (*Result, error) {
	if err := p.Submit(task); err != nil {
		return nil, err
	}

	// Wait for result with context
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-p.resultChan:
		if result.TaskID == task.ID {
			return result, nil
		}
		// Put result back if not ours (not ideal, but handles race)
		p.resultChan <- result
		return nil, fmt.Errorf("timeout waiting for task result")
	}
}

// Results returns the result channel for async processing
func (p *Pool) Results() <-chan *Result {
	return p.resultChan
}

// Stop gracefully shuts down the pool
func (p *Pool) Stop() error {
	p.logger.Info("stopping worker pool")

	// Signal shutdown
	p.cancel()

	// Close task channel to stop workers from receiving new tasks
	close(p.taskChan)

	// Wait for workers with timeout
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.logger.Info("worker pool stopped gracefully")
	case <-time.After(p.config.GracefulShutdownTimeout):
		p.logger.Warn("worker pool shutdown timed out")
	}

	close(p.resultChan)
	return nil
}

// worker is the main worker goroutine
func (p *Pool) worker(id int) {
	defer p.wg.Done()

	p.logger.Debug("worker started", zap.Int("worker_id", id))
	atomic.AddInt64(&p.activeWorkers, 1)
	defer atomic.AddInt64(&p.activeWorkers, -1)

	for task := range p.taskChan {
		atomic.AddInt64(&p.queueDepth, -1)
		p.processTask(id, task)
	}

	p.logger.Debug("worker stopped", zap.Int("worker_id", id))
}

// processTask handles a single task with retries
func (p *Pool) processTask(workerID int, task *Task) {
	var result *Result
	var lastErr error

	ctx := task.Context
	if ctx == nil {
		ctx = p.ctx
	}

	for attempt := 0; attempt <= p.config.MaxRetries; attempt++ {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			result = &Result{
				TaskID:  task.ID,
				Success: false,
				Error:   ctx.Err(),
			}
			goto done
		default:
		}

		// Execute the task
		result = p.workerFunc(ctx, task)

		if result.Success {
			goto done
		}

		lastErr = result.Error

		// Don't retry on last attempt
		if attempt < p.config.MaxRetries {
			atomic.AddInt64(&p.tasksRetried, 1)
			p.logger.Debug("retrying task",
				zap.String("task_id", task.ID),
				zap.Int("attempt", attempt+1),
				zap.Error(lastErr))

			select {
			case <-ctx.Done():
				result = &Result{
					TaskID:  task.ID,
					Success: false,
					Error:   ctx.Err(),
				}
				goto done
			case <-time.After(p.config.RetryDelay * time.Duration(attempt+1)):
				// Exponential backoff
			}
		}
	}

	// All retries exhausted
	result = &Result{
		TaskID:  task.ID,
		Success: false,
		Error:   fmt.Errorf("task failed after %d retries: %w", p.config.MaxRetries, lastErr),
	}

done:
	if result.Success {
		atomic.AddInt64(&p.tasksCompleted, 1)
	} else {
		atomic.AddInt64(&p.tasksFailed, 1)
		p.logger.Error("task failed",
			zap.String("task_id", task.ID),
			zap.Int("worker_id", workerID),
			zap.Error(result.Error))
	}

	// Send result (non-blocking)
	select {
	case p.resultChan <- result:
	default:
		p.logger.Warn("result channel full, dropping result",
			zap.String("task_id", task.ID))
	}
}

// Stats returns current pool statistics
type Stats struct {
	TasksSubmitted int64
	TasksCompleted int64
	TasksFailed    int64
	TasksRetried   int64
	ActiveWorkers  int64
	QueueDepth     int64
	QueueCapacity  int
	Workers        int
}

// Stats returns current pool statistics
func (p *Pool) Stats() Stats {
	return Stats{
		TasksSubmitted: atomic.LoadInt64(&p.tasksSubmitted),
		TasksCompleted: atomic.LoadInt64(&p.tasksCompleted),
		TasksFailed:    atomic.LoadInt64(&p.tasksFailed),
		TasksRetried:   atomic.LoadInt64(&p.tasksRetried),
		ActiveWorkers:  atomic.LoadInt64(&p.activeWorkers),
		QueueDepth:     atomic.LoadInt64(&p.queueDepth),
		QueueCapacity:  p.config.QueueSize,
		Workers:        p.config.Workers,
	}
}

// IsHealthy returns true if the pool is operating normally
func (p *Pool) IsHealthy() bool {
	stats := p.Stats()
	// Healthy if queue isn't backing up significantly
	return float64(stats.QueueDepth)/float64(stats.QueueCapacity) < 0.9
}

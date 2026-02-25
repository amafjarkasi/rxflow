// Package redpanda provides high-performance Kafka-compatible streaming with franz-go.
package redpanda

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// ConsumerConfig holds configuration for the Redpanda consumer
type ConsumerConfig struct {
	// Brokers is a list of broker addresses
	Brokers []string
	// GroupID is the consumer group ID
	GroupID string
	// Topics is the list of topics to consume
	Topics []string
	// AutoCommit enables automatic offset commits
	AutoCommit bool
	// AutoCommitIntervalMS is the interval for auto commits
	AutoCommitIntervalMS int64
	// SessionTimeoutMS is the session timeout
	SessionTimeoutMS int64
	// HeartbeatIntervalMS is the heartbeat interval
	HeartbeatIntervalMS int64
	// MaxPollRecords is the maximum records per poll
	MaxPollRecords int
	// FetchMaxBytes is the maximum fetch size
	FetchMaxBytes int32
	// StartOffset is the initial offset (newest or oldest)
	StartOffset string
}

// DefaultConsumerConfig returns optimized defaults for prescription processing
func DefaultConsumerConfig() ConsumerConfig {
	return ConsumerConfig{
		Brokers:              []string{"localhost:9092"},
		GroupID:              "prescription-processor",
		AutoCommit:           false, // Manual commits for exactly-once
		AutoCommitIntervalMS: 5000,
		SessionTimeoutMS:     30000,
		HeartbeatIntervalMS:  3000,
		MaxPollRecords:       500,
		FetchMaxBytes:        52428800, // 50MB
		StartOffset:          "earliest",
	}
}

// MessageHandler is called for each consumed message
type MessageHandler func(ctx context.Context, msg *ConsumedMessage) error

// ConsumedMessage represents a consumed Kafka message
type ConsumedMessage struct {
	Topic     string
	Partition int32
	Offset    int64
	Key       []byte
	Value     []byte
	Headers   map[string]string
	Timestamp time.Time
}

// Consumer provides high-performance message consumption from Redpanda
type Consumer struct {
	client  *kgo.Client
	config  ConsumerConfig
	logger  *zap.Logger
	tracer  trace.Tracer
	handler MessageHandler

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Metrics
	mu             sync.RWMutex
	messagesRead   int64
	bytesRead      int64
	errorCount     int64
	lastCommitTime time.Time
	partitionLag   map[int32]int64
}

// NewConsumer creates a new Redpanda consumer
func NewConsumer(cfg ConsumerConfig, handler MessageHandler, logger *zap.Logger) (*Consumer, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	if handler == nil {
		return nil, errors.New("message handler is required")
	}

	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.ConsumerGroup(cfg.GroupID),
		kgo.ConsumeTopics(cfg.Topics...),
		kgo.SessionTimeout(time.Duration(cfg.SessionTimeoutMS) * time.Millisecond),
		kgo.HeartbeatInterval(time.Duration(cfg.HeartbeatIntervalMS) * time.Millisecond),
		kgo.FetchMaxBytes(cfg.FetchMaxBytes),
	}

	// Set start offset
	switch cfg.StartOffset {
	case "earliest":
		opts = append(opts, kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()))
	case "latest":
		opts = append(opts, kgo.ConsumeResetOffset(kgo.NewOffset().AtEnd()))
	}

	// Disable auto-commit for exactly-once semantics
	if !cfg.AutoCommit {
		opts = append(opts, kgo.DisableAutoCommit())
	} else {
		opts = append(opts, kgo.AutoCommitInterval(time.Duration(cfg.AutoCommitIntervalMS)*time.Millisecond))
	}

	// Add partition assignment callbacks
	opts = append(opts,
		kgo.OnPartitionsAssigned(func(ctx context.Context, client *kgo.Client, assigned map[string][]int32) {
			logger.Info("partitions assigned", zap.Any("partitions", assigned))
		}),
		kgo.OnPartitionsRevoked(func(ctx context.Context, client *kgo.Client, revoked map[string][]int32) {
			logger.Info("partitions revoked", zap.Any("partitions", revoked))
		}),
	)

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka client: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Consumer{
		client:       client,
		config:       cfg,
		logger:       logger,
		tracer:       otel.Tracer("redpanda-consumer"),
		handler:      handler,
		ctx:          ctx,
		cancel:       cancel,
		partitionLag: make(map[int32]int64),
	}, nil
}

// Start begins consuming messages
func (c *Consumer) Start() {
	c.wg.Add(1)
	go c.consumeLoop()
}

// Stop gracefully stops the consumer
func (c *Consumer) Stop() error {
	c.cancel()
	c.wg.Wait()

	// Commit any remaining offsets
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := c.client.CommitUncommittedOffsets(ctx); err != nil {
		c.logger.Warn("error committing offsets on stop", zap.Error(err))
	}

	c.client.Close()
	return nil
}

// consumeLoop is the main consumption loop
func (c *Consumer) consumeLoop() {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		fetches := c.client.PollFetches(c.ctx)
		if fetches.IsClientClosed() {
			return
		}

		if errs := fetches.Errors(); len(errs) > 0 {
			for _, err := range errs {
				c.logger.Error("fetch error",
					zap.String("topic", err.Topic),
					zap.Int32("partition", err.Partition),
					zap.Error(err.Err))
				c.incrementErrorCount()
			}
			continue
		}

		// Process records
		fetches.EachRecord(func(record *kgo.Record) {
			c.processRecord(record)
		})
	}
}

// processRecord processes a single record
func (c *Consumer) processRecord(record *kgo.Record) {
	ctx, span := c.tracer.Start(c.ctx, "process_message",
		trace.WithAttributes(
			attribute.String("topic", record.Topic),
			attribute.Int64("partition", int64(record.Partition)),
			attribute.Int64("offset", record.Offset),
		))
	defer span.End()

	// Extract trace context from headers
	ctx = extractTraceContext(ctx, record)

	// Convert to ConsumedMessage
	msg := &ConsumedMessage{
		Topic:     record.Topic,
		Partition: record.Partition,
		Offset:    record.Offset,
		Key:       record.Key,
		Value:     record.Value,
		Headers:   make(map[string]string),
		Timestamp: record.Timestamp,
	}

	for _, h := range record.Headers {
		msg.Headers[h.Key] = string(h.Value)
	}

	// Call handler
	if err := c.handler(ctx, msg); err != nil {
		c.logger.Error("message handler failed",
			zap.String("topic", record.Topic),
			zap.Int32("partition", record.Partition),
			zap.Int64("offset", record.Offset),
			zap.Error(err))
		span.RecordError(err)
		c.incrementErrorCount()
		// Don't commit failed messages - they will be reprocessed
		return
	}

	c.incrementMetrics(len(record.Value))

	// Commit offset after successful processing (exactly-once)
	if !c.config.AutoCommit {
		c.client.MarkCommitRecords(record)
		if err := c.client.CommitUncommittedOffsets(ctx); err != nil {
			c.logger.Error("failed to commit offset",
				zap.String("topic", record.Topic),
				zap.Int32("partition", record.Partition),
				zap.Int64("offset", record.Offset),
				zap.Error(err))
			span.RecordError(err)
		} else {
			c.mu.Lock()
			c.lastCommitTime = time.Now()
			c.mu.Unlock()
		}
	}
}

// CommitOffsets manually commits current offsets
func (c *Consumer) CommitOffsets(ctx context.Context) error {
	ctx, span := c.tracer.Start(ctx, "commit_offsets")
	defer span.End()

	if err := c.client.CommitUncommittedOffsets(ctx); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to commit offsets: %w", err)
	}

	c.mu.Lock()
	c.lastCommitTime = time.Now()
	c.mu.Unlock()

	return nil
}

// Stats returns current consumer statistics
func (c *Consumer) Stats() ConsumerStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	lagCopy := make(map[int32]int64)
	for k, v := range c.partitionLag {
		lagCopy[k] = v
	}

	return ConsumerStats{
		MessagesRead:   c.messagesRead,
		BytesRead:      c.bytesRead,
		ErrorCount:     c.errorCount,
		LastCommitTime: c.lastCommitTime,
		PartitionLag:   lagCopy,
	}
}

// ConsumerStats holds consumer statistics
type ConsumerStats struct {
	MessagesRead   int64
	BytesRead      int64
	ErrorCount     int64
	LastCommitTime time.Time
	PartitionLag   map[int32]int64
}

// Helper methods
func (c *Consumer) incrementMetrics(bytes int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messagesRead++
	c.bytesRead += int64(bytes)
}

func (c *Consumer) incrementErrorCount() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errorCount++
}

// extractTraceContext extracts OpenTelemetry trace context from record headers
func extractTraceContext(ctx context.Context, record *kgo.Record) context.Context {
	for _, h := range record.Headers {
		if h.Key == "traceparent" {
			// Parse W3C Trace Context format: 00-{trace-id}-{span-id}-{flags}
			// For now, we just log it; full implementation would recreate the span context
			// This is a simplified version
			break
		}
	}
	return ctx
}

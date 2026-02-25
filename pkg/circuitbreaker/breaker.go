// Package circuitbreaker provides resilience patterns for external service calls.
// Wraps sony/gobreaker with OpenTelemetry integration and prescription-specific defaults.
package circuitbreaker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sony/gobreaker"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// State represents the circuit breaker state
type State string

const (
	StateClosed   State = "closed"
	StateOpen     State = "open"
	StateHalfOpen State = "half-open"
)

// Config holds circuit breaker configuration
type Config struct {
	// Name identifies the circuit breaker
	Name string
	// MaxRequests is max requests allowed in half-open state
	MaxRequests uint32
	// Interval is the cyclic period for clearing counts in closed state
	Interval time.Duration
	// Timeout is how long to wait before transitioning from open to half-open
	Timeout time.Duration
	// FailureThreshold is the number of failures before opening
	FailureThreshold uint32
	// SuccessThreshold is the number of successes needed to close from half-open
	SuccessThreshold uint32
	// FailureRatio is the failure ratio threshold (alternative to count)
	FailureRatio float64
	// MinRequests is minimum requests before ratio is considered
	MinRequests uint32
}

// DefaultConfig returns defaults suitable for pharmacy routing services
func DefaultConfig(name string) Config {
	return Config{
		Name:             name,
		MaxRequests:      3,
		Interval:         60 * time.Second,
		Timeout:          30 * time.Second,
		FailureThreshold: 5,
		SuccessThreshold: 3,
		FailureRatio:     0.6, // Open if 60% of requests fail
		MinRequests:      10,
	}
}

// CircuitBreaker wraps gobreaker with observability
type CircuitBreaker struct {
	cb     *gobreaker.CircuitBreaker
	name   string
	logger *zap.Logger
	tracer trace.Tracer

	// Metrics
	meter          metric.Meter
	stateGauge     metric.Int64ObservableGauge
	requestCounter metric.Int64Counter
	failureCounter metric.Int64Counter
	successCounter metric.Int64Counter
	timeoutCounter metric.Int64Counter
	currentState   State
	stateMu        sync.RWMutex
}

// New creates a new circuit breaker
func New(cfg Config, logger *zap.Logger) (*CircuitBreaker, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	cb := &CircuitBreaker{
		name:         cfg.Name,
		logger:       logger,
		tracer:       otel.Tracer("circuit-breaker"),
		meter:        otel.Meter("circuit-breaker"),
		currentState: StateClosed,
	}

	// Initialize metrics
	var err error
	cb.requestCounter, err = cb.meter.Int64Counter("circuit_breaker_requests_total",
		metric.WithDescription("Total requests through circuit breaker"))
	if err != nil {
		return nil, fmt.Errorf("failed to create request counter: %w", err)
	}

	cb.failureCounter, err = cb.meter.Int64Counter("circuit_breaker_failures_total",
		metric.WithDescription("Total failed requests"))
	if err != nil {
		return nil, fmt.Errorf("failed to create failure counter: %w", err)
	}

	cb.successCounter, err = cb.meter.Int64Counter("circuit_breaker_successes_total",
		metric.WithDescription("Total successful requests"))
	if err != nil {
		return nil, fmt.Errorf("failed to create success counter: %w", err)
	}

	cb.timeoutCounter, err = cb.meter.Int64Counter("circuit_breaker_timeouts_total",
		metric.WithDescription("Total requests rejected due to open circuit"))
	if err != nil {
		return nil, fmt.Errorf("failed to create timeout counter: %w", err)
	}

	// Create gobreaker settings
	settings := gobreaker.Settings{
		Name:        cfg.Name,
		MaxRequests: cfg.MaxRequests,
		Interval:    cfg.Interval,
		Timeout:     cfg.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// Use either failure count or ratio
			if counts.Requests < cfg.MinRequests {
				return counts.ConsecutiveFailures >= cfg.FailureThreshold
			}
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return failureRatio >= cfg.FailureRatio
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			cb.onStateChange(from, to)
		},
		IsSuccessful: func(err error) bool {
			// Define what constitutes a successful request
			// Some errors might not indicate service issues
			return err == nil
		},
	}

	cb.cb = gobreaker.NewCircuitBreaker(settings)

	return cb, nil
}

// Execute runs a function through the circuit breaker
func (c *CircuitBreaker) Execute(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	ctx, span := c.tracer.Start(ctx, "circuit_breaker_execute",
		trace.WithAttributes(
			attribute.String("breaker_name", c.name),
			attribute.String("state", string(c.GetState())),
		))
	defer span.End()

	c.requestCounter.Add(ctx, 1, metric.WithAttributes(attribute.String("name", c.name)))

	result, err := c.cb.Execute(fn)

	if err != nil {
		if err == gobreaker.ErrOpenState || err == gobreaker.ErrTooManyRequests {
			c.timeoutCounter.Add(ctx, 1, metric.WithAttributes(attribute.String("name", c.name)))
			span.SetAttributes(attribute.Bool("circuit_open", true))
		} else {
			c.failureCounter.Add(ctx, 1, metric.WithAttributes(attribute.String("name", c.name)))
		}
		span.RecordError(err)
		return nil, err
	}

	c.successCounter.Add(ctx, 1, metric.WithAttributes(attribute.String("name", c.name)))
	return result, nil
}

// ExecuteWithFallback runs a function with a fallback on circuit open
func (c *CircuitBreaker) ExecuteWithFallback(ctx context.Context, fn func() (interface{}, error), fallback func(error) (interface{}, error)) (interface{}, error) {
	result, err := c.Execute(ctx, fn)
	if err != nil {
		if err == gobreaker.ErrOpenState || err == gobreaker.ErrTooManyRequests {
			c.logger.Warn("circuit open, using fallback",
				zap.String("breaker", c.name),
				zap.Error(err))
			return fallback(err)
		}
		return nil, err
	}
	return result, nil
}

// GetState returns the current circuit breaker state
func (c *CircuitBreaker) GetState() State {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.currentState
}

// onStateChange handles state transitions
func (c *CircuitBreaker) onStateChange(from, to gobreaker.State) {
	fromState := mapState(from)
	toState := mapState(to)

	c.stateMu.Lock()
	c.currentState = toState
	c.stateMu.Unlock()

	c.logger.Warn("circuit breaker state changed",
		zap.String("breaker", c.name),
		zap.String("from", string(fromState)),
		zap.String("to", string(toState)))
}

// mapState converts gobreaker.State to our State type
func mapState(s gobreaker.State) State {
	switch s {
	case gobreaker.StateClosed:
		return StateClosed
	case gobreaker.StateOpen:
		return StateOpen
	case gobreaker.StateHalfOpen:
		return StateHalfOpen
	default:
		return StateClosed
	}
}

// IsOpen returns true if the circuit is open
func (c *CircuitBreaker) IsOpen() bool {
	return c.GetState() == StateOpen
}

// IsClosed returns true if the circuit is closed
func (c *CircuitBreaker) IsClosed() bool {
	return c.GetState() == StateClosed
}

// Counts returns the current counts from the circuit breaker
func (c *CircuitBreaker) Counts() gobreaker.Counts {
	return c.cb.Counts()
}

// Manager manages multiple circuit breakers
type Manager struct {
	breakers map[string]*CircuitBreaker
	mu       sync.RWMutex
	logger   *zap.Logger
}

// NewManager creates a circuit breaker manager
func NewManager(logger *zap.Logger) *Manager {
	return &Manager{
		breakers: make(map[string]*CircuitBreaker),
		logger:   logger,
	}
}

// GetOrCreate returns an existing breaker or creates a new one
func (m *Manager) GetOrCreate(name string, cfg Config) (*CircuitBreaker, error) {
	m.mu.RLock()
	if cb, ok := m.breakers[name]; ok {
		m.mu.RUnlock()
		return cb, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if cb, ok := m.breakers[name]; ok {
		return cb, nil
	}

	cfg.Name = name
	cb, err := New(cfg, m.logger)
	if err != nil {
		return nil, err
	}

	m.breakers[name] = cb
	return cb, nil
}

// Get returns a circuit breaker by name
func (m *Manager) Get(name string) (*CircuitBreaker, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cb, ok := m.breakers[name]
	return cb, ok
}

// All returns all circuit breakers
func (m *Manager) All() map[string]*CircuitBreaker {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*CircuitBreaker, len(m.breakers))
	for k, v := range m.breakers {
		result[k] = v
	}
	return result
}

// HealthStatus returns the health status of all breakers
type HealthStatus struct {
	Name     string
	State    State
	Requests uint32
	Failures uint32
	Healthy  bool
}

// GetHealthStatus returns health status for all circuit breakers
func (m *Manager) GetHealthStatus() []HealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var statuses []HealthStatus
	for name, cb := range m.breakers {
		counts := cb.Counts()
		statuses = append(statuses, HealthStatus{
			Name:     name,
			State:    cb.GetState(),
			Requests: counts.Requests,
			Failures: counts.TotalFailures,
			Healthy:  cb.IsClosed(),
		})
	}
	return statuses
}

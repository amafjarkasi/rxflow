// Package postgres provides PostgreSQL infrastructure components.
// Implements Transactional Outbox pattern for reliable event publishing.
package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// OutboxEntry represents an event to be published via the outbox pattern
type OutboxEntry struct {
	ID            int64
	AggregateID   string
	AggregateType string
	EventType     string
	Payload       json.RawMessage
	KafkaTopic    string
	KafkaKey      string
	CreatedAt     time.Time
	ProcessedAt   *time.Time
	RetryCount    int
	LastError     *string
}

// OutboxConfig holds configuration for the outbox processor
type OutboxConfig struct {
	// BatchSize is the number of entries to process per batch
	BatchSize int
	// PollInterval is how often to poll for new entries
	PollInterval time.Duration
	// MaxRetries is the maximum retries before moving to dead letter
	MaxRetries int
	// LockTimeout is the advisory lock timeout
	LockTimeout time.Duration
}

// DefaultOutboxConfig returns sensible defaults
func DefaultOutboxConfig() OutboxConfig {
	return OutboxConfig{
		BatchSize:    100,
		PollInterval: 100 * time.Millisecond,
		MaxRetries:   5,
		LockTimeout:  30 * time.Second,
	}
}

// OutboxPublisher defines the interface for publishing outbox entries
type OutboxPublisher interface {
	Publish(ctx context.Context, topic, key string, value []byte) error
}

// Outbox manages the transactional outbox pattern
type Outbox struct {
	pool      *pgxpool.Pool
	config    OutboxConfig
	publisher OutboxPublisher
	logger    *zap.Logger
	tracer    trace.Tracer

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

// NewOutbox creates a new outbox processor
func NewOutbox(pool *pgxpool.Pool, publisher OutboxPublisher, cfg OutboxConfig, logger *zap.Logger) *Outbox {
	if logger == nil {
		logger = zap.NewNop()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Outbox{
		pool:      pool,
		config:    cfg,
		publisher: publisher,
		logger:    logger,
		tracer:    otel.Tracer("outbox"),
		ctx:       ctx,
		cancel:    cancel,
		done:      make(chan struct{}),
	}
}

// WriteEntry writes an outbox entry within a transaction
// This should be called within the same transaction as the domain operation
func WriteEntry(ctx context.Context, tx pgx.Tx, entry *OutboxEntry) error {
	query := `
		INSERT INTO outbox (aggregate_id, aggregate_type, event_type, payload, kafka_topic, kafka_key)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`

	err := tx.QueryRow(ctx, query,
		entry.AggregateID,
		entry.AggregateType,
		entry.EventType,
		entry.Payload,
		entry.KafkaTopic,
		entry.KafkaKey,
	).Scan(&entry.ID, &entry.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to write outbox entry: %w", err)
	}

	return nil
}

// Start begins polling and processing outbox entries
func (o *Outbox) Start() {
	go o.processLoop()
	o.logger.Info("outbox processor started",
		zap.Int("batch_size", o.config.BatchSize),
		zap.Duration("poll_interval", o.config.PollInterval))
}

// Stop gracefully stops the outbox processor
func (o *Outbox) Stop() {
	o.cancel()
	<-o.done
	o.logger.Info("outbox processor stopped")
}

// processLoop is the main processing loop
func (o *Outbox) processLoop() {
	defer close(o.done)

	ticker := time.NewTicker(o.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-o.ctx.Done():
			return
		case <-ticker.C:
			o.processBatch()
		}
	}
}

// processBatch processes a batch of outbox entries
func (o *Outbox) processBatch() {
	ctx, span := o.tracer.Start(o.ctx, "outbox_process_batch")
	defer span.End()

	// Try to acquire advisory lock for this processor
	lockID := int64(123456789) // Use a consistent lock ID
	var acquired bool
	err := o.pool.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", lockID).Scan(&acquired)
	if err != nil || !acquired {
		return // Another processor has the lock
	}
	defer o.pool.Exec(ctx, "SELECT pg_advisory_unlock($1)", lockID)

	// Fetch unprocessed entries
	entries, err := o.fetchUnprocessed(ctx)
	if err != nil {
		o.logger.Error("failed to fetch outbox entries", zap.Error(err))
		span.RecordError(err)
		return
	}

	if len(entries) == 0 {
		return
	}

	span.SetAttributes(attribute.Int("batch_size", len(entries)))

	for _, entry := range entries {
		if err := o.processEntry(ctx, entry); err != nil {
			o.logger.Error("failed to process outbox entry",
				zap.Int64("id", entry.ID),
				zap.String("event_type", entry.EventType),
				zap.Error(err))
		}
	}
}

// fetchUnprocessed retrieves unprocessed outbox entries
func (o *Outbox) fetchUnprocessed(ctx context.Context) ([]*OutboxEntry, error) {
	query := `
		SELECT id, aggregate_id, aggregate_type, event_type, payload, 
		       kafka_topic, kafka_key, created_at, retry_count, last_error
		FROM outbox
		WHERE processed_at IS NULL
		  AND retry_count < $1
		ORDER BY created_at ASC
		LIMIT $2
		FOR UPDATE SKIP LOCKED
	`

	rows, err := o.pool.Query(ctx, query, o.config.MaxRetries, o.config.BatchSize)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var entries []*OutboxEntry
	for rows.Next() {
		entry := &OutboxEntry{}
		err := rows.Scan(
			&entry.ID, &entry.AggregateID, &entry.AggregateType,
			&entry.EventType, &entry.Payload, &entry.KafkaTopic,
			&entry.KafkaKey, &entry.CreatedAt, &entry.RetryCount, &entry.LastError,
		)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// processEntry publishes a single entry and marks it processed
func (o *Outbox) processEntry(ctx context.Context, entry *OutboxEntry) error {
	ctx, span := o.tracer.Start(ctx, "outbox_process_entry",
		trace.WithAttributes(
			attribute.Int64("entry_id", entry.ID),
			attribute.String("event_type", entry.EventType),
			attribute.String("aggregate_id", entry.AggregateID),
		))
	defer span.End()

	// Publish to Kafka
	err := o.publisher.Publish(ctx, entry.KafkaTopic, entry.KafkaKey, entry.Payload)
	if err != nil {
		// Update retry count and error
		updateQuery := `
			UPDATE outbox 
			SET retry_count = retry_count + 1, last_error = $1, updated_at = NOW()
			WHERE id = $2
		`
		errStr := err.Error()
		if _, updateErr := o.pool.Exec(ctx, updateQuery, errStr, entry.ID); updateErr != nil {
			o.logger.Error("failed to update retry count", zap.Error(updateErr))
		}
		span.RecordError(err)
		return fmt.Errorf("publish failed: %w", err)
	}

	// Mark as processed
	markQuery := `
		UPDATE outbox 
		SET processed_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`
	if _, err := o.pool.Exec(ctx, markQuery, entry.ID); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to mark processed: %w", err)
	}

	o.logger.Debug("outbox entry processed",
		zap.Int64("id", entry.ID),
		zap.String("topic", entry.KafkaTopic))

	return nil
}

// CleanupProcessed removes old processed entries
func (o *Outbox) CleanupProcessed(ctx context.Context, olderThan time.Duration) (int64, error) {
	query := `
		DELETE FROM outbox
		WHERE processed_at IS NOT NULL
		  AND processed_at < NOW() - $1::interval
	`

	result, err := o.pool.Exec(ctx, query, olderThan.String())
	if err != nil {
		return 0, fmt.Errorf("cleanup failed: %w", err)
	}

	return result.RowsAffected(), nil
}

// MoveToDeadLetter moves failed entries to dead letter queue
func (o *Outbox) MoveToDeadLetter(ctx context.Context) (int64, error) {
	// For entries that exceeded max retries, publish to dead letter topic
	query := `
		SELECT id, aggregate_id, aggregate_type, event_type, payload,
		       kafka_key, created_at, retry_count, last_error
		FROM outbox
		WHERE processed_at IS NULL
		  AND retry_count >= $1
		FOR UPDATE SKIP LOCKED
	`

	rows, err := o.pool.Query(ctx, query, o.config.MaxRetries)
	if err != nil {
		return 0, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var count int64
	for rows.Next() {
		entry := &OutboxEntry{}
		err := rows.Scan(
			&entry.ID, &entry.AggregateID, &entry.AggregateType,
			&entry.EventType, &entry.Payload, &entry.KafkaKey,
			&entry.CreatedAt, &entry.RetryCount, &entry.LastError,
		)
		if err != nil {
			continue
		}

		// Publish to dead letter topic
		dlPayload, _ := json.Marshal(map[string]interface{}{
			"original_topic": entry.KafkaTopic,
			"event_type":     entry.EventType,
			"aggregate_id":   entry.AggregateID,
			"payload":        entry.Payload,
			"retry_count":    entry.RetryCount,
			"last_error":     entry.LastError,
			"created_at":     entry.CreatedAt,
		})

		if err := o.publisher.Publish(ctx, "dead.letter", entry.KafkaKey, dlPayload); err != nil {
			o.logger.Error("failed to publish to dead letter", zap.Error(err))
			continue
		}

		// Mark as processed (moved to DLQ)
		if _, err := o.pool.Exec(ctx, "UPDATE outbox SET processed_at = NOW() WHERE id = $1", entry.ID); err != nil {
			o.logger.Error("failed to mark DLQ entry", zap.Error(err))
			continue
		}

		count++
	}

	return count, nil
}

// Stats returns outbox statistics
type OutboxStats struct {
	Pending       int64
	Processed     int64
	Failed        int64
	OldestPending *time.Time
}

// GetStats returns current outbox statistics
func (o *Outbox) GetStats(ctx context.Context) (*OutboxStats, error) {
	stats := &OutboxStats{}

	// Pending count
	err := o.pool.QueryRow(ctx, "SELECT COUNT(*) FROM outbox WHERE processed_at IS NULL AND retry_count < $1", o.config.MaxRetries).Scan(&stats.Pending)
	if err != nil {
		return nil, err
	}

	// Processed count (last 24 hours)
	err = o.pool.QueryRow(ctx, "SELECT COUNT(*) FROM outbox WHERE processed_at IS NOT NULL AND processed_at > NOW() - INTERVAL '24 hours'").Scan(&stats.Processed)
	if err != nil {
		return nil, err
	}

	// Failed count
	err = o.pool.QueryRow(ctx, "SELECT COUNT(*) FROM outbox WHERE processed_at IS NULL AND retry_count >= $1", o.config.MaxRetries).Scan(&stats.Failed)
	if err != nil {
		return nil, err
	}

	// Oldest pending
	o.pool.QueryRow(ctx, "SELECT MIN(created_at) FROM outbox WHERE processed_at IS NULL").Scan(&stats.OldestPending)

	return stats, nil
}

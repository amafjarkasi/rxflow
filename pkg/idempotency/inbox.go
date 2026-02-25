// Package idempotency provides the Inbox pattern for exactly-once message processing.
// Uses deterministic idempotency keys: Hash(PrescriberID+PatientID+Medication+Timestamp)
package idempotency

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Status represents the processing status of an inbox entry
type Status string

const (
	StatusStarted     Status = "STARTED"
	StatusFinished    Status = "FINISHED"
	StatusRecoverable Status = "RECOVERABLE"
	StatusFailed      Status = "FAILED"
)

// InboxEntry represents an idempotency inbox record
type InboxEntry struct {
	IdempotencyKey string
	HandlerName    string
	Status         Status
	Payload        json.RawMessage
	Result         json.RawMessage
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ExpiresAt      *time.Time
}

// InboxConfig holds configuration for the inbox
type InboxConfig struct {
	// DefaultTTL is the default time-to-live for inbox entries
	DefaultTTL time.Duration
	// CleanupInterval is how often to clean expired entries
	CleanupInterval time.Duration
	// RecoveryTimeout is when to consider a STARTED entry as stale
	RecoveryTimeout time.Duration
}

// DefaultInboxConfig returns sensible defaults
func DefaultInboxConfig() InboxConfig {
	return InboxConfig{
		DefaultTTL:      7 * 24 * time.Hour, // 7 days
		CleanupInterval: 1 * time.Hour,
		RecoveryTimeout: 5 * time.Minute,
	}
}

// Inbox manages idempotent message processing
type Inbox struct {
	pool   *pgxpool.Pool
	config InboxConfig
	logger *zap.Logger
	tracer trace.Tracer

	// Control for cleanup goroutine
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

// NewInbox creates a new inbox manager
func NewInbox(pool *pgxpool.Pool, cfg InboxConfig, logger *zap.Logger) *Inbox {
	if logger == nil {
		logger = zap.NewNop()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Inbox{
		pool:   pool,
		config: cfg,
		logger: logger,
		tracer: otel.Tracer("inbox"),
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
	}
}

// ErrDuplicateMessage indicates message was already processed
var ErrDuplicateMessage = errors.New("duplicate message: already processed")

// ErrMessageInProgress indicates message is currently being processed
var ErrMessageInProgress = errors.New("message in progress by another handler")

// ProcessResult represents the result of idempotent processing
type ProcessResult struct {
	IsNew        bool
	WasRecovered bool
	Result       json.RawMessage
}

// ProcessFunc is the function signature for idempotent handlers
type ProcessFunc func(ctx context.Context, payload json.RawMessage) (json.RawMessage, error)

// Process executes a handler with idempotency guarantees
func (i *Inbox) Process(ctx context.Context, key, handlerName string, payload json.RawMessage, fn ProcessFunc) (*ProcessResult, error) {
	ctx, span := i.tracer.Start(ctx, "inbox_process",
		trace.WithAttributes(
			attribute.String("idempotency_key", key),
			attribute.String("handler", handlerName),
		))
	defer span.End()

	// Check if already processed
	entry, err := i.getEntry(ctx, key)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to check inbox: %w", err)
	}

	if entry != nil {
		switch entry.Status {
		case StatusFinished:
			// Already successfully processed
			span.SetAttributes(attribute.Bool("duplicate", true))
			return &ProcessResult{
				IsNew:  false,
				Result: entry.Result,
			}, nil

		case StatusFailed:
			// Previously failed permanently, don't reprocess
			span.SetAttributes(attribute.Bool("previously_failed", true))
			return nil, fmt.Errorf("message previously failed permanently: %s", key)

		case StatusStarted:
			// Check if stale (potential crash recovery)
			if time.Since(entry.UpdatedAt) > i.config.RecoveryTimeout {
				// Recover stale entry
				if err := i.markRecoverable(ctx, key); err != nil {
					return nil, fmt.Errorf("failed to mark recoverable: %w", err)
				}
				// Continue to process
			} else {
				// Still in progress
				return nil, ErrMessageInProgress
			}

		case StatusRecoverable:
			// Can be reprocessed
			span.SetAttributes(attribute.Bool("recovered", true))
		}
	}

	// Create or update entry as STARTED
	if err := i.startProcessing(ctx, key, handlerName, payload); err != nil {
		if errors.Is(err, ErrDuplicateMessage) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to start processing: %w", err)
	}

	// Execute handler
	result, handlerErr := fn(ctx, payload)

	if handlerErr != nil {
		// Mark as recoverable or failed based on error type
		status := StatusRecoverable
		if isTerminalError(handlerErr) {
			status = StatusFailed
		}
		if err := i.markStatus(ctx, key, status, nil, handlerErr.Error()); err != nil {
			i.logger.Error("failed to mark error status", zap.Error(err))
		}
		span.RecordError(handlerErr)
		return nil, handlerErr
	}

	// Mark as finished
	if err := i.markFinished(ctx, key, result); err != nil {
		i.logger.Error("failed to mark finished", zap.Error(err))
		// Don't fail - the handler succeeded
	}

	return &ProcessResult{
		IsNew:        entry == nil,
		WasRecovered: entry != nil && entry.Status == StatusRecoverable,
		Result:       result,
	}, nil
}

// GenerateKey creates a deterministic idempotency key from prescription components
func GenerateKey(prescriberNPI, patientID, medicationCode string, timestamp time.Time) string {
	// Truncate timestamp to minute for clock drift tolerance
	truncatedTime := timestamp.Truncate(time.Minute).Format(time.RFC3339)

	parts := []string{
		prescriberNPI,
		patientID,
		medicationCode,
		truncatedTime,
	}

	data := strings.Join(parts, "|")
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// getEntry retrieves an inbox entry by key
func (i *Inbox) getEntry(ctx context.Context, key string) (*InboxEntry, error) {
	query := `
		SELECT idempotency_key, handler_name, status, payload, result, created_at, updated_at, expires_at
		FROM inbox
		WHERE idempotency_key = $1
	`

	entry := &InboxEntry{}
	err := i.pool.QueryRow(ctx, query, key).Scan(
		&entry.IdempotencyKey, &entry.HandlerName, &entry.Status,
		&entry.Payload, &entry.Result, &entry.CreatedAt, &entry.UpdatedAt, &entry.ExpiresAt,
	)
	if err != nil {
		return nil, err
	}

	return entry, nil
}

// startProcessing creates or updates an entry as STARTED
func (i *Inbox) startProcessing(ctx context.Context, key, handlerName string, payload json.RawMessage) error {
	expiresAt := time.Now().Add(i.config.DefaultTTL)

	query := `
		INSERT INTO inbox (idempotency_key, handler_name, status, payload, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (idempotency_key) DO UPDATE
		SET status = $3, updated_at = NOW()
		WHERE inbox.status IN ('RECOVERABLE')
		RETURNING idempotency_key
	`

	var returned string
	err := i.pool.QueryRow(ctx, query, key, handlerName, StatusStarted, payload, expiresAt).Scan(&returned)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Conflict but not recoverable - duplicate
			return ErrDuplicateMessage
		}
		return err
	}

	return nil
}

// markFinished marks an entry as successfully finished
func (i *Inbox) markFinished(ctx context.Context, key string, result json.RawMessage) error {
	query := `
		UPDATE inbox
		SET status = $1, result = $2, updated_at = NOW()
		WHERE idempotency_key = $3
	`

	_, err := i.pool.Exec(ctx, query, StatusFinished, result, key)
	return err
}

// markRecoverable marks an entry as recoverable
func (i *Inbox) markRecoverable(ctx context.Context, key string) error {
	query := `
		UPDATE inbox
		SET status = $1, updated_at = NOW()
		WHERE idempotency_key = $2
	`

	_, err := i.pool.Exec(ctx, query, StatusRecoverable, key)
	return err
}

// markStatus marks an entry with a status and optional error
func (i *Inbox) markStatus(ctx context.Context, key string, status Status, result json.RawMessage, errMsg string) error {
	query := `
		UPDATE inbox
		SET status = $1, result = $2, updated_at = NOW()
		WHERE idempotency_key = $3
	`

	// Store error in result as JSON
	if errMsg != "" && result == nil {
		result, _ = json.Marshal(map[string]string{"error": errMsg})
	}

	_, err := i.pool.Exec(ctx, query, status, result, key)
	return err
}

// StartCleanup starts the background cleanup goroutine
func (i *Inbox) StartCleanup() {
	go i.cleanupLoop()
	i.logger.Info("inbox cleanup started", zap.Duration("interval", i.config.CleanupInterval))
}

// Stop stops the inbox cleanup
func (i *Inbox) Stop() {
	i.cancel()
	<-i.done
	i.logger.Info("inbox stopped")
}

// cleanupLoop periodically cleans expired entries
func (i *Inbox) cleanupLoop() {
	defer close(i.done)

	ticker := time.NewTicker(i.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-i.ctx.Done():
			return
		case <-ticker.C:
			if err := i.cleanup(i.ctx); err != nil {
				i.logger.Error("inbox cleanup failed", zap.Error(err))
			}
		}
	}
}

// cleanup removes expired entries
func (i *Inbox) cleanup(ctx context.Context) error {
	query := `
		DELETE FROM inbox
		WHERE expires_at < NOW()
		   OR (status = 'FINISHED' AND updated_at < NOW() - INTERVAL '7 days')
	`

	result, err := i.pool.Exec(ctx, query)
	if err != nil {
		return err
	}

	if result.RowsAffected() > 0 {
		i.logger.Info("inbox cleanup completed", zap.Int64("deleted", result.RowsAffected()))
	}

	return nil
}

// RecoverStaleEntries marks stale STARTED entries as RECOVERABLE
func (i *Inbox) RecoverStaleEntries(ctx context.Context) (int64, error) {
	query := `
		UPDATE inbox
		SET status = 'RECOVERABLE', updated_at = NOW()
		WHERE status = 'STARTED'
		  AND updated_at < NOW() - $1::interval
	`

	result, err := i.pool.Exec(ctx, query, i.config.RecoveryTimeout.String())
	if err != nil {
		return 0, err
	}

	return result.RowsAffected(), nil
}

// isTerminalError determines if an error should not be retried
func isTerminalError(err error) bool {
	// Add checks for terminal errors
	// Examples: validation errors, business rule violations
	errStr := err.Error()
	terminalPhrases := []string{
		"validation",
		"invalid",
		"not found",
		"unauthorized",
		"forbidden",
	}
	for _, phrase := range terminalPhrases {
		if strings.Contains(strings.ToLower(errStr), phrase) {
			return true
		}
	}
	return false
}

// Stats returns inbox statistics
type InboxStats struct {
	TotalEntries int64
	Started      int64
	Finished     int64
	Recoverable  int64
	Failed       int64
}

// GetStats returns current inbox statistics
func (i *Inbox) GetStats(ctx context.Context) (*InboxStats, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'STARTED') as started,
			COUNT(*) FILTER (WHERE status = 'FINISHED') as finished,
			COUNT(*) FILTER (WHERE status = 'RECOVERABLE') as recoverable,
			COUNT(*) FILTER (WHERE status = 'FAILED') as failed
		FROM inbox
	`

	stats := &InboxStats{}
	err := i.pool.QueryRow(ctx, query).Scan(
		&stats.TotalEntries, &stats.Started, &stats.Finished,
		&stats.Recoverable, &stats.Failed,
	)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

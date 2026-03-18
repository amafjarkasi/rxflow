package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/drfirst/go-oec/internal/domain/prescription"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// PrescriptionRepository implements the event-sourced repository for prescriptions
type PrescriptionRepository struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
	tracer trace.Tracer
}

// NewPrescriptionRepository creates a new instance of the repository
func NewPrescriptionRepository(pool *pgxpool.Pool, logger *zap.Logger) *PrescriptionRepository {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &PrescriptionRepository{
		pool:   pool,
		logger: logger,
		tracer: otel.Tracer("postgres_repository"),
	}
}

// Save persists all uncommitted events for a prescription aggregate using a single transaction.
// It also writes them to the outbox for reliable messaging.
func (r *PrescriptionRepository) Save(ctx context.Context, aggregate *prescription.Aggregate) error {
	ctx, span := r.tracer.Start(ctx, "postgres_save_aggregate", trace.WithAttributes(
		attribute.String("aggregate_id", aggregate.ID()),
	))
	defer span.End()

	changes := aggregate.Changes()
	if len(changes) == 0 {
		return nil // Nothing to save
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	// Use explicit rollback in defer; it is a no-op if tx is already committed
	defer tx.Rollback(ctx)

	// Append events to event store
	for _, event := range changes {
		eventData, err := json.Marshal(event.EventData)
		if err != nil {
			return fmt.Errorf("failed to marshal event data: %w", err)
		}

		// Save to event store table
		query := `
			INSERT INTO prescription_events
			(id, aggregate_id, aggregate_type, event_type, event_data, version, timestamp, prescriber_npi, prescriber_dea, patient_hash, correlation_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`
		_, err = tx.Exec(ctx, query,
			event.ID,
			event.AggregateID,
			event.AggregateType,
			string(event.EventType),
			eventData,
			event.Version,
			event.Timestamp,
			event.PrescriberNPI,
			event.PrescriberDEA,
			event.PatientHash,
			event.CorrelationID,
		)
		if err != nil {
			return fmt.Errorf("failed to insert event %s: %w", event.ID, err)
		}

		// Also write to outbox for the Transactional Outbox pattern
		outboxPayload, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal outbox payload: %w", err)
		}

		outboxEntry := &OutboxEntry{
			AggregateID:   event.AggregateID,
			AggregateType: event.AggregateType,
			EventType:     string(event.EventType),
			Payload:       outboxPayload,
			KafkaTopic:    "prescription.events",
			KafkaKey:      event.AggregateID,
		}

		if err := WriteEntry(ctx, tx, outboxEntry); err != nil {
			return fmt.Errorf("failed to write to outbox: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Important: clear changes after successful commit
	aggregate.ClearChanges()

	r.logger.Debug("saved prescription events",
		zap.String("aggregate_id", aggregate.ID()),
		zap.Int("event_count", len(changes)),
	)

	return nil
}

// Load retrieves an aggregate by its ID by replaying its entire event history
func (r *PrescriptionRepository) Load(ctx context.Context, id string) (*prescription.Aggregate, error) {
	ctx, span := r.tracer.Start(ctx, "postgres_load_aggregate", trace.WithAttributes(
		attribute.String("aggregate_id", id),
	))
	defer span.End()

	query := `
		SELECT id, aggregate_id, aggregate_type, event_type, event_data, version, timestamp,
		       prescriber_npi, prescriber_dea, patient_hash, correlation_id
		FROM prescription_events
		WHERE aggregate_id = $1
		ORDER BY version ASC
	`

	rows, err := r.pool.Query(ctx, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []*prescription.Event
	for rows.Next() {
		event := &prescription.Event{}
		var eventType string

		err := rows.Scan(
			&event.ID,
			&event.AggregateID,
			&event.AggregateType,
			&eventType,
			&event.EventData,
			&event.Version,
			&event.Timestamp,
			&event.PrescriberNPI,
			&event.PrescriberDEA,
			&event.PatientHash,
			&event.CorrelationID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		event.EventType = prescription.EventType(eventType)
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating event rows: %w", err)
	}

	if len(events) == 0 {
		return nil, fmt.Errorf("prescription not found: %s", id)
	}

	aggregate := prescription.NewAggregate(id)
	aggregate.LoadFromHistory(events)

	return aggregate, nil
}

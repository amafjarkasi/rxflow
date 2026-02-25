// Package prescription provides the event store repository.
package prescription

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// Repository provides event sourcing persistence
type Repository struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewRepository creates a new repository
func NewRepository(pool *pgxpool.Pool, logger *zap.Logger) *Repository {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Repository{pool: pool, logger: logger}
}

// Save persists new events for an aggregate
func (r *Repository) Save(ctx context.Context, agg *Aggregate) error {
	if len(agg.Changes()) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	for i, event := range agg.Changes() {
		event.Version = agg.Version() - len(agg.Changes()) + i + 1
		if err := r.insertEvent(ctx, tx, event); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	agg.ClearChanges()
	return nil
}

func (r *Repository) insertEvent(ctx context.Context, tx pgx.Tx, event *Event) error {
	query := `
		INSERT INTO prescription_events 
		(aggregate_id, event_type, event_data, version, prescriber_npi, prescriber_dea, patient_hash, correlation_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := tx.Exec(ctx, query,
		event.AggregateID,
		event.EventType,
		event.EventData,
		event.Version,
		event.PrescriberNPI,
		event.PrescriberDEA,
		event.PatientHash,
		event.CorrelationID,
	)
	return err
}

// Load retrieves an aggregate by ID
func (r *Repository) Load(ctx context.Context, id string) (*Aggregate, error) {
	events, err := r.GetEvents(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return nil, fmt.Errorf("aggregate not found: %s", id)
	}

	agg := NewAggregate(id)
	agg.LoadFromHistory(events)
	return agg, nil
}

// GetEvents retrieves all events for an aggregate
func (r *Repository) GetEvents(ctx context.Context, aggregateID string) ([]*Event, error) {
	query := `
		SELECT aggregate_id, event_type, event_data, version, timestamp,
		       prescriber_npi, prescriber_dea, patient_hash, correlation_id
		FROM prescription_events
		WHERE aggregate_id = $1
		ORDER BY version ASC
	`

	rows, err := r.pool.Query(ctx, query, aggregateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		e := &Event{}
		err := rows.Scan(
			&e.AggregateID, &e.EventType, &e.EventData, &e.Version,
			&e.Timestamp, &e.PrescriberNPI, &e.PrescriberDEA,
			&e.PatientHash, &e.CorrelationID,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// GetEventsByType retrieves events by type
func (r *Repository) GetEventsByType(ctx context.Context, eventType EventType, limit int) ([]*Event, error) {
	query := `
		SELECT aggregate_id, event_type, event_data, version, timestamp
		FROM prescription_events
		WHERE event_type = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, eventType, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		e := &Event{}
		err := rows.Scan(&e.AggregateID, &e.EventType, &e.EventData, &e.Version, &e.Timestamp)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// init sets up the json unmarshaler
func init() {
	jsonUnmarshal = json.Unmarshal
}

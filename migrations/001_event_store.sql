-- Migration 001: Event Store (TimescaleDB hypertable)
-- Stores all prescription domain events for event sourcing and DEA audit trail

CREATE TABLE IF NOT EXISTS prescription_events (
    event_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    aggregate_id UUID NOT NULL,
    aggregate_type VARCHAR(50) NOT NULL DEFAULT 'Prescription',
    event_type VARCHAR(100) NOT NULL,
    event_version INT NOT NULL DEFAULT 1,
    event_data JSONB NOT NULL,
    metadata JSONB,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- DEA/EPCS Audit Trail Fields
    prescriber_npi VARCHAR(10),
    prescriber_dea VARCHAR(20),
    patient_hash VARCHAR(64),  -- SHA-256 hash for privacy-preserving indexing
    
    -- Ensure ordering within aggregate
    CONSTRAINT unique_aggregate_version UNIQUE (aggregate_id, event_version)
);

-- Convert to TimescaleDB hypertable for efficient time-series queries
SELECT create_hypertable('prescription_events', 'occurred_at', 
    chunk_time_interval => INTERVAL '1 day',
    if_not_exists => TRUE
);

-- Indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_events_aggregate 
ON prescription_events (aggregate_id, event_version);

CREATE INDEX IF NOT EXISTS idx_events_type 
ON prescription_events (event_type, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_events_prescriber_dea 
ON prescription_events (prescriber_dea, occurred_at DESC)
WHERE prescriber_dea IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_events_patient_hash 
ON prescription_events (patient_hash, occurred_at DESC)
WHERE patient_hash IS NOT NULL;

-- Compression policy for older events (after 7 days)
ALTER TABLE prescription_events SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'aggregate_id',
    timescaledb.compress_orderby = 'occurred_at DESC'
);

SELECT add_compression_policy('prescription_events', INTERVAL '7 days', if_not_exists => TRUE);

-- Retention policy (keep 7 years for DEA compliance)
SELECT add_retention_policy('prescription_events', INTERVAL '2555 days', if_not_exists => TRUE);

-- Comments for documentation
COMMENT ON TABLE prescription_events IS 'Event-sourced storage for prescription lifecycle events - DEA audit trail';
COMMENT ON COLUMN prescription_events.patient_hash IS 'SHA-256 hash of patient identifiers for privacy-preserving queries';

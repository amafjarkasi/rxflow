-- Migration 002: Transactional Outbox
-- Solves the dual-write problem by atomically writing events to be published

CREATE TABLE IF NOT EXISTS outbox (
    id BIGSERIAL PRIMARY KEY,
    aggregate_id UUID NOT NULL,
    aggregate_type VARCHAR(50) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    kafka_topic VARCHAR(255) NOT NULL,
    kafka_key VARCHAR(255),
    kafka_headers JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    retry_count INT NOT NULL DEFAULT 0,
    last_error TEXT
);

-- Partial index for efficient polling of unprocessed messages
CREATE INDEX IF NOT EXISTS idx_outbox_unprocessed 
ON outbox (created_at ASC) 
WHERE processed_at IS NULL;

-- Index for retry handling
CREATE INDEX IF NOT EXISTS idx_outbox_retry 
ON outbox (retry_count, created_at ASC) 
WHERE processed_at IS NULL AND retry_count > 0;

-- Index for aggregate lookup (debugging/auditing)
CREATE INDEX IF NOT EXISTS idx_outbox_aggregate 
ON outbox (aggregate_id, created_at DESC);

-- Comments
COMMENT ON TABLE outbox IS 'Transactional outbox for reliable event publishing to Redpanda';
COMMENT ON COLUMN outbox.processed_at IS 'NULL indicates pending, timestamp indicates successfully published';
COMMENT ON COLUMN outbox.retry_count IS 'Number of failed publication attempts';

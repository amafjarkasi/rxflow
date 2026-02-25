-- Migration 003: Idempotency Inbox
-- Ensures exactly-once processing semantics for incoming messages

CREATE TYPE inbox_status AS ENUM ('STARTED', 'FINISHED', 'RECOVERABLE', 'FAILED');

CREATE TABLE IF NOT EXISTS idempotency_inbox (
    idempotency_key VARCHAR(64) PRIMARY KEY,
    handler_name VARCHAR(100) NOT NULL,
    status inbox_status NOT NULL DEFAULT 'STARTED',
    request_payload JSONB,
    result_payload JSONB,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

-- Index for cleanup of old completed entries
CREATE INDEX IF NOT EXISTS idx_inbox_cleanup 
ON idempotency_inbox (completed_at) 
WHERE completed_at IS NOT NULL;

-- Index for finding recoverable entries
CREATE INDEX IF NOT EXISTS idx_inbox_recoverable 
ON idempotency_inbox (created_at) 
WHERE status = 'RECOVERABLE';

-- Auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION update_inbox_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER inbox_timestamp_trigger
BEFORE UPDATE ON idempotency_inbox
FOR EACH ROW
EXECUTE FUNCTION update_inbox_timestamp();

-- Comments
COMMENT ON TABLE idempotency_inbox IS 'Inbox pattern for idempotent message processing';
COMMENT ON COLUMN idempotency_inbox.idempotency_key IS 'Hash of PrescriberID+PatientID+Medication+Timestamp';

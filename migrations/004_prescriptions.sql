-- Migration 004: Prescriptions Read Model (CQRS Query Side)
-- Denormalized view optimized for fast queries

CREATE TABLE IF NOT EXISTS prescriptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    external_id VARCHAR(100) UNIQUE,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    
    -- Original FHIR payload
    fhir_payload JSONB NOT NULL,
    
    -- Denormalized patient fields
    patient_id VARCHAR(100),
    patient_hash VARCHAR(64),
    patient_name TEXT,
    patient_dob DATE,
    
    -- Denormalized prescriber fields
    prescriber_id VARCHAR(100),
    prescriber_npi VARCHAR(10),
    prescriber_dea VARCHAR(20),
    prescriber_name TEXT,
    
    -- Medication fields
    medication_code VARCHAR(50),
    medication_ndc VARCHAR(20),
    medication_rxnorm VARCHAR(20),
    medication_name TEXT,
    medication_strength TEXT,
    quantity DECIMAL(10,2),
    quantity_unit VARCHAR(20),
    days_supply INT,
    refills_allowed INT DEFAULT 0,
    
    -- DEA schedule for controlled substances
    dea_schedule VARCHAR(5),
    is_controlled BOOLEAN DEFAULT FALSE,
    
    -- Pharmacy routing
    pharmacy_ncpdp_id VARCHAR(20),
    pharmacy_name TEXT,
    routing_status VARCHAR(50),
    routed_at TIMESTAMPTZ,
    
    -- NCPDP SCRIPT transmission
    script_message_id VARCHAR(100),
    script_transmitted_at TIMESTAMPTZ,
    script_status VARCHAR(50),
    
    -- Sig (directions)
    sig_text TEXT,
    
    -- Timestamps
    authored_on TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Full-text search vector
    search_vector TSVECTOR GENERATED ALWAYS AS (
        setweight(to_tsvector('english', COALESCE(patient_name, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(medication_name, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(prescriber_name, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(sig_text, '')), 'C')
    ) STORED
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_prescriptions_status ON prescriptions (status);
CREATE INDEX IF NOT EXISTS idx_prescriptions_patient ON prescriptions (patient_hash);
CREATE INDEX IF NOT EXISTS idx_prescriptions_prescriber_npi ON prescriptions (prescriber_npi);
CREATE INDEX IF NOT EXISTS idx_prescriptions_prescriber_dea ON prescriptions (prescriber_dea);
CREATE INDEX IF NOT EXISTS idx_prescriptions_medication ON prescriptions (medication_rxnorm);
CREATE INDEX IF NOT EXISTS idx_prescriptions_pharmacy ON prescriptions (pharmacy_ncpdp_id);
CREATE INDEX IF NOT EXISTS idx_prescriptions_authored ON prescriptions (authored_on DESC);
CREATE INDEX IF NOT EXISTS idx_prescriptions_created ON prescriptions (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_prescriptions_controlled ON prescriptions (is_controlled, created_at DESC) WHERE is_controlled = TRUE;
CREATE INDEX IF NOT EXISTS idx_prescriptions_search ON prescriptions USING GIN (search_vector);

-- Auto-update timestamp trigger
CREATE OR REPLACE FUNCTION update_prescription_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prescription_timestamp_trigger
BEFORE UPDATE ON prescriptions
FOR EACH ROW
EXECUTE FUNCTION update_prescription_timestamp();

-- Comments
COMMENT ON TABLE prescriptions IS 'CQRS read model for prescription queries - denormalized for performance';

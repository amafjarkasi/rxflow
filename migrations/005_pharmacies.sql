-- Migration 005: Pharmacy Registry
-- Stores pharmacy information for routing decisions

CREATE TABLE IF NOT EXISTS pharmacies (
    ncpdp_id VARCHAR(20) PRIMARY KEY,
    npi VARCHAR(10),
    name TEXT NOT NULL,
    dba_name TEXT,
    
    -- Address
    address_line1 TEXT,
    address_line2 TEXT,
    city VARCHAR(100),
    state VARCHAR(2),
    zip_code VARCHAR(10),
    country VARCHAR(3) DEFAULT 'USA',
    
    -- Contact
    phone VARCHAR(20),
    fax VARCHAR(20),
    email VARCHAR(255),
    
    -- Geolocation for proximity routing
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    
    -- Capabilities
    accepts_epcs BOOLEAN DEFAULT TRUE,
    accepts_script_2023011 BOOLEAN DEFAULT TRUE,
    supports_rtpb BOOLEAN DEFAULT FALSE,
    specialty_types TEXT[],
    
    -- Operating hours (JSON array of day schedules)
    operating_hours JSONB,
    
    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    verified_at TIMESTAMPTZ,
    
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_pharmacies_status ON pharmacies (status);
CREATE INDEX IF NOT EXISTS idx_pharmacies_state ON pharmacies (state);
CREATE INDEX IF NOT EXISTS idx_pharmacies_zip ON pharmacies (zip_code);
CREATE INDEX IF NOT EXISTS idx_pharmacies_location ON pharmacies (latitude, longitude);
CREATE INDEX IF NOT EXISTS idx_pharmacies_epcs ON pharmacies (accepts_epcs) WHERE accepts_epcs = TRUE;

-- Full-text search
CREATE INDEX IF NOT EXISTS idx_pharmacies_name_search 
ON pharmacies USING GIN (to_tsvector('english', name || ' ' || COALESCE(dba_name, '')));

-- Insert some sample pharmacies for demo
INSERT INTO pharmacies (ncpdp_id, npi, name, city, state, zip_code, latitude, longitude, status) VALUES
('1234567890', '1234567890', 'Demo Pharmacy #1', 'New York', 'NY', '10001', 40.7484, -73.9967, 'active'),
('2345678901', '2345678901', 'Demo Pharmacy #2', 'Los Angeles', 'CA', '90001', 34.0522, -118.2437, 'active'),
('3456789012', '3456789012', 'Demo Pharmacy #3', 'Chicago', 'IL', '60601', 41.8781, -87.6298, 'active')
ON CONFLICT (ncpdp_id) DO NOTHING;

COMMENT ON TABLE pharmacies IS 'Registry of pharmacies for prescription routing';

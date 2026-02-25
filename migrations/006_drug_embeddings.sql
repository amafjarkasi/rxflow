-- Migration 006: Drug Embeddings (pgvector)
-- Stores vector embeddings for AI-assisted routing and DDI prediction

CREATE TABLE IF NOT EXISTS drug_embeddings (
    id SERIAL PRIMARY KEY,
    ndc_code VARCHAR(20) UNIQUE,
    rxnorm_cui VARCHAR(20),
    drug_name TEXT NOT NULL,
    generic_name TEXT,
    brand_name TEXT,
    
    -- DEA Schedule
    dea_schedule VARCHAR(5),
    is_controlled BOOLEAN DEFAULT FALSE,
    
    -- Drug classification
    therapeutic_class TEXT,
    pharmacologic_class TEXT,
    
    -- Vector embedding (768 dimensions for OpenAI ada-002 compatible)
    embedding vector(768),
    
    -- Metadata for AI context
    metadata JSONB,
    
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- HNSW index for efficient similarity search
CREATE INDEX IF NOT EXISTS idx_drug_embedding_hnsw 
ON drug_embeddings USING hnsw (embedding vector_cosine_ops)
WITH (m = 16, ef_construction = 64);

-- Standard indexes
CREATE INDEX IF NOT EXISTS idx_drug_rxnorm ON drug_embeddings (rxnorm_cui);
CREATE INDEX IF NOT EXISTS idx_drug_controlled ON drug_embeddings (is_controlled) WHERE is_controlled = TRUE;
CREATE INDEX IF NOT EXISTS idx_drug_schedule ON drug_embeddings (dea_schedule) WHERE dea_schedule IS NOT NULL;

-- Full-text search on drug names
CREATE INDEX IF NOT EXISTS idx_drug_name_search 
ON drug_embeddings USING GIN (to_tsvector('english', drug_name || ' ' || COALESCE(generic_name, '') || ' ' || COALESCE(brand_name, '')));

-- Insert some sample drugs for demo (without embeddings - those would come from ML pipeline)
INSERT INTO drug_embeddings (ndc_code, rxnorm_cui, drug_name, generic_name, dea_schedule, is_controlled) VALUES
('00069015001', '207106', 'Oxycodone HCL 10mg Tablet', 'Oxycodone', 'C2', TRUE),
('00093317901', '197361', 'Lisinopril 10mg Tablet', 'Lisinopril', NULL, FALSE),
('00591013701', '310965', 'Metformin HCL 500mg Tablet', 'Metformin', NULL, FALSE),
('00093505601', '313782', 'Atorvastatin 20mg Tablet', 'Atorvastatin', NULL, FALSE),
('00093314001', '198211', 'Amoxicillin 500mg Capsule', 'Amoxicillin', NULL, FALSE)
ON CONFLICT (ndc_code) DO NOTHING;

COMMENT ON TABLE drug_embeddings IS 'Drug information with vector embeddings for AI-assisted routing and DDI prediction';
COMMENT ON COLUMN drug_embeddings.embedding IS '768-dimensional vector embedding for similarity search (OpenAI ada-002 compatible)';

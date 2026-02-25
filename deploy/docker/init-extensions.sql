-- Initialize PostgreSQL extensions for prescription engine
-- This runs automatically when PostgreSQL container starts

-- Enable TimescaleDB for time-series event storage
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Enable pgvector for AI embeddings
CREATE EXTENSION IF NOT EXISTS vector;

-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enable cryptographic functions
CREATE EXTENSION IF NOT EXISTS pgcrypto;

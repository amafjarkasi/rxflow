package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/drfirst/go-oec/pkg/ai"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// CDSAnomalyResult represents a clinical decision support semantic match
type CDSAnomalyResult struct {
	OriginalSig   string
	MatchedSig    string
	AnomalyScore  float32 // 0.0 means identical, 1.0 means highly anomalous
	FlaggedReason string
}

// VectorRepository integrates with the pgvector Postgres extension for
// AI-powered semantic drug matching and anomalous prescription detection.
// This is an advanced capability separating RxFlow from basic orchestrators.
type VectorRepository struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewVectorRepository creates a vector repository for high-dimensional embeddings
func NewVectorRepository(pool *pgxpool.Pool, logger *zap.Logger) *VectorRepository {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &VectorRepository{
		pool:   pool,
		logger: logger,
	}
}

// EnsureExtension verifies that pgvector is installed in the database
func (v *VectorRepository) EnsureExtension(ctx context.Context) error {
	_, err := v.pool.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS vector;")
	if err != nil {
		return fmt.Errorf("failed to ensure pgvector extension is installed: %w", err)
	}

	// Ensure the known_safe_sigs table exists for anomaly detection reference
	tableQuery := `
		CREATE TABLE IF NOT EXISTS known_safe_sigs (
			id SERIAL PRIMARY KEY,
			ndc_code VARCHAR(11) NOT NULL,
			sig_text TEXT NOT NULL,
			embedding vector(1536) NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);
		-- HNSW index for ultra-fast Approximate Nearest Neighbor (ANN) search over massive datasets
		CREATE INDEX IF NOT EXISTS idx_safe_sigs_embedding ON known_safe_sigs USING hnsw (embedding vector_cosine_ops);
	`
	if _, err := v.pool.Exec(ctx, tableQuery); err != nil {
		return fmt.Errorf("failed to create vector search table: %w", err)
	}

	v.logger.Info("pgvector extension and HNSW indexing initialized")
	return nil
}

// DetectAnomalousSig performs a high-dimensional nearest neighbor search (ANN)
// against a database of millions of known safe prescription directions for a specific NDC.
// It returns a high anomaly score if the new embedding is semantically distant from safety baselines.
func (v *VectorRepository) DetectAnomalousSig(ctx context.Context, ndc string, currentSig string, queryVector ai.EmbeddingVector) (*CDSAnomalyResult, error) {
	// Construct the pgvector string representation: "[0.1, 0.2, ...]"
	vectorStrs := make([]string, len(queryVector))
	for i, val := range queryVector {
		vectorStrs[i] = fmt.Sprintf("%f", val)
	}
	vectorSQL := fmt.Sprintf("[%s]", strings.Join(vectorStrs, ","))

	// Find the most semantically similar known safe instruction using Cosine Distance (<=>)
	// Lower distance means higher similarity.
	query := `
		SELECT sig_text, (embedding <=> $1::vector) AS cosine_distance
		FROM known_safe_sigs
		WHERE ndc_code = $2
		ORDER BY cosine_distance ASC
		LIMIT 1;
	`

	var bestMatchSig string
	var anomalyScore float32

	err := v.pool.QueryRow(ctx, query, vectorSQL, ndc).Scan(&bestMatchSig, &anomalyScore)
	if err != nil {
		if err.Error() == "no rows in result set" {
			// If no known sigs exist for this NDC, we cannot determine anomaly
			return &CDSAnomalyResult{
				OriginalSig:   currentSig,
				AnomalyScore:  1.0, // High risk, unknown drug
				FlaggedReason: "No known safe directions baseline found for this drug in the CDS database.",
			}, nil
		}
		return nil, fmt.Errorf("failed semantic anomaly detection query: %w", err)
	}

	result := &CDSAnomalyResult{
		OriginalSig:  currentSig,
		MatchedSig:   bestMatchSig,
		AnomalyScore: anomalyScore,
	}

	// Thresholds for semantic deviation (e.g. text embedding ada-002, > 0.15 is highly divergent)
	if anomalyScore > 0.15 {
		result.FlaggedReason = "WARNING: The semantics of these instructions significantly deviate from standard safe clinical practice for this drug."
		v.logger.Warn("CDS anomaly detected",
			zap.String("ndc", ndc),
			zap.Float32("anomaly_score", anomalyScore),
		)
	}

	return result, nil
}

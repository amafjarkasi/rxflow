package clinical

import (
	"context"
	"fmt"

	"github.com/drfirst/go-oec/internal/infrastructure/postgres"
	"github.com/drfirst/go-oec/pkg/ai"
)

// CDSService is a Clinical Decision Support domain service that evaluates
// prescriptions for patient safety before routing them.
type CDSService struct {
	aiMatcher *ai.SemanticMatcher
	vectorDB  *postgres.VectorRepository
}

// NewCDSService creates a robust semantic analysis engine for detecting dangerous
// or anomalous prescribing habits.
func NewCDSService(aiMatcher *ai.SemanticMatcher, vectorDB *postgres.VectorRepository) *CDSService {
	return &CDSService{
		aiMatcher: aiMatcher,
		vectorDB:  vectorDB,
	}
}

// EvaluatePrescriptionSafety examines the raw text of a prescription
// and transforms it into a 1536-dimension NLP vector. It performs a Cosine Similarity
// search over millions of known safe historical prescriptions for this specific drug (NDC).
//
// This allows the system to catch "Take 10 pills daily" when the semantic norm is
// "Take 1 pill daily", saving patient lives.
func (c *CDSService) EvaluatePrescriptionSafety(ctx context.Context, ndc string, sigText string) (*postgres.CDSAnomalyResult, error) {

	// Step 1: LLM text-to-vector embedding
	vector, err := c.aiMatcher.AnalyzeSigInstructions(ctx, sigText)
	if err != nil {
		return nil, fmt.Errorf("failed to generate NLP vector: %w", err)
	}

	// Step 2: ANN pgvector semantic search
	result, err := c.vectorDB.DetectAnomalousSig(ctx, ndc, sigText, vector)
	if err != nil {
		return nil, fmt.Errorf("failed semantic search: %w", err)
	}

	return result, nil
}

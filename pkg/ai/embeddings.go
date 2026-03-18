// Package ai provides advanced artificial intelligence and machine learning capabilities
// for the RxFlow orchestration engine. It specializes in NLP-based semantic matching
// to ensure clinical safety, anomaly detection, and decision support (CDS).
package ai

import (
	"context"
	"fmt"
	"strings"
)

// EmbeddingVector represents a high-dimensional dense vector produced by an NLP model.
// Common models like text-embedding-ada-002 use 1536 dimensions.
type EmbeddingVector []float32

// ModelProvider defines the interface for an external LLM/Embedding provider (e.g. OpenAI, HuggingFace).
type ModelProvider interface {
	// GenerateEmbedding converts plain text (like complex Sig directions or drug names)
	// into a semantic numerical vector.
	GenerateEmbedding(ctx context.Context, text string) (EmbeddingVector, error)
}

// SemanticMatcher evaluates the contextual similarity between clinical texts
type SemanticMatcher struct {
	provider ModelProvider
}

// NewSemanticMatcher initializes an AI matcher for clinical text similarity
func NewSemanticMatcher(provider ModelProvider) *SemanticMatcher {
	return &SemanticMatcher{
		provider: provider,
	}
}

// AnalyzeSigInstructions takes an unstructured prescription Sig (directions)
// and converts it into a vector representation for anomaly detection in the database.
func (sm *SemanticMatcher) AnalyzeSigInstructions(ctx context.Context, sigText string) (EmbeddingVector, error) {
	// Pre-process sig text (normalize whitespace, casing) for consistent embeddings
	cleanSig := strings.TrimSpace(strings.ToLower(sigText))

	if cleanSig == "" {
		return nil, fmt.Errorf("sig text cannot be empty for semantic analysis")
	}

	// Generate embedding from the provider
	vector, err := sm.provider.GenerateEmbedding(ctx, cleanSig)
	if err != nil {
		return nil, fmt.Errorf("failed to generate NLP embedding for sig text: %w", err)
	}

	return vector, nil
}

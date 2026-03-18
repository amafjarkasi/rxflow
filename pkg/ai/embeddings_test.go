package ai

import (
	"context"
	"reflect"
	"testing"
)

// MockProvider is a stubbed out NLP provider returning constant size embeddings
type MockProvider struct{}

func (m *MockProvider) GenerateEmbedding(ctx context.Context, text string) (EmbeddingVector, error) {
	// A simple 3-dimensional mock vector for testing
	return EmbeddingVector{0.1, 0.5, 0.9}, nil
}

func TestSemanticMatcher_AnalyzeSigInstructions(t *testing.T) {
	matcher := NewSemanticMatcher(&MockProvider{})

	sigText := "Take 1 tablet by mouth daily for 30 days"

	vector, err := matcher.AnalyzeSigInstructions(context.Background(), sigText)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := EmbeddingVector{0.1, 0.5, 0.9}
	if !reflect.DeepEqual(vector, expected) {
		t.Errorf("expected %v, got %v", expected, vector)
	}
}

func TestSemanticMatcher_EmptySigText(t *testing.T) {
	matcher := NewSemanticMatcher(&MockProvider{})

	_, err := matcher.AnalyzeSigInstructions(context.Background(), "   ")
	if err == nil {
		t.Fatal("expected error on empty sig text")
	}

	if err.Error() != "sig text cannot be empty for semantic analysis" {
		t.Errorf("unexpected error message: %v", err)
	}
}

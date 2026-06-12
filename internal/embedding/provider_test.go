package embedding

import (
	"context"
	"math"
	"testing"
)

func TestFakeProviderIsDeterministic(t *testing.T) {
	provider := NewFakeProvider("fake-8", 8)
	ctx := context.Background()

	first, err := provider.Embed(ctx, []string{"phone sync phone"})
	if err != nil {
		t.Fatalf("Embed first: %v", err)
	}
	second, err := provider.Embed(ctx, []string{"phone sync phone"})
	if err != nil {
		t.Fatalf("Embed second: %v", err)
	}

	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("batch lengths = %d/%d, want 1/1", len(first), len(second))
	}
	assertVectorEqual(t, first[0], second[0])
}

func TestFakeProviderUsesConfiguredDimension(t *testing.T) {
	provider := NewFakeProvider("fake-16", 16)
	vectors, err := provider.Embed(context.Background(), []string{"recall memory"})
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if got := provider.Name(); got != "fake" {
		t.Fatalf("Name = %q, want fake", got)
	}
	if got := provider.Model(); got != "fake-16" {
		t.Fatalf("Model = %q, want fake-16", got)
	}
	if got := provider.Dimension(); got != 16 {
		t.Fatalf("Dimension = %d, want 16", got)
	}
	if len(vectors[0]) != 16 {
		t.Fatalf("vector dim = %d, want 16", len(vectors[0]))
	}
}

func TestFakeProviderGivesRelatedTextsHigherSimilarity(t *testing.T) {
	provider := NewFakeProvider("fake-64", 64)
	vectors, err := provider.Embed(context.Background(), []string{
		"phone sync setup",
		"iphone sync setup preference",
		"docker deploy container",
	})
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}

	related := cosine(vectors[0], vectors[1])
	unrelated := cosine(vectors[0], vectors[2])
	if related <= unrelated {
		t.Fatalf("related similarity %v <= unrelated similarity %v", related, unrelated)
	}
}

func assertVectorEqual(t *testing.T, got, want []float32) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("vector[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func cosine(a, b []float32) float64 {
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

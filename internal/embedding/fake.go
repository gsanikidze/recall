package embedding

import (
	"context"
	"hash/fnv"
	"math"
	"strings"
	"unicode"
)

type FakeProvider struct {
	model string
	dim   int
}

func NewFakeProvider(model string, dim int) FakeProvider {
	if model == "" {
		model = "fake"
	}
	if dim <= 0 {
		dim = 64
	}
	return FakeProvider{model: model, dim: dim}
}

func (p FakeProvider) Name() string { return "fake" }

func (p FakeProvider) Model() string { return p.model }

func (p FakeProvider) Dimension() int { return p.dim }

func (p FakeProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	vectors := make([][]float32, 0, len(texts))
	for _, text := range texts {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		vectors = append(vectors, p.embedOne(text))
	}
	return vectors, nil
}

func (p FakeProvider) embedOne(text string) []float32 {
	vector := make([]float32, p.dim)
	for _, token := range tokens(text) {
		bucket := int(hashToken(token) % uint64(p.dim))
		vector[bucket]++
	}
	normalize(vector)
	return vector
}

func tokens(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

func hashToken(token string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(token))
	return h.Sum64()
}

func normalize(vector []float32) {
	var norm float64
	for _, value := range vector {
		norm += float64(value * value)
	}
	if norm == 0 {
		return
	}
	scale := float32(1 / math.Sqrt(norm))
	for i := range vector {
		vector[i] *= scale
	}
}

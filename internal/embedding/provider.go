package embedding

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Provider generates embedding vectors for text inputs.
type Provider interface {
	Name() string
	Model() string
	Dimension() int
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// ErrUnknownProvider is returned by NewProvider for unsupported provider names.
var ErrUnknownProvider = errors.New("embedding: unknown provider")

// NewProvider constructs a Provider by name. "ollama" and "fake" are supported.
// model is required; baseURL is used by ollama (empty falls back to the default).
func NewProvider(name, model, baseURL string) (Provider, error) {
	if strings.TrimSpace(model) == "" {
		return nil, fmt.Errorf("embedding: model is required")
	}
	switch strings.TrimSpace(name) {
	case "ollama":
		return NewOllamaProvider(baseURL, model), nil
	case "fake":
		return NewFakeProvider(model, 32), nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownProvider, name)
	}
}

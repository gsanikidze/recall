package embedding

import "context"

type Provider interface {
	Name() string
	Model() string
	Dimension() int
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

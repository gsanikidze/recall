package index

import (
	"context"
	"fmt"

	"recall/internal/index/db"
)

type Embedding struct {
	MemoryID    string
	Provider    string
	Model       string
	Dim         int
	Vector      []float32
	ContentHash string
}

func (ix *Index) UpsertEmbedding(ctx context.Context, e Embedding) error {
	if e.Dim != len(e.Vector) {
		return fmt.Errorf("index: embedding dimension %d does not match vector length %d", e.Dim, len(e.Vector))
	}
	blob, err := encodeVector(e.Vector)
	if err != nil {
		return fmt.Errorf("index: encode embedding vector: %w", err)
	}
	if err := ix.q.UpsertEmbedding(ctx, db.UpsertEmbeddingParams{
		MemoryID:    e.MemoryID,
		Provider:    e.Provider,
		Model:       e.Model,
		Dim:         int64(e.Dim),
		Vector:      blob,
		ContentHash: e.ContentHash,
	}); err != nil {
		return fmt.Errorf("index: upsert embedding: %w", err)
	}
	return nil
}

func (ix *Index) EmbeddingForMemory(ctx context.Context, memoryID, provider, model string) (Embedding, error) {
	row, err := ix.q.GetEmbeddingForMemory(ctx, db.GetEmbeddingForMemoryParams{
		MemoryID: memoryID,
		Provider: provider,
		Model:    model,
	})
	if err != nil {
		return Embedding{}, fmt.Errorf("index: get embedding: %w", err)
	}
	return embeddingFromRow(row)
}

func (ix *Index) Embeddings(ctx context.Context, provider, model string) ([]Embedding, error) {
	rows, err := ix.q.ListEmbeddingsForModel(ctx, db.ListEmbeddingsForModelParams{
		Provider: provider,
		Model:    model,
	})
	if err != nil {
		return nil, fmt.Errorf("index: list embeddings: %w", err)
	}
	out := make([]Embedding, 0, len(rows))
	for _, row := range rows {
		e, err := embeddingFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

func embeddingFromRow(row db.MemoryEmbedding) (Embedding, error) {
	dim := int(row.Dim)
	vector, err := decodeVector(row.Vector, dim)
	if err != nil {
		return Embedding{}, fmt.Errorf("index: decode embedding vector: %w", err)
	}
	return Embedding{
		MemoryID:    row.MemoryID,
		Provider:    row.Provider,
		Model:       row.Model,
		Dim:         dim,
		Vector:      vector,
		ContentHash: row.ContentHash,
	}, nil
}

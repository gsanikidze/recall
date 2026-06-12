package recall

import (
	"context"
	"fmt"
	"sort"

	"recall/internal/embedding"
	"recall/internal/index"
)

type EmbedStats struct {
	Embedded int `json:"embedded"`
	Skipped  int `json:"skipped"`
	Failed   int `json:"failed"`
}

func (e *Engine) EmbedAll(ctx context.Context, provider embedding.Provider, force bool) (EmbedStats, error) {
	ids, err := e.index.ListIDs(ctx)
	if err != nil {
		return EmbedStats{}, err
	}
	sort.Strings(ids)

	existingRows, err := e.index.Embeddings(ctx, provider.Name(), provider.Model())
	if err != nil {
		return EmbedStats{}, err
	}
	existingByID := make(map[string]index.Embedding, len(existingRows))
	for _, row := range existingRows {
		existingByID[row.MemoryID] = row
	}

	stats := EmbedStats{}
	for _, id := range ids {
		m, _, err := e.Get(ctx, id)
		if err != nil {
			stats.Failed++
			return stats, err
		}
		hash := EmbedContentHash(m)
		if !force {
			if existing, ok := existingByID[id]; ok && existing.ContentHash == hash {
				stats.Skipped++
				continue
			}
		}

		vectors, err := provider.Embed(ctx, []string{EmbedText(m)})
		if err != nil {
			stats.Failed++
			return stats, err
		}
		if len(vectors) != 1 {
			stats.Failed++
			return stats, fmt.Errorf("recall: embedding provider returned %d vectors for one memory", len(vectors))
		}
		vector := vectors[0]
		if len(vector) == 0 {
			stats.Failed++
			return stats, fmt.Errorf("recall: embedding provider returned empty vector")
		}
		if dim := provider.Dimension(); dim > 0 && len(vector) != dim {
			stats.Failed++
			return stats, fmt.Errorf("recall: embedding dimension %d does not match provider dimension %d", len(vector), dim)
		}

		if err := e.index.UpsertEmbedding(ctx, index.Embedding{
			MemoryID:    id,
			Provider:    provider.Name(),
			Model:       provider.Model(),
			Dim:         len(vector),
			Vector:      vector,
			ContentHash: hash,
		}); err != nil {
			stats.Failed++
			return stats, err
		}
		stats.Embedded++
	}
	return stats, nil
}

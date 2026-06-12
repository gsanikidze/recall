package index

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"recall/internal/memory"
)

// ErrInvalidFilter marks malformed user-supplied search filters.
var ErrInvalidFilter = errors.New("index: invalid search filter")

// defaultLimit caps result count when a caller does not specify one.
const defaultLimit = 20

// MaxLimit is the hard upper bound for one search call.
const MaxLimit = 200

// recencyPenalty sinks stale memories: each day since a memory was updated adds
// this much to its (lower-is-better) score, so old items drift down without
// being removed. The agent still decides final relevance.
const recencyPenalty = 0.01

// importanceBoost lifts high-importance memories in the lower-is-better score.
const importanceBoost = 0.25

type SearchMode string

const (
	SearchModeKeyword  SearchMode = "keyword"
	SearchModeSemantic SearchMode = "semantic"
	SearchModeHybrid   SearchMode = "hybrid"
)

// Filter describes a search. All fields are optional and combine with AND.
type Filter struct {
	Query          string     // full-text query over title+body; empty = browse by filters
	Domain         string     // restrict to one domain
	Tags           []string   // match any of these tags
	Project        string     // restrict to one project
	Lifecycle      string     // "evergreen" | "expires"
	Since          string     // updated >= this date (YYYY-MM-DD)
	Until          string     // updated <= this date (YYYY-MM-DD)
	IncludeExpired bool       // include memories past their expires_on
	Limit          int        // max hits (defaults to defaultLimit)
	Mode           SearchMode // keyword (default), semantic, or hybrid
	QueryVector    []float32  // semantic query vector
	Provider       string     // embedding provider for semantic/hybrid
	Model          string     // embedding model for semantic/hybrid
}

// Hit is a lightweight search result. Callers fetch full content separately by
// reading the memory file at Path.
type Hit struct {
	ID            string  `json:"id"`
	Title         string  `json:"title"`
	Snippet       string  `json:"snippet"`
	Path          string  `json:"path"`
	Domain        string  `json:"domain"`
	Importance    int     `json:"importance"`
	Score         float64 `json:"score"` // lower is better
	KeywordScore  float64 `json:"keyword_score,omitempty"`
	SemanticScore float64 `json:"semantic_score,omitempty"`
}

// Search runs a filtered, ranked query. With a Query it uses FTS5 (ranked by
// relevance blended with recency); without one it browses by filters, newest
// first.
func (ix *Index) Search(ctx context.Context, f Filter) ([]Hit, error) {
	if err := validateFilter(f); err != nil {
		return nil, err
	}
	if f.Mode == SearchModeSemantic {
		return ix.semanticSearch(ctx, f)
	}
	if f.Mode == SearchModeHybrid {
		return ix.hybridSearch(ctx, f)
	}
	return ix.keywordSearch(ctx, f)
}

func (ix *Index) keywordSearch(ctx context.Context, f Filter) ([]Hit, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}

	var (
		b    strings.Builder
		args []any
	)
	fts := strings.TrimSpace(f.Query) != ""

	if fts {
		matchQuery := sanitizeFTSQuery(f.Query)
		if matchQuery == "" {
			fts = false
		} else {
			b.WriteString(`SELECT m.id, m.title, m.path, m.domain, m.importance,
       snippet(memories_fts, 2, '[', ']', ' … ', 12) AS snippet,
       bm25(memories_fts) AS rank
FROM memories_fts f JOIN memories m ON m.id = f.id
WHERE memories_fts MATCH ?`)
			args = append(args, matchQuery)
		}
	}
	if !fts {
		b.WriteString(`SELECT m.id, m.title, m.path, m.domain, m.importance,
       substr(m.body, 1, 160) AS snippet,
       0.0 AS rank
FROM memories m
WHERE 1 = 1`)
	}

	if f.Domain != "" {
		b.WriteString(" AND m.domain = ?")
		args = append(args, f.Domain)
	}
	if f.Project != "" {
		b.WriteString(" AND m.project = ?")
		args = append(args, f.Project)
	}
	if f.Lifecycle != "" {
		b.WriteString(" AND m.lifecycle = ?")
		args = append(args, f.Lifecycle)
	}
	if f.Since != "" {
		b.WriteString(" AND m.updated >= ?")
		args = append(args, f.Since)
	}
	if f.Until != "" {
		b.WriteString(" AND m.updated <= ?")
		args = append(args, f.Until)
	}
	if tags := dedupe(f.Tags); len(tags) > 0 {
		b.WriteString(" AND m.id IN (SELECT memory_id FROM tags WHERE tag IN (")
		b.WriteString(placeholders(len(tags)))
		b.WriteString("))")
		for _, t := range tags {
			args = append(args, t)
		}
	}
	if !f.IncludeExpired {
		b.WriteString(" AND NOT (m.lifecycle = 'expires' AND m.expires_on != '' AND m.expires_on < date('now'))")
	}

	if fts {
		b.WriteString(" ORDER BY bm25(memories_fts) + (julianday('now') - julianday(m.updated)) * ? - ((m.importance - 3) * ?) ASC")
		args = append(args, recencyPenalty, importanceBoost)
	} else {
		b.WriteString(" ORDER BY m.importance DESC, m.updated DESC")
	}
	b.WriteString(" LIMIT ?")
	args = append(args, limit)

	rows, err := ix.sql.QueryContext(ctx, b.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("index: search query: %w", err)
	}
	defer rows.Close()

	var hits []Hit
	for rows.Next() {
		var h Hit
		if err := rows.Scan(&h.ID, &h.Title, &h.Path, &h.Domain, &h.Importance, &h.Snippet, &h.Score); err != nil {
			return nil, fmt.Errorf("index: scan hit: %w", err)
		}
		h.Snippet = strings.TrimSpace(h.Snippet)
		hits = append(hits, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("index: search rows: %w", err)
	}
	return hits, nil
}

func (ix *Index) hybridSearch(ctx context.Context, f Filter) ([]Hit, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	candidateLimit := limit * 5
	if candidateLimit < 50 {
		candidateLimit = 50
	}
	if candidateLimit > MaxLimit {
		candidateLimit = MaxLimit
	}

	keywordFilter := f
	keywordFilter.Mode = SearchModeKeyword
	keywordFilter.QueryVector = nil
	keywordFilter.Provider = ""
	keywordFilter.Model = ""
	keywordFilter.Limit = candidateLimit
	keywordHits, err := ix.keywordSearch(ctx, keywordFilter)
	if err != nil {
		return nil, err
	}

	semanticFilter := f
	semanticFilter.Mode = SearchModeSemantic
	semanticFilter.Limit = candidateLimit
	semanticHits, err := ix.semanticSearch(ctx, semanticFilter)
	if err != nil {
		return nil, err
	}

	merged := make(map[string]Hit, len(keywordHits)+len(semanticHits))
	keywordDenom := float64(len(keywordHits) - 1)
	if keywordDenom < 1 {
		keywordDenom = 1
	}
	for rank, hit := range keywordHits {
		hit.KeywordScore = 1 - (float64(rank) / keywordDenom)
		merged[hit.ID] = hit
	}
	for _, hit := range semanticHits {
		existing, ok := merged[hit.ID]
		if ok {
			existing.SemanticScore = hit.SemanticScore
			if existing.Snippet == "" {
				existing.Snippet = hit.Snippet
			}
			merged[hit.ID] = existing
			continue
		}
		merged[hit.ID] = hit
	}

	hits := make([]Hit, 0, len(merged))
	for _, hit := range merged {
		importanceNorm := float64(hit.Importance-1) / 4.0
		hit.Score = -(0.60 * hit.KeywordScore) - (0.40 * hit.SemanticScore) - (0.05 * importanceNorm)
		hits = append(hits, hit)
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Score == hits[j].Score {
			return hits[i].ID < hits[j].ID
		}
		return hits[i].Score < hits[j].Score
	})
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits, nil
}

func (ix *Index) semanticSearch(ctx context.Context, f Filter) ([]Hit, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}

	candidates, err := ix.filteredCandidates(ctx, f)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	embeddings, err := ix.Embeddings(ctx, f.Provider, f.Model)
	if err != nil {
		return nil, err
	}
	embeddingsByID := make(map[string]Embedding, len(embeddings))
	for _, e := range embeddings {
		embeddingsByID[e.MemoryID] = e
	}

	hits := make([]Hit, 0, len(candidates))
	for _, candidate := range candidates {
		emb, ok := embeddingsByID[candidate.ID]
		if !ok {
			continue
		}
		similarity, err := cosineSimilarity(f.QueryVector, emb.Vector)
		if err != nil {
			return nil, fmt.Errorf("index: semantic similarity for %s: %w", candidate.ID, err)
		}
		candidate.SemanticScore = similarity
		candidate.Score = (1 - similarity) - (float64(candidate.Importance-3) * 0.03)
		hits = append(hits, candidate)
	}

	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Score == hits[j].Score {
			return hits[i].ID < hits[j].ID
		}
		return hits[i].Score < hits[j].Score
	})
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits, nil
}

func (ix *Index) filteredCandidates(ctx context.Context, f Filter) ([]Hit, error) {
	var b strings.Builder
	args := []any{}
	b.WriteString(`SELECT m.id, m.title, m.path, m.domain, m.importance,
       substr(m.body, 1, 160) AS snippet
FROM memories m
WHERE 1 = 1`)
	appendStructuredFilters(&b, &args, f)
	b.WriteString(" ORDER BY m.importance DESC, m.updated DESC")

	rows, err := ix.sql.QueryContext(ctx, b.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("index: candidate query: %w", err)
	}
	defer rows.Close()

	var hits []Hit
	for rows.Next() {
		var h Hit
		if err := rows.Scan(&h.ID, &h.Title, &h.Path, &h.Domain, &h.Importance, &h.Snippet); err != nil {
			return nil, fmt.Errorf("index: scan candidate: %w", err)
		}
		h.Snippet = strings.TrimSpace(h.Snippet)
		hits = append(hits, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("index: candidate rows: %w", err)
	}
	return hits, nil
}

func appendStructuredFilters(b *strings.Builder, args *[]any, f Filter) {
	if f.Domain != "" {
		b.WriteString(" AND m.domain = ?")
		*args = append(*args, f.Domain)
	}
	if f.Project != "" {
		b.WriteString(" AND m.project = ?")
		*args = append(*args, f.Project)
	}
	if f.Lifecycle != "" {
		b.WriteString(" AND m.lifecycle = ?")
		*args = append(*args, f.Lifecycle)
	}
	if f.Since != "" {
		b.WriteString(" AND m.updated >= ?")
		*args = append(*args, f.Since)
	}
	if f.Until != "" {
		b.WriteString(" AND m.updated <= ?")
		*args = append(*args, f.Until)
	}
	if tags := dedupe(f.Tags); len(tags) > 0 {
		b.WriteString(" AND m.id IN (SELECT memory_id FROM tags WHERE tag IN (")
		b.WriteString(placeholders(len(tags)))
		b.WriteString("))")
		for _, t := range tags {
			*args = append(*args, t)
		}
	}
	if !f.IncludeExpired {
		b.WriteString(" AND NOT (m.lifecycle = 'expires' AND m.expires_on != '' AND m.expires_on < date('now'))")
	}
}

func validateFilter(f Filter) error {
	if f.Mode != "" && f.Mode != SearchModeKeyword && f.Mode != SearchModeSemantic && f.Mode != SearchModeHybrid {
		return fmt.Errorf("%w: mode must be 'keyword', 'semantic', or 'hybrid', got %q", ErrInvalidFilter, f.Mode)
	}
	if f.Mode == SearchModeSemantic || f.Mode == SearchModeHybrid {
		if len(f.QueryVector) == 0 {
			return fmt.Errorf("%w: semantic search requires query vector", ErrInvalidFilter)
		}
		if strings.TrimSpace(f.Provider) == "" {
			return fmt.Errorf("%w: semantic search requires provider", ErrInvalidFilter)
		}
		if strings.TrimSpace(f.Model) == "" {
			return fmt.Errorf("%w: semantic search requires model", ErrInvalidFilter)
		}
	}
	if f.Lifecycle != "" && f.Lifecycle != string(memory.Evergreen) && f.Lifecycle != string(memory.Expires) {
		return fmt.Errorf("%w: lifecycle must be 'evergreen' or 'expires', got %q", ErrInvalidFilter, f.Lifecycle)
	}
	if f.Since != "" {
		if _, err := memory.ParseDate(f.Since); err != nil {
			return fmt.Errorf("%w: invalid since date: %v", ErrInvalidFilter, err)
		}
	}
	if f.Until != "" {
		if _, err := memory.ParseDate(f.Until); err != nil {
			return fmt.Errorf("%w: invalid until date: %v", ErrInvalidFilter, err)
		}
	}
	if f.Since != "" && f.Until != "" && f.Since > f.Until {
		return fmt.Errorf("%w: since must be before or equal to until", ErrInvalidFilter)
	}
	return nil
}

// placeholders returns "?, ?, ..." with n entries.
func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?, ", n), ", ")
}

var ftsToken = regexp.MustCompile(`[\p{L}\p{N}_]+`)

func sanitizeFTSQuery(q string) string {
	tokens := ftsToken.FindAllString(strings.ToLower(q), -1)
	if len(tokens) == 0 {
		return ""
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		if seen[tok] {
			continue
		}
		seen[tok] = true
		out = append(out, strconvQuote(tok))
	}
	return strings.Join(out, " OR ")
}

func strconvQuote(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

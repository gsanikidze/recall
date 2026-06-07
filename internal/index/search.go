package index

import (
	"context"
	"fmt"
	"strings"
)

// defaultLimit caps result count when a caller does not specify one.
const defaultLimit = 20

// recencyPenalty sinks stale memories: each day since a memory was updated adds
// this much to its (lower-is-better) score, so old items drift down without
// being removed. The agent still decides final relevance.
const recencyPenalty = 0.01

// Filter describes a search. All fields are optional and combine with AND.
type Filter struct {
	Query          string   // full-text query over title+body; empty = browse by filters
	Domain         string   // restrict to one domain
	Tags           []string // match any of these tags
	Project        string   // restrict to one project
	Lifecycle      string   // "evergreen" | "expires"
	Since          string   // updated >= this date (YYYY-MM-DD)
	Until          string   // updated <= this date (YYYY-MM-DD)
	IncludeExpired bool     // include memories past their expires_on
	Limit          int      // max hits (defaults to defaultLimit)
}

// Hit is a lightweight search result. Callers fetch full content separately by
// reading the memory file at Path.
type Hit struct {
	ID      string
	Title   string
	Snippet string
	Path    string
	Domain  string
	Score   float64 // lower is better
}

// Search runs a filtered, ranked query. With a Query it uses FTS5 (ranked by
// relevance blended with recency); without one it browses by filters, newest
// first.
func (ix *Index) Search(ctx context.Context, f Filter) ([]Hit, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = defaultLimit
	}

	var (
		b    strings.Builder
		args []any
	)
	fts := strings.TrimSpace(f.Query) != ""

	if fts {
		b.WriteString(`SELECT m.id, m.title, m.path, m.domain,
       snippet(memories_fts, 2, '[', ']', ' … ', 12) AS snippet,
       bm25(memories_fts) AS rank
FROM memories_fts f JOIN memories m ON m.id = f.id
WHERE memories_fts MATCH ?`)
		args = append(args, f.Query)
	} else {
		b.WriteString(`SELECT m.id, m.title, m.path, m.domain,
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
		b.WriteString(" ORDER BY bm25(memories_fts) + (julianday('now') - julianday(m.updated)) * ? ASC")
		args = append(args, recencyPenalty)
	} else {
		b.WriteString(" ORDER BY m.updated DESC")
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
		if err := rows.Scan(&h.ID, &h.Title, &h.Path, &h.Domain, &h.Snippet, &h.Score); err != nil {
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

// placeholders returns "?, ?, ..." with n entries.
func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?, ", n), ", ")
}

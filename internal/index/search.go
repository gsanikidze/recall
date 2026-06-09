package index

import (
	"context"
	"errors"
	"fmt"
	"regexp"
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
	if err := validateFilter(f); err != nil {
		return nil, err
	}
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
			b.WriteString(`SELECT m.id, m.title, m.path, m.domain,
       snippet(memories_fts, 2, '[', ']', ' … ', 12) AS snippet,
       bm25(memories_fts) AS rank
FROM memories_fts f JOIN memories m ON m.id = f.id
WHERE memories_fts MATCH ?`)
			args = append(args, matchQuery)
		}
	}
	if !fts {
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

func validateFilter(f Filter) error {
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

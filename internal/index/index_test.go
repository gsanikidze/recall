package index

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"recall/internal/memory"
)

func openIndex(t *testing.T) *Index {
	t.Helper()
	ix, err := Open(filepath.Join(t.TempDir(), "recall.sqlite"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = ix.Close() })
	return ix
}

func mem(t *testing.T, id, title, domain, body string) memory.Memory {
	t.Helper()
	d, _ := memory.ParseDate("2026-06-07")
	return memory.Memory{
		ID:        id,
		Title:     title,
		Domain:    domain,
		Created:   d,
		Updated:   d,
		Lifecycle: memory.Evergreen,
		Body:      body,
	}
}

func TestUpsertAndFullTextSearch(t *testing.T) {
	ix := openIndex(t)
	ctx := context.Background()

	m := mem(t, "01AAA", "Kamal deploy", "tools", "Production deploys run through **Kamal**, not Compose.")
	m.Tags = []string{"deploy", "infra"}
	if err := ix.Upsert(ctx, "tools/kamal.md", m); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	hits, err := ix.Search(ctx, Filter{Query: "kamal"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("got %d hits, want 1: %+v", len(hits), hits)
	}
	if hits[0].ID != "01AAA" || hits[0].Path != "tools/kamal.md" {
		t.Errorf("hit = %+v", hits[0])
	}
	// Body is stored stripped: no ** emphasis markers.
	row, _ := ix.q.GetMemory(ctx, "01AAA")
	if want := "Production deploys run through Kamal, not Compose."; row.Body != want {
		t.Errorf("stored body = %q, want %q", row.Body, want)
	}
}

func TestSearchSanitizesFTSOperators(t *testing.T) {
	ix := openIndex(t)
	ctx := context.Background()
	if err := ix.Upsert(ctx, "tools/smoke.md", mem(t, "01FTS", "Smoke memory", "tools", "Smoke memory text")); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	hits, err := ix.Search(ctx, Filter{Query: "Smoke memory --domain tools"})
	if err != nil {
		t.Fatalf("Search with hyphen-like user text should not error: %v", err)
	}
	if len(hits) != 1 || hits[0].ID != "01FTS" {
		t.Fatalf("hits = %+v", hits)
	}
}

func TestSearchLimitDefaultsAndClamps(t *testing.T) {
	ix := openIndex(t)
	ctx := context.Background()
	for i := 0; i < MaxLimit+5; i++ {
		m := mem(t, "01LIM"+string(rune('A'+i/26))+string(rune('A'+i%26)), "Limit memory", "tools", "same body")
		if err := ix.Upsert(ctx, m.ID+".md", m); err != nil {
			t.Fatalf("Upsert %d: %v", i, err)
		}
	}

	hits, err := ix.Search(ctx, Filter{Limit: 0})
	if err != nil {
		t.Fatalf("Search default: %v", err)
	}
	if len(hits) != defaultLimit {
		t.Fatalf("default limit returned %d hits, want %d", len(hits), defaultLimit)
	}

	hits, err = ix.Search(ctx, Filter{Limit: MaxLimit + 1000})
	if err != nil {
		t.Fatalf("Search clamped: %v", err)
	}
	if len(hits) != MaxLimit {
		t.Fatalf("clamped limit returned %d hits, want %d", len(hits), MaxLimit)
	}
}

func TestSearchFilters(t *testing.T) {
	ix := openIndex(t)
	ctx := context.Background()

	a := mem(t, "01A", "Deploy with Kamal", "tools", "kamal deploy infra")
	a.Tags = []string{"deploy"}
	b := mem(t, "01B", "Hiring plan", "people", "kamal is a person we deploy ideas with")
	b.Project = "acme"
	for _, m := range []memory.Memory{a, b} {
		if err := ix.Upsert(ctx, m.Domain+"/"+m.ID+".md", m); err != nil {
			t.Fatalf("Upsert %s: %v", m.ID, err)
		}
	}

	cases := []struct {
		name   string
		filter Filter
		wantID string
		wantN  int
	}{
		{"domain", Filter{Query: "deploy", Domain: "tools"}, "01A", 1},
		{"tag", Filter{Query: "deploy", Tags: []string{"deploy"}}, "01A", 1},
		{"project", Filter{Query: "deploy", Project: "acme"}, "01B", 1},
		{"browse all", Filter{}, "", 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hits, err := ix.Search(ctx, tc.filter)
			if err != nil {
				t.Fatalf("Search: %v", err)
			}
			if len(hits) != tc.wantN {
				t.Fatalf("got %d hits, want %d: %+v", len(hits), tc.wantN, hits)
			}
			if tc.wantID != "" && hits[0].ID != tc.wantID {
				t.Errorf("got id %q, want %q", hits[0].ID, tc.wantID)
			}
		})
	}
}

func TestSearchRejectsInvalidFilters(t *testing.T) {
	ix := openIndex(t)
	ctx := context.Background()

	cases := []struct {
		name    string
		filter  Filter
		wantErr string
	}{
		{"invalid lifecycle", Filter{Lifecycle: "temporary"}, "lifecycle"},
		{"invalid since", Filter{Since: "2026/06/07"}, "since"},
		{"invalid until", Filter{Until: "tomorrow"}, "until"},
		{"since after until", Filter{Since: "2026-06-08", Until: "2026-06-07"}, "since must be before or equal to until"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ix.Search(ctx, tc.filter)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !contains(err.Error(), tc.wantErr) {
				t.Fatalf("error %q missing %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestExpiredHiddenByDefault(t *testing.T) {
	ix := openIndex(t)
	ctx := context.Background()

	past, _ := memory.ParseDate("2020-01-01")
	m := mem(t, "01EXP", "Old freeze", "decisions", "deploy freeze over")
	m.Lifecycle = memory.Expires
	m.ExpiresOn = past
	if err := ix.Upsert(ctx, "decisions/freeze.md", m); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	if hits, _ := ix.Search(ctx, Filter{Query: "freeze"}); len(hits) != 0 {
		t.Errorf("expired memory should be hidden, got %+v", hits)
	}
	if hits, _ := ix.Search(ctx, Filter{Query: "freeze", IncludeExpired: true}); len(hits) != 1 {
		t.Errorf("expected 1 hit with IncludeExpired, got %+v", hits)
	}
}

func TestUpsertReplacesTagsAndDelete(t *testing.T) {
	ix := openIndex(t)
	ctx := context.Background()

	m := mem(t, "01R", "Repo facts", "tools", "go project")
	m.Tags = []string{"go", "old"}
	if err := ix.Upsert(ctx, "tools/repo.md", m); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	// Re-upsert with different tags; old tag must not linger.
	m.Tags = []string{"go"}
	if err := ix.Upsert(ctx, "tools/repo.md", m); err != nil {
		t.Fatalf("re-Upsert: %v", err)
	}
	if hits, _ := ix.Search(ctx, Filter{Tags: []string{"old"}}); len(hits) != 0 {
		t.Errorf("stale tag 'old' still matches: %+v", hits)
	}

	if err := ix.Delete(ctx, "01R"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if hits, _ := ix.Search(ctx, Filter{Query: "go"}); len(hits) != 0 {
		t.Errorf("deleted memory still searchable: %+v", hits)
	}
	ids, _ := ix.ListIDs(ctx)
	if len(ids) != 0 {
		t.Errorf("ListIDs = %v, want empty", ids)
	}
}

func TestReopenPersistsAndMigratesOnce(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "recall.sqlite")
	ctx := context.Background()

	ix1, err := Open(path)
	if err != nil {
		t.Fatalf("Open 1: %v", err)
	}
	if err := ix1.Upsert(ctx, "tools/x.md", mem(t, "01P", "Persisted", "tools", "stays")); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	_ = ix1.Close()

	ix2, err := Open(path) // re-running migrations must be a no-op
	if err != nil {
		t.Fatalf("Open 2: %v", err)
	}
	defer ix2.Close()
	if hits, _ := ix2.Search(ctx, Filter{Query: "stays"}); len(hits) != 1 {
		t.Errorf("expected persisted memory, got %+v", hits)
	}
}

func TestOpenConfiguresSQLitePragmas(t *testing.T) {
	ix := openIndex(t)
	var busyTimeout int
	if err := ix.sql.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		t.Fatalf("busy_timeout pragma: %v", err)
	}
	if busyTimeout < 5000 {
		t.Fatalf("busy_timeout = %d, want at least 5000", busyTimeout)
	}

	var journalMode string
	if err := ix.sql.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("journal_mode pragma: %v", err)
	}
	if journalMode != "wal" {
		t.Fatalf("journal_mode = %q, want wal", journalMode)
	}

	var foreignKeys int
	if err := ix.sql.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		t.Fatalf("foreign_keys pragma: %v", err)
	}
	if foreignKeys != 1 {
		t.Fatalf("foreign_keys = %d, want 1", foreignKeys)
	}
}

func TestConcurrentUpserts(t *testing.T) {
	ix := openIndex(t)
	ctx := context.Background()
	var wg sync.WaitGroup
	errCh := make(chan error, 25)
	for i := 0; i < 25; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := fmt.Sprintf("01CON%03d", i)
			if err := ix.Upsert(ctx, id+".md", mem(t, id, "Concurrent", "tools", "body")); err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatalf("concurrent Upsert: %v", err)
	}
	ids, err := ix.ListIDs(ctx)
	if err != nil {
		t.Fatalf("ListIDs: %v", err)
	}
	if len(ids) != 25 {
		t.Fatalf("ListIDs returned %d ids, want 25", len(ids))
	}
}

func TestStripMarkdown(t *testing.T) {
	in := "# Heading\n\nUse `kamal` and [the docs](http://x). **Bold** and [[wiki]].\n\n- item one\n- item two"
	got := StripMarkdown(in)
	for _, banned := range []string{"#", "`", "**", "](", "[[", "- "} {
		if contains(got, banned) {
			t.Errorf("stripped text still contains %q:\n%s", banned, got)
		}
	}
	for _, want := range []string{"Heading", "kamal", "the docs", "Bold", "wiki", "item one"} {
		if !contains(got, want) {
			t.Errorf("stripped text missing %q:\n%s", want, got)
		}
	}
}

func contains(s, sub string) bool {
	return len(sub) > 0 && len(s) >= len(sub) && (indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

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

func TestSemanticSearchReturnsNearestEmbeddingWithoutKeywordMatch(t *testing.T) {
	ix := openIndex(t)
	ctx := context.Background()

	phone := mem(t, "01PHONE", "Phone setup", "tools", "iPhone sync preference")
	docker := mem(t, "01DOCKER", "Docker deploy", "tools", "container deployment")
	for _, item := range []struct {
		path string
		m    memory.Memory
	}{
		{"tools/phone.md", phone},
		{"tools/docker.md", docker},
	} {
		if err := ix.Upsert(ctx, item.path, item.m); err != nil {
			t.Fatalf("Upsert %s: %v", item.m.ID, err)
		}
	}
	if err := ix.UpsertEmbedding(ctx, Embedding{MemoryID: phone.ID, Provider: "fake", Model: "fake", Dim: 2, Vector: []float32{1, 0}, ContentHash: "phone"}); err != nil {
		t.Fatalf("phone embedding: %v", err)
	}
	if err := ix.UpsertEmbedding(ctx, Embedding{MemoryID: docker.ID, Provider: "fake", Model: "fake", Dim: 2, Vector: []float32{0, 1}, ContentHash: "docker"}); err != nil {
		t.Fatalf("docker embedding: %v", err)
	}

	hits, err := ix.Search(ctx, Filter{Mode: SearchModeSemantic, Query: "unmatched words", QueryVector: []float32{0.9, 0.1}, Provider: "fake", Model: "fake", Limit: 2})
	if err != nil {
		t.Fatalf("Search semantic: %v", err)
	}
	if len(hits) != 2 || hits[0].ID != phone.ID {
		t.Fatalf("semantic hits = %+v, want phone first", hits)
	}
	if hits[0].SemanticScore <= hits[1].SemanticScore {
		t.Fatalf("semantic scores = %+v, want higher similarity on first hit", hits)
	}
}

func TestSemanticSearchRespectsDomainAndLimit(t *testing.T) {
	ix := openIndex(t)
	ctx := context.Background()

	tool := mem(t, "01TOOL", "Tool memory", "tools", "tool body")
	project := mem(t, "01PROJECT", "Project memory", "projects", "project body")
	if err := ix.Upsert(ctx, "tools/tool.md", tool); err != nil {
		t.Fatalf("Upsert tool: %v", err)
	}
	if err := ix.Upsert(ctx, "projects/project.md", project); err != nil {
		t.Fatalf("Upsert project: %v", err)
	}
	for _, id := range []string{tool.ID, project.ID} {
		if err := ix.UpsertEmbedding(ctx, Embedding{MemoryID: id, Provider: "fake", Model: "fake", Dim: 2, Vector: []float32{1, 0}, ContentHash: id}); err != nil {
			t.Fatalf("embedding %s: %v", id, err)
		}
	}

	hits, err := ix.Search(ctx, Filter{Mode: SearchModeSemantic, QueryVector: []float32{1, 0}, Provider: "fake", Model: "fake", Domain: "tools", Limit: 1})
	if err != nil {
		t.Fatalf("Search semantic: %v", err)
	}
	if len(hits) != 1 || hits[0].ID != tool.ID {
		t.Fatalf("semantic filtered hits = %+v, want only tool", hits)
	}
}

func TestSemanticSearchValidatesRequiredVectorProviderAndModel(t *testing.T) {
	ix := openIndex(t)
	ctx := context.Background()

	cases := []Filter{
		{Mode: SearchModeSemantic, Provider: "fake", Model: "fake"},
		{Mode: SearchModeSemantic, QueryVector: []float32{1}, Model: "fake"},
		{Mode: SearchModeSemantic, QueryVector: []float32{1}, Provider: "fake"},
	}
	for _, f := range cases {
		if _, err := ix.Search(ctx, f); err == nil {
			t.Fatalf("Search(%+v) succeeded, want validation error", f)
		}
	}
}

func TestEmbeddingStorageRoundTrip(t *testing.T) {
	ix := openIndex(t)
	ctx := context.Background()

	m := mem(t, "01EMB", "Vector memory", "tools", "embedding body")
	if err := ix.Upsert(ctx, "tools/vector.md", m); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	want := Embedding{
		MemoryID:    "01EMB",
		Provider:    "fake",
		Model:       "fake-8",
		Dim:         3,
		Vector:      []float32{0.25, 0.5, 1},
		ContentHash: "hash-1",
	}
	if err := ix.UpsertEmbedding(ctx, want); err != nil {
		t.Fatalf("UpsertEmbedding: %v", err)
	}

	got, err := ix.EmbeddingForMemory(ctx, "01EMB", "fake", "fake-8")
	if err != nil {
		t.Fatalf("EmbeddingForMemory: %v", err)
	}
	assertEmbeddingEqual(t, got, want)

	all, err := ix.Embeddings(ctx, "fake", "fake-8")
	if err != nil {
		t.Fatalf("Embeddings: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("got %d embeddings, want 1", len(all))
	}
	assertEmbeddingEqual(t, all[0], want)
}

func TestEmbeddingDeletedWithMemory(t *testing.T) {
	ix := openIndex(t)
	ctx := context.Background()

	m := mem(t, "01DEL", "Deleted vector memory", "tools", "embedding body")
	if err := ix.Upsert(ctx, "tools/deleted-vector.md", m); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if err := ix.UpsertEmbedding(ctx, Embedding{
		MemoryID: "01DEL", Provider: "fake", Model: "fake-8", Dim: 2,
		Vector: []float32{1, 0}, ContentHash: "hash-2",
	}); err != nil {
		t.Fatalf("UpsertEmbedding: %v", err)
	}
	if err := ix.Delete(ctx, "01DEL"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := ix.EmbeddingForMemory(ctx, "01DEL", "fake", "fake-8"); err == nil {
		t.Fatal("EmbeddingForMemory succeeded after memory delete")
	}
}

func assertEmbeddingEqual(t *testing.T, got, want Embedding) {
	t.Helper()
	if got.MemoryID != want.MemoryID || got.Provider != want.Provider || got.Model != want.Model || got.Dim != want.Dim || got.ContentHash != want.ContentHash {
		t.Fatalf("embedding metadata = %+v, want %+v", got, want)
	}
	if len(got.Vector) != len(want.Vector) {
		t.Fatalf("vector len = %d, want %d", len(got.Vector), len(want.Vector))
	}
	for i := range want.Vector {
		if got.Vector[i] != want.Vector[i] {
			t.Fatalf("vector[%d] = %v, want %v", i, got.Vector[i], want.Vector[i])
		}
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

func TestSearchReturnsImportance(t *testing.T) {
	ix := openIndex(t)
	ctx := context.Background()

	m := mem(t, "01IMP", "Critical Recall config", "tools", "recall config path")
	m.Importance = 5
	if err := ix.Upsert(ctx, "tools/critical.md", m); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	hits, err := ix.Search(ctx, Filter{Query: "recall config"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("got %d hits, want 1: %+v", len(hits), hits)
	}
	if hits[0].Importance != 5 {
		t.Fatalf("importance = %d, want 5", hits[0].Importance)
	}
}

func TestSearchImportanceRanking(t *testing.T) {
	ix := openIndex(t)
	ctx := context.Background()

	low := mem(t, "01LOW", "Recall config low", "tools", "same ranking text")
	low.Importance = 1
	high := mem(t, "01HIGH", "Recall config high", "tools", "same ranking text")
	high.Importance = 5
	for _, item := range []struct {
		path string
		m    memory.Memory
	}{
		{"tools/low.md", low},
		{"tools/high.md", high},
	} {
		if err := ix.Upsert(ctx, item.path, item.m); err != nil {
			t.Fatalf("Upsert %s: %v", item.m.ID, err)
		}
	}

	hits, err := ix.Search(ctx, Filter{Query: "ranking"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) < 2 {
		t.Fatalf("got %d hits, want at least 2: %+v", len(hits), hits)
	}
	if hits[0].ID != "01HIGH" {
		t.Fatalf("top hit = %s, want high-importance memory; hits=%+v", hits[0].ID, hits)
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

func TestUpsertReplacesRelationshipsAndDelete(t *testing.T) {
	ix := openIndex(t)
	ctx := context.Background()

	m := mem(t, "01REL", "Recall graph", "projects", "relationships create graph context")
	m.Relationships = []memory.Relationship{
		{TargetID: "01TARGETA", Type: memory.RelationshipUsesTool, Note: "uses MCP"},
		{TargetID: "01TARGETB", Type: memory.RelationshipDependsOn},
	}
	if err := ix.Upsert(ctx, "projects/graph.md", m); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	assertRelationshipRows(t, ix, "01REL", []memory.Relationship{
		{TargetID: "01TARGETA", Type: memory.RelationshipUsesTool, Note: "uses MCP"},
		{TargetID: "01TARGETB", Type: memory.RelationshipDependsOn},
	})

	m.Relationships = []memory.Relationship{{TargetID: "01TARGETC", Type: memory.RelationshipSupersedes, Note: "new fact"}}
	if err := ix.Upsert(ctx, "projects/graph.md", m); err != nil {
		t.Fatalf("re-Upsert: %v", err)
	}
	assertRelationshipRows(t, ix, "01REL", []memory.Relationship{{TargetID: "01TARGETC", Type: memory.RelationshipSupersedes, Note: "new fact"}})

	if err := ix.Delete(ctx, "01REL"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	assertRelationshipRows(t, ix, "01REL", nil)
}

func assertRelationshipRows(t *testing.T, ix *Index, memoryID string, want []memory.Relationship) {
	t.Helper()
	rows, err := ix.sql.Query("SELECT target_id, type, note FROM memory_relationships WHERE source_id = ? ORDER BY target_id, type", memoryID)
	if err != nil {
		t.Fatalf("query relationships: %v", err)
	}
	defer rows.Close()
	var got []memory.Relationship
	for rows.Next() {
		var rel memory.Relationship
		var relType string
		if err := rows.Scan(&rel.TargetID, &relType, &rel.Note); err != nil {
			t.Fatalf("scan relationship: %v", err)
		}
		rel.Type = memory.RelationshipType(relType)
		got = append(got, rel)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("relationship rows: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("relationships = %+v, want %+v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("relationships = %+v, want %+v", got, want)
		}
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

func TestSchemaEnforcesConstraints(t *testing.T) {
	ix := openIndex(t)
	ctx := context.Background()
	first := mem(t, "01UNIQ1", "First", "tools", "body")
	second := mem(t, "01UNIQ2", "Second", "tools", "body")
	if err := ix.Upsert(ctx, "tools/same.md", first); err != nil {
		t.Fatalf("Upsert first: %v", err)
	}
	if err := ix.Upsert(ctx, "tools/same.md", second); err == nil {
		t.Fatal("expected duplicate path to fail")
	}
	if _, err := ix.sql.ExecContext(ctx, "INSERT INTO tags (memory_id, tag) VALUES (?, ?)", "missing", "tag"); err == nil {
		t.Fatal("expected foreign key failure for unknown tag memory")
	}
	if _, err := ix.sql.ExecContext(ctx, "INSERT INTO tags (memory_id, tag) VALUES (?, ?)", first.ID, "dup"); err != nil {
		t.Fatalf("insert first tag: %v", err)
	}
	if _, err := ix.sql.ExecContext(ctx, "INSERT INTO tags (memory_id, tag) VALUES (?, ?)", first.ID, "dup"); err == nil {
		t.Fatal("expected duplicate tag to fail")
	}
	if _, err := ix.sql.ExecContext(ctx, "INSERT INTO links (memory_id, target_id) VALUES (?, ?)", first.ID, "target"); err != nil {
		t.Fatalf("insert first link: %v", err)
	}
	if _, err := ix.sql.ExecContext(ctx, "INSERT INTO links (memory_id, target_id) VALUES (?, ?)", first.ID, "target"); err == nil {
		t.Fatal("expected duplicate link to fail")
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

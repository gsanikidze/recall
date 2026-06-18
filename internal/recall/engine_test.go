package recall

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"recall/internal/embedding"
	"recall/internal/index"
	"recall/internal/memory"
	"recall/internal/vault"
)

func newEngine(t *testing.T) *Engine {
	t.Helper()
	proj := t.TempDir()
	e, err := Open(proj)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = e.Close() })
	if err := e.Vault().Scaffold(); err != nil {
		t.Fatalf("Scaffold: %v", err)
	}
	return e
}

func newEngineWithFakeIndex(t *testing.T) (*Engine, *fakeIndex) {
	t.Helper()
	v := vault.Open(t.TempDir())
	if err := v.Scaffold(); err != nil {
		t.Fatalf("Scaffold: %v", err)
	}
	fake := &fakeIndex{paths: map[string]string{}}
	return &Engine{vault: v, index: fake}, fake
}

func sampleEngineMemory(t *testing.T) memory.Memory {
	t.Helper()
	d, err := memory.ParseDate("2026-06-07")
	if err != nil {
		t.Fatalf("ParseDate: %v", err)
	}
	return memory.Memory{
		ID:         memory.NewID(),
		Title:      "Original",
		Domain:     "tools",
		Created:    d,
		Updated:    d,
		Importance: 3,
		Lifecycle:  memory.Evergreen,
		Body:       "original body",
	}
}

type fakeIndex struct {
	paths      map[string]string
	embeddings []index.Embedding
	upsertErr  error
	deleteErr  error
}

func (f *fakeIndex) Upsert(_ context.Context, relPath string, m memory.Memory) error {
	if f.upsertErr != nil {
		return f.upsertErr
	}
	f.paths[m.ID] = relPath
	return nil
}

func (f *fakeIndex) Delete(_ context.Context, id string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	delete(f.paths, id)
	return nil
}

func (f *fakeIndex) Path(_ context.Context, id string) (string, error) {
	relPath, ok := f.paths[id]
	if !ok {
		return "", sql.ErrNoRows
	}
	return relPath, nil
}

func (f *fakeIndex) Search(context.Context, index.Filter) ([]index.Hit, error) { return nil, nil }
func (f *fakeIndex) ListIDs(context.Context) ([]string, error)                 { return nil, nil }
func (f *fakeIndex) UpsertEmbedding(_ context.Context, e index.Embedding) error {
	for i, existing := range f.embeddings {
		if existing.MemoryID == e.MemoryID && existing.Provider == e.Provider && existing.Model == e.Model {
			f.embeddings[i] = e
			return nil
		}
	}
	f.embeddings = append(f.embeddings, e)
	return nil
}
func (f *fakeIndex) Embeddings(_ context.Context, provider, model string) ([]index.Embedding, error) {
	var out []index.Embedding
	for _, e := range f.embeddings {
		if e.Provider == provider && e.Model == model {
			out = append(out, e)
		}
	}
	return out, nil
}
func (f *fakeIndex) Close() error { return nil }

func TestAddGetSearch(t *testing.T) {
	e := newEngine(t)
	ctx := context.Background()

	m, relPath, err := e.Add(ctx, AddParams{
		Title:  "Kamal deploy",
		Body:   "Production deploys run through **Kamal**.",
		Domain: "tools",
		Tags:   []string{"deploy"},
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if m.Lifecycle != memory.Evergreen || m.Created.IsZero() {
		t.Errorf("defaults not applied: %+v", m)
	}

	// The MD file exists in the vault and keeps its formatting.
	data, err := os.ReadFile(filepath.Join(e.Vault().Root(), relPath))
	if err != nil {
		t.Fatalf("reading vault file: %v", err)
	}
	if !strings.Contains(string(data), "**Kamal**") {
		t.Errorf("vault file lost markdown formatting:\n%s", data)
	}

	got, _, err := e.Get(ctx, m.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "Kamal deploy" {
		t.Errorf("Get title = %q", got.Title)
	}

	hits, err := e.Search(ctx, index.Filter{Query: "kamal"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) != 1 || hits[0].ID != m.ID {
		t.Errorf("Search = %+v", hits)
	}
}

func TestEmbedAllEmbedsAndSkipsUnchangedMemories(t *testing.T) {
	e := newEngine(t)
	ctx := context.Background()

	if _, _, err := e.Add(ctx, AddParams{Title: "Phone sync", Body: "iPhone setup preference", Domain: "tools"}); err != nil {
		t.Fatalf("Add first: %v", err)
	}
	if _, _, err := e.Add(ctx, AddParams{Title: "Recall policy", Body: "local-first memory policy", Domain: "decisions"}); err != nil {
		t.Fatalf("Add second: %v", err)
	}

	provider := embedding.NewFakeProvider("fake-32", 32)
	stats, err := e.EmbedAll(ctx, provider, false)
	if err != nil {
		t.Fatalf("EmbedAll first: %v", err)
	}
	if stats.Embedded != 2 || stats.Skipped != 0 || stats.Failed != 0 {
		t.Fatalf("first stats = %+v, want embedded 2 skipped 0 failed 0", stats)
	}

	stats, err = e.EmbedAll(ctx, provider, false)
	if err != nil {
		t.Fatalf("EmbedAll second: %v", err)
	}
	if stats.Embedded != 0 || stats.Skipped != 2 || stats.Failed != 0 {
		t.Fatalf("second stats = %+v, want embedded 0 skipped 2 failed 0", stats)
	}
}

func TestAddImportance(t *testing.T) {
	e := newEngine(t)
	ctx := context.Background()

	defaulted, _, err := e.Add(ctx, AddParams{Title: "Default rank", Body: "body", Domain: "tools"})
	if err != nil {
		t.Fatalf("Add default importance: %v", err)
	}
	if defaulted.Importance != 3 {
		t.Fatalf("default importance = %d, want 3", defaulted.Importance)
	}

	critical, _, err := e.Add(ctx, AddParams{Title: "Critical rank", Body: "body", Domain: "tools", Importance: 5})
	if err != nil {
		t.Fatalf("Add explicit importance: %v", err)
	}
	if critical.Importance != 5 {
		t.Fatalf("explicit importance = %d, want 5", critical.Importance)
	}
	got, _, err := e.Get(ctx, critical.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Importance != 5 {
		t.Fatalf("stored importance = %d, want 5", got.Importance)
	}
}

func TestUpdateImportance(t *testing.T) {
	e := newEngine(t)
	ctx := context.Background()
	m, _, err := e.Add(ctx, AddParams{Title: "Ranked", Body: "old", Domain: "tools"})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	importance := 4
	updated, _, err := e.Update(ctx, m.ID, UpdateParams{Importance: &importance})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Importance != 4 {
		t.Fatalf("updated importance = %d, want 4", updated.Importance)
	}
}

func TestAddRelationships(t *testing.T) {
	e := newEngine(t)
	ctx := context.Background()

	m, _, err := e.Add(ctx, AddParams{
		Title:  "Graph fact",
		Body:   "Hermes uses Recall MCP.",
		Domain: "tools",
		Relationships: []memory.Relationship{
			{TargetID: "01PROJECT000000000000000001", Type: memory.RelationshipUsesTool, Note: "via MCP"},
		},
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	got, _, err := e.Get(ctx, m.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Relationships) != 1 || got.Relationships[0].Type != memory.RelationshipUsesTool {
		t.Fatalf("relationships = %+v, want uses_tool edge", got.Relationships)
	}
}

func TestUpdateRelationships(t *testing.T) {
	e := newEngine(t)
	ctx := context.Background()
	m, _, err := e.Add(ctx, AddParams{Title: "Graph fact", Body: "old", Domain: "tools"})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	rels := []memory.Relationship{{TargetID: "01PROJECT000000000000000001", Type: memory.RelationshipAboutProject}}
	updated, _, err := e.Update(ctx, m.ID, UpdateParams{Relationships: &rels})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if len(updated.Relationships) != 1 || updated.Relationships[0].Type != memory.RelationshipAboutProject {
		t.Fatalf("relationships = %+v, want about_project edge", updated.Relationships)
	}
}

func TestGraphReturnsMemoryRelationships(t *testing.T) {
	e := newEngine(t)
	ctx := context.Background()

	target, _, err := e.Add(ctx, AddParams{Title: "Recall project", Body: "project body", Domain: "projects"})
	if err != nil {
		t.Fatalf("Add target: %v", err)
	}
	source, _, err := e.Add(ctx, AddParams{
		Title:  "Hermes Recall MCP",
		Body:   "Hermes uses Recall MCP.",
		Domain: "tools",
		Relationships: []memory.Relationship{{
			TargetID: target.ID,
			Type:     memory.RelationshipUsesTool,
			Note:     "stdio MCP",
		}},
	})
	if err != nil {
		t.Fatalf("Add source: %v", err)
	}

	graph, err := e.Graph(ctx, "")
	if err != nil {
		t.Fatalf("Graph: %v", err)
	}
	if len(graph.Nodes) != 2 {
		t.Fatalf("nodes = %+v, want 2", graph.Nodes)
	}
	if len(graph.Edges) != 1 {
		t.Fatalf("edges = %+v, want 1", graph.Edges)
	}
	edge := graph.Edges[0]
	if edge.Source != source.ID || edge.Target != target.ID || edge.Type != string(memory.RelationshipUsesTool) || edge.Note != "stdio MCP" {
		t.Fatalf("edge = %+v", edge)
	}

	toolsGraph, err := e.Graph(ctx, "tools")
	if err != nil {
		t.Fatalf("Graph tools: %v", err)
	}
	if len(toolsGraph.Nodes) != 2 || len(toolsGraph.Edges) != 1 {
		t.Fatalf("tools graph = %+v, want source plus target placeholder and edge", toolsGraph)
	}
}

func TestAddUnknownDomainRejected(t *testing.T) {
	e := newEngine(t)
	if _, _, err := e.Add(context.Background(), AddParams{Title: "x", Body: "y", Domain: "nope"}); err == nil {
		t.Error("expected error for unknown domain")
	}
}

func TestAddExpiresRequiresDate(t *testing.T) {
	e := newEngine(t)
	ctx := context.Background()
	if _, _, err := e.Add(ctx, AddParams{Title: "x", Body: "y", Domain: "decisions", Lifecycle: "expires"}); err == nil {
		t.Error("expected error: expires without expires_on")
	}
	if _, _, err := e.Add(ctx, AddParams{Title: "x", Body: "y", Domain: "decisions", Lifecycle: "expires", ExpiresOn: "2030-01-01"}); err != nil {
		t.Errorf("valid expires rejected: %v", err)
	}
}

func TestUpdate(t *testing.T) {
	e := newEngine(t)
	ctx := context.Background()
	m, _, _ := e.Add(ctx, AddParams{Title: "Draft", Body: "old", Domain: "inbox"})

	newBody := "new content about widgets"
	updated, _, err := e.Update(ctx, m.ID, UpdateParams{Body: &newBody})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Body != newBody {
		t.Errorf("body not updated: %q", updated.Body)
	}
	hits, _ := e.Search(ctx, index.Filter{Query: "widgets"})
	if len(hits) != 1 {
		t.Errorf("search after update = %+v", hits)
	}
}

func TestAddIndexFailureRemovesVaultFile(t *testing.T) {
	e, fake := newEngineWithFakeIndex(t)
	fake.upsertErr = errors.New("boom")

	_, _, err := e.Add(context.Background(), AddParams{Title: "Draft", Body: "body", Domain: "inbox"})
	if err == nil {
		t.Fatal("expected Add error")
	}
	paths, scanErr := e.Vault().Scan()
	if scanErr != nil {
		t.Fatalf("Scan: %v", scanErr)
	}
	if len(paths) != 0 {
		t.Fatalf("vault file left behind after index failure: %v", paths)
	}
}

func TestUpdateIndexFailureRestoresOldFile(t *testing.T) {
	e, fake := newEngineWithFakeIndex(t)
	original := sampleEngineMemory(t)
	relPath, err := e.Vault().Write(original)
	if err != nil {
		t.Fatalf("Write original: %v", err)
	}
	fake.paths[original.ID] = relPath
	fake.upsertErr = errors.New("boom")

	newBody := "new body"
	if _, _, err := e.Update(context.Background(), original.ID, UpdateParams{Body: &newBody}); err == nil {
		t.Fatal("expected Update error")
	}
	got, err := e.Vault().Read(relPath)
	if err != nil {
		t.Fatalf("Read restored memory: %v", err)
	}
	if got.Body != original.Body+"\n" && got.Body != original.Body {
		t.Fatalf("body not restored: got %q want %q", got.Body, original.Body)
	}
}

func TestDeleteIndexFailureRestoresVaultFile(t *testing.T) {
	e, fake := newEngineWithFakeIndex(t)
	original := sampleEngineMemory(t)
	relPath, err := e.Vault().Write(original)
	if err != nil {
		t.Fatalf("Write original: %v", err)
	}
	fake.paths[original.ID] = relPath
	fake.deleteErr = errors.New("boom")

	if err := e.Delete(context.Background(), original.ID); err == nil {
		t.Fatal("expected Delete error")
	}
	got, err := e.Vault().Read(relPath)
	if err != nil {
		t.Fatalf("Read restored memory: %v", err)
	}
	if got.ID != original.ID {
		t.Fatalf("restored memory id = %q, want %q", got.ID, original.ID)
	}
}

func TestReindexAfterHandEditAndDelete(t *testing.T) {
	e := newEngine(t)
	ctx := context.Background()
	m, relPath, _ := e.Add(ctx, AddParams{Title: "Editable", Body: "original text", Domain: "research"})

	// Simulate a human hand-editing the MD file directly.
	edited, err := e.Vault().Read(relPath)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	edited.Body = "rewritten by a human about quasars"
	if err := e.Vault().WriteAt(relPath, edited); err != nil {
		t.Fatalf("WriteAt: %v", err)
	}

	stats, err := e.Reindex(ctx)
	if err != nil {
		t.Fatalf("Reindex: %v", err)
	}
	if stats.Indexed != 1 {
		t.Errorf("Indexed = %d, want 1", stats.Indexed)
	}
	if hits, _ := e.Search(ctx, index.Filter{Query: "quasars"}); len(hits) != 1 {
		t.Errorf("hand-edit not reflected after reindex: %+v", hits)
	}

	// Delete the file on disk; reindex must drop it from the index.
	if err := os.Remove(filepath.Join(e.Vault().Root(), relPath)); err != nil {
		t.Fatalf("rm: %v", err)
	}
	stats, err = e.Reindex(ctx)
	if err != nil {
		t.Fatalf("Reindex 2: %v", err)
	}
	if stats.Deleted != 1 {
		t.Errorf("Deleted = %d, want 1", stats.Deleted)
	}
	if _, _, err := e.Get(ctx, m.ID); err != ErrNotFound {
		t.Errorf("expected ErrNotFound after reindex delete, got %v", err)
	}
}

func TestRebuildFromVault(t *testing.T) {
	e := newEngine(t)
	ctx := context.Background()
	e.Add(ctx, AddParams{Title: "One", Body: "alpha", Domain: "tools"})
	e.Add(ctx, AddParams{Title: "Two", Body: "beta", Domain: "tools"})

	// Wipe the database file and reopen: reindex must restore everything.
	root := e.Vault().Root()
	proj := filepath.Dir(root)
	_ = e.Close()
	if err := os.Remove(filepath.Join(proj, "db", "recall.sqlite")); err != nil {
		t.Fatalf("rm db: %v", err)
	}

	e2, err := Open(proj)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer e2.Close()
	if _, err := e2.Reindex(ctx); err != nil {
		t.Fatalf("Reindex: %v", err)
	}
	if hits, _ := e2.Search(ctx, index.Filter{}); len(hits) != 2 {
		t.Errorf("expected 2 memories rebuilt from vault, got %d", len(hits))
	}
}

func TestReindexRejectsDuplicateMemoryIDs(t *testing.T) {
	e := newEngine(t)
	m1 := sampleEngineMemory(t)
	m1.Title = "First duplicate"
	m2 := m1
	m2.Title = "Second duplicate"

	firstPath := filepath.Join("tools", "first.md")
	secondPath := filepath.Join("tools", "second.md")
	if err := e.Vault().WriteAt(firstPath, m1); err != nil {
		t.Fatalf("WriteAt first: %v", err)
	}
	if err := e.Vault().WriteAt(secondPath, m2); err != nil {
		t.Fatalf("WriteAt second: %v", err)
	}

	_, err := e.Reindex(context.Background())
	if err == nil {
		t.Fatal("expected duplicate id error")
	}
	if !strings.Contains(err.Error(), m1.ID) || !strings.Contains(err.Error(), firstPath) || !strings.Contains(err.Error(), secondPath) {
		t.Fatalf("duplicate error missing id or paths: %v", err)
	}
}

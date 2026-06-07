package recall

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"recall/internal/index"
	"recall/internal/memory"
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
	if !contains(string(data), "**Kamal**") {
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

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

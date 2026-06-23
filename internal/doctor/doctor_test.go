package doctor

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"recall/internal/memory"
	"recall/internal/recall"
)

func newDoctorEngine(t *testing.T) (*recall.Engine, string) {
	t.Helper()
	project := t.TempDir()
	e, err := recall.Open(project)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = e.Close() })
	if err := e.Vault().Scaffold(); err != nil {
		t.Fatalf("Scaffold: %v", err)
	}
	return e, project
}

func writeDoctorMemory(t *testing.T, e *recall.Engine, relPath string, m memory.Memory) {
	t.Helper()
	data, err := m.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	abs := filepath.Join(e.Vault().Root(), relPath)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(abs, data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func validDoctorMemory(title, body string) memory.Memory {
	today := memory.Today()
	return memory.Memory{
		ID:         memory.NewID(),
		Title:      title,
		Domain:     "tools",
		Created:    today,
		Updated:    today,
		Importance: 3,
		Lifecycle:  memory.Evergreen,
		Body:       body,
	}
}

func TestDeepDoctorReportsUnindexedVaultFiles(t *testing.T) {
	e, project := newDoctorEngine(t)
	ctx := context.Background()
	if _, _, err := e.Add(ctx, recall.AddParams{Title: "Indexed", Domain: "tools", Body: "indexed body"}); err != nil {
		t.Fatalf("Add indexed: %v", err)
	}

	unindexed := validDoctorMemory("Unindexed", "only in vault")
	writeDoctorMemory(t, e, filepath.Join("tools", "unindexed.md"), unindexed)

	report := Run(ctx, e, Options{Deep: true}, project, JoinVaultPath(project), JoinDBPath(project), "")
	if report.OK {
		t.Fatalf("doctor ok=true with unindexed vault file: %+v", report)
	}
	if len(report.UnindexedVaultFiles) != 1 {
		t.Fatalf("unindexed files = %+v", report.UnindexedVaultFiles)
	}
	if report.UnindexedVaultFiles[0].ID != unindexed.ID || report.UnindexedVaultFiles[0].Path != filepath.Join("tools", "unindexed.md") {
		t.Fatalf("unindexed file = %+v", report.UnindexedVaultFiles[0])
	}
	if len(report.Suggestions) == 0 || report.Suggestions[0].ID != "vault-index-drift" {
		t.Fatalf("suggestions = %+v", report.Suggestions)
	}
}

func TestDeepDoctorReportsDuplicateVaultIDs(t *testing.T) {
	e, project := newDoctorEngine(t)
	ctx := context.Background()
	indexed, _, err := e.Add(ctx, recall.AddParams{Title: "Original", Domain: "tools", Body: "original body"})
	if err != nil {
		t.Fatalf("Add indexed: %v", err)
	}
	duplicate := indexed
	duplicate.Title = "Duplicate"
	duplicate.Body = "duplicate body"
	writeDoctorMemory(t, e, filepath.Join("tools", "duplicate.md"), duplicate)

	report := Run(ctx, e, Options{Deep: true}, project, JoinVaultPath(project), JoinDBPath(project), "")
	if report.OK {
		t.Fatalf("doctor ok=true with duplicate vault ids: %+v", report)
	}
	if len(report.DuplicateVaultIDs) != 1 {
		t.Fatalf("duplicates = %+v", report.DuplicateVaultIDs)
	}
	got := report.DuplicateVaultIDs[0]
	if got.ID != indexed.ID || len(got.Paths) != 2 {
		t.Fatalf("duplicate entry = %+v", got)
	}
}

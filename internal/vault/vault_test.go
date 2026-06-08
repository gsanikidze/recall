package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"recall/internal/memory"
)

func newVault(t *testing.T) *Vault {
	t.Helper()
	v := Open(t.TempDir())
	if err := v.Scaffold(); err != nil {
		t.Fatalf("Scaffold: %v", err)
	}
	return v
}

func sampleMemory(t *testing.T) memory.Memory {
	t.Helper()
	d, _ := memory.ParseDate("2026-06-07")
	return memory.Memory{
		ID:        memory.NewID(),
		Title:     "Kamal deploy",
		Domain:    "tools",
		Created:   d,
		Updated:   d,
		Lifecycle: memory.Evergreen,
		Body:      "Use Kamal.",
	}
}

func TestScaffoldCreatesDomainsAndIndex(t *testing.T) {
	v := newVault(t)

	for _, d := range PredefinedDomains {
		readme := filepath.Join(v.DomainPath(d.Name), readmeName)
		if _, err := os.Stat(readme); err != nil {
			t.Errorf("missing README for domain %q: %v", d.Name, err)
		}
	}

	index, err := os.ReadFile(filepath.Join(v.Root(), readmeName))
	if err != nil {
		t.Fatalf("reading index: %v", err)
	}
	if !strings.Contains(string(index), "tools/") {
		t.Errorf("index missing tools domain:\n%s", index)
	}
}

func TestListDomainsParsesDescriptions(t *testing.T) {
	v := newVault(t)
	if err := os.MkdirAll(filepath.Join(v.Root(), ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(v.Root(), "Bad Name"), 0o755); err != nil {
		t.Fatal(err)
	}

	domains, err := v.ListDomains()
	if err != nil {
		t.Fatalf("ListDomains: %v", err)
	}
	got := map[string]string{}
	for _, d := range domains {
		got[d.Name] = d.Description
		if strings.HasPrefix(d.Name, ".") || strings.Contains(d.Name, " ") {
			t.Fatalf("invalid folder listed as domain: %+v", d)
		}
	}
	if !strings.Contains(got["tools"], "Reusable tools") {
		t.Errorf("tools description = %q", got["tools"])
	}
	if _, ok := got["decisions"]; !ok {
		t.Error("missing decisions domain")
	}
}

func TestAddDomain(t *testing.T) {
	v := newVault(t)
	if err := v.AddDomain("Clients", "People we sell to."); err != nil {
		t.Fatalf("AddDomain: %v", err)
	}
	desc := v.readDomainDescription("clients")
	if desc != "People we sell to." {
		t.Errorf("description = %q", desc)
	}
	index, _ := os.ReadFile(filepath.Join(v.Root(), readmeName))
	if !strings.Contains(string(index), "clients/") {
		t.Errorf("index not refreshed:\n%s", index)
	}

	if err := v.AddDomain("bad name!", "x"); err == nil {
		t.Error("expected error for invalid domain name")
	}
}

func TestWriteReadRoundTrip(t *testing.T) {
	v := newVault(t)
	m := sampleMemory(t)

	rel, err := v.Write(m)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	want := filepath.Join("tools", "2026-06-07-kamal-deploy.md")
	if rel != want {
		t.Errorf("relpath = %q, want %q", rel, want)
	}

	got, err := v.Read(rel)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.ID != m.ID || got.Title != m.Title {
		t.Errorf("round-trip mismatch: got %+v", got)
	}
}

func TestReadRejectsInvalidMemoryWithPathContext(t *testing.T) {
	v := newVault(t)
	rel := filepath.Join("tools", "2026-06-07-invalid.md")
	data := `---
id: 01J8X3QH000000000000000000
title: Invalid memory
domain: tools
created: "2026-06-07"
lifecycle: evergreen
---

body without updated date
`
	if err := os.WriteFile(filepath.Join(v.Root(), rel), []byte(data), 0o644); err != nil {
		t.Fatalf("write invalid memory: %v", err)
	}

	_, err := v.Read(rel)
	if err == nil {
		t.Fatal("expected invalid memory error")
	}
	if !strings.Contains(err.Error(), rel) {
		t.Fatalf("error missing path %q: %v", rel, err)
	}
	if !strings.Contains(err.Error(), "updated date is required") {
		t.Fatalf("error missing validation cause: %v", err)
	}
}

func TestWriteCollisionAppendsSuffix(t *testing.T) {
	v := newVault(t)
	m1 := sampleMemory(t)
	m2 := sampleMemory(t) // same title/date/domain, different id

	rel1, err := v.Write(m1)
	if err != nil {
		t.Fatalf("Write m1: %v", err)
	}
	rel2, err := v.Write(m2)
	if err != nil {
		t.Fatalf("Write m2: %v", err)
	}
	if rel1 == rel2 {
		t.Fatalf("expected distinct paths, both = %q", rel1)
	}
	// Re-writing the same memory keeps its path stable.
	rel1again, err := v.Write(m1)
	if err != nil {
		t.Fatalf("re-Write m1: %v", err)
	}
	if rel1again != rel1 {
		t.Errorf("re-write changed path: %q -> %q", rel1, rel1again)
	}
}

func TestScanAndReadAll(t *testing.T) {
	v := newVault(t)
	if _, err := v.Write(sampleMemory(t)); err != nil {
		t.Fatalf("Write: %v", err)
	}

	paths, err := v.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(paths) != 1 {
		t.Fatalf("Scan returned %d paths, want 1 (READMEs must be excluded): %v", len(paths), paths)
	}

	all, err := v.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(all) != 1 || all[0].Memory.Title != "Kamal deploy" {
		t.Errorf("ReadAll = %+v", all)
	}
}

func TestDelete(t *testing.T) {
	v := newVault(t)
	rel, _ := v.Write(sampleMemory(t))
	if err := v.Delete(rel); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := os.Stat(filepath.Join(v.Root(), rel)); !os.IsNotExist(err) {
		t.Errorf("file still exists after delete")
	}
	// Deleting a missing file is not an error.
	if err := v.Delete(rel); err != nil {
		t.Errorf("Delete missing: %v", err)
	}
}

func TestRejectsUnsafeRelativePaths(t *testing.T) {
	v := newVault(t)
	m := sampleMemory(t)
	for _, rel := range []string{"../escape.md", "/tmp/escape.md", "tools/../../escape.md"} {
		if err := v.WriteAt(rel, m); err == nil {
			t.Fatalf("WriteAt accepted unsafe path %q", rel)
		}
		if _, err := v.Read(rel); err == nil {
			t.Fatalf("Read accepted unsafe path %q", rel)
		}
		if err := v.Delete(rel); err == nil {
			t.Fatalf("Delete accepted unsafe path %q", rel)
		}
	}
}

func TestRejectsSymlinkDomain(t *testing.T) {
	v := newVault(t)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(v.Root(), "escape")); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	if v.HasDomain("escape") {
		t.Fatal("HasDomain accepted symlinked domain")
	}

	m := sampleMemory(t)
	m.Domain = "escape"
	if _, err := v.Write(m); err == nil {
		t.Fatal("Write accepted symlinked domain")
	}
}

func TestRejectsSymlinkEscapes(t *testing.T) {
	v := newVault(t)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(v.Root(), "links")); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	m := sampleMemory(t)
	rel := filepath.Join("links", "escape.md")
	if err := v.WriteAt(rel, m); err == nil {
		t.Fatal("WriteAt accepted symlink escape path")
	}

	data, err := m.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	outsideFile := filepath.Join(outside, "escape.md")
	if err := os.WriteFile(outsideFile, data, 0o644); err != nil {
		t.Fatalf("write outside memory: %v", err)
	}

	if _, err := v.Read(rel); err == nil {
		t.Fatal("Read accepted symlink escape path")
	}
	if err := v.Delete(rel); err == nil {
		t.Fatal("Delete accepted symlink escape path")
	}
	if _, err := os.Stat(outsideFile); err != nil {
		t.Fatalf("Delete touched escaped file: %v", err)
	}
}

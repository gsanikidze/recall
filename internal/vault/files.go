package vault

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"recall/internal/memory"
)

// ScannedMemory pairs a parsed memory with its vault-relative path.
type ScannedMemory struct {
	RelPath string
	Memory  memory.Memory
}

// Write stores a memory as a Markdown file under its domain folder and returns
// the vault-relative path. The file name is YYYY-MM-DD-<slug>.md; if that name
// is already taken by a different memory, a short id suffix is appended to keep
// it unique.
func (v *Vault) Write(m memory.Memory) (string, error) {
	if err := m.Validate(); err != nil {
		return "", err
	}
	dir := v.DomainPath(m.Domain)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("vault: creating domain %q: %w", m.Domain, err)
	}

	name := m.Filename()
	if taken, err := v.nameTakenByOther(m.Domain, name, m.ID); err != nil {
		return "", err
	} else if taken {
		name = appendIDSuffix(name, m.ID)
	}

	rel := filepath.Join(m.Domain, name)
	if err := v.WriteAt(rel, m); err != nil {
		return "", err
	}
	return rel, nil
}

// WriteAt writes a memory to an exact vault-relative path, overwriting it. Used
// for updates, where the path is already known.
func (v *Vault) WriteAt(relPath string, m memory.Memory) error {
	data, err := m.Marshal()
	if err != nil {
		return err
	}
	abs := filepath.Join(v.root, relPath)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return fmt.Errorf("vault: creating dir for %q: %w", relPath, err)
	}
	if err := os.WriteFile(abs, data, 0o644); err != nil {
		return fmt.Errorf("vault: writing %q: %w", relPath, err)
	}
	return nil
}

// Read parses the memory file at a vault-relative path.
func (v *Vault) Read(relPath string) (memory.Memory, error) {
	data, err := os.ReadFile(filepath.Join(v.root, relPath))
	if err != nil {
		return memory.Memory{}, fmt.Errorf("vault: reading %q: %w", relPath, err)
	}
	m, err := memory.Parse(data)
	if err != nil {
		return memory.Memory{}, fmt.Errorf("vault: parsing %q: %w", relPath, err)
	}
	return m, nil
}

// Delete removes the memory file at a vault-relative path.
func (v *Vault) Delete(relPath string) error {
	if err := os.Remove(filepath.Join(v.root, relPath)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("vault: deleting %q: %w", relPath, err)
	}
	return nil
}

// Scan returns the vault-relative paths of every memory file (all *.md except
// README.md files), sorted.
func (v *Vault) Scan() ([]string, error) {
	var paths []string
	err := filepath.WalkDir(v.root, func(abs string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".md") || strings.EqualFold(name, readmeName) {
			return nil
		}
		rel, err := filepath.Rel(v.root, abs)
		if err != nil {
			return err
		}
		paths = append(paths, rel)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("vault: scanning: %w", err)
	}
	return paths, nil
}

// ReadAll scans and parses every memory file in the vault.
func (v *Vault) ReadAll() ([]ScannedMemory, error) {
	paths, err := v.Scan()
	if err != nil {
		return nil, err
	}
	out := make([]ScannedMemory, 0, len(paths))
	for _, rel := range paths {
		m, err := v.Read(rel)
		if err != nil {
			return nil, err
		}
		out = append(out, ScannedMemory{RelPath: rel, Memory: m})
	}
	return out, nil
}

// nameTakenByOther reports whether domain/name exists and belongs to a memory
// with a different id.
func (v *Vault) nameTakenByOther(domain, name, id string) (bool, error) {
	abs := filepath.Join(v.DomainPath(domain), name)
	data, err := os.ReadFile(abs)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("vault: checking %q: %w", name, err)
	}
	existing, err := memory.Parse(data)
	if err != nil {
		// Unparseable file in the way: treat as taken to avoid clobbering it.
		return true, nil
	}
	return existing.ID != id, nil
}

// appendIDSuffix inserts a short, lowercased id fragment before the .md suffix.
func appendIDSuffix(name, id string) string {
	short := strings.ToLower(id)
	if len(short) > 6 {
		short = short[len(short)-6:]
	}
	base := strings.TrimSuffix(name, ".md")
	return base + "-" + short + ".md"
}

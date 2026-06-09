// Package recall is the orchestration engine that ties the vault (source of
// truth) to the SQLite index (search cache). Every write goes through here:
// it writes the Markdown file first, then updates the index; if the index step
// fails, the engine compensates by deleting the new file or restoring the old
// file so the rebuildable index never points at missing or half-written vault
// content from writes recall itself makes. Both the CLI and the MCP server call
// into this one engine.
package recall

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"recall/internal/index"
	"recall/internal/memory"
	"recall/internal/vault"
)

// ErrNotFound is returned when a memory id is not in the index.
var ErrNotFound = errors.New("recall: memory not found")

// ErrValidation marks invalid user input at the engine boundary.
var ErrValidation = errors.New("recall: validation failed")

// Engine couples a vault with its index.
type Engine struct {
	vault *vault.Vault
	index indexStore
}

type indexStore interface {
	Upsert(context.Context, string, memory.Memory) error
	Delete(context.Context, string) error
	Path(context.Context, string) (string, error)
	Search(context.Context, index.Filter) ([]index.Hit, error)
	ListIDs(context.Context) ([]string, error)
	Close() error
}

// Open opens the engine for a recall project directory, which must contain a
// vault/ folder; the index lives at db/recall.sqlite (created if needed).
func Open(projectPath string) (*Engine, error) {
	vaultDir := filepath.Join(projectPath, "vault")
	dbDir := filepath.Join(projectPath, "db")
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		return nil, fmt.Errorf("recall: creating db dir: %w", err)
	}
	ix, err := index.Open(filepath.Join(dbDir, "recall.sqlite"))
	if err != nil {
		return nil, err
	}
	return &Engine{vault: vault.Open(vaultDir), index: ix}, nil
}

// Close releases the index handle.
func (e *Engine) Close() error { return e.index.Close() }

// Vault exposes the underlying vault (for domain operations and init scaffolding).
func (e *Engine) Vault() *vault.Vault { return e.vault }

// AddParams describes a new memory. Lifecycle defaults to evergreen; ExpiresOn
// is required (and only valid) when Lifecycle is "expires".
type AddParams struct {
	Title         string
	Body          string
	Domain        string
	Tags          []string
	Project       string
	Lifecycle     string
	ExpiresOn     string
	Source        string
	Links         []string
	Relationships []memory.Relationship
	Importance    int
}

// Add creates a memory: writes its Markdown file (truth) then indexes it.
// It returns the stored memory and its vault-relative path.
func (e *Engine) Add(ctx context.Context, p AddParams) (memory.Memory, string, error) {
	if err := e.requireDomain(p.Domain); err != nil {
		return memory.Memory{}, "", err
	}

	today := memory.Today()
	m := memory.Memory{
		ID:            memory.NewID(),
		Title:         p.Title,
		Domain:        p.Domain,
		Tags:          p.Tags,
		Project:       p.Project,
		Created:       today,
		Updated:       today,
		Importance:    importanceOrDefault(p.Importance),
		Source:        p.Source,
		Links:         p.Links,
		Relationships: p.Relationships,
		Body:          p.Body,
	}
	if err := applyLifecycle(&m, p.Lifecycle, p.ExpiresOn); err != nil {
		return memory.Memory{}, "", err
	}

	relPath, err := e.vault.Write(m)
	if err != nil {
		return memory.Memory{}, "", err
	}
	if err := e.index.Upsert(ctx, relPath, m); err != nil {
		if cleanupErr := e.vault.Delete(relPath); cleanupErr != nil {
			return memory.Memory{}, "", fmt.Errorf("recall: indexing new memory failed: %w; cleanup failed: %v", err, cleanupErr)
		}
		return memory.Memory{}, "", err
	}
	return m, relPath, nil
}

// Get returns a memory and its vault-relative path by id.
func (e *Engine) Get(ctx context.Context, id string) (memory.Memory, string, error) {
	relPath, err := e.index.Path(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return memory.Memory{}, "", ErrNotFound
	}
	if err != nil {
		return memory.Memory{}, "", err
	}
	m, err := e.vault.Read(relPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return memory.Memory{}, "", ErrNotFound
		}
		return memory.Memory{}, "", err
	}
	return m, relPath, nil
}

// Delete removes a memory from both the vault and the index.
func (e *Engine) Delete(ctx context.Context, id string) error {
	m, relPath, err := e.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := e.vault.Delete(relPath); err != nil {
		return err
	}
	if err := e.index.Delete(ctx, id); err != nil {
		if restoreErr := e.vault.WriteAt(relPath, m); restoreErr != nil {
			return fmt.Errorf("recall: deleting index entry failed: %w; restore failed: %v", err, restoreErr)
		}
		return err
	}
	return nil
}

// Search runs a filtered, ranked query against the index.
func (e *Engine) Search(ctx context.Context, f index.Filter) ([]index.Hit, error) {
	return e.index.Search(ctx, f)
}

// MemoryCount returns the number of indexed memories.
func (e *Engine) MemoryCount(ctx context.Context) (int, error) {
	ids, err := e.index.ListIDs(ctx)
	if err != nil {
		return 0, err
	}
	return len(ids), nil
}

// UpdateParams holds optional edits; only non-nil fields are applied. The
// memory's domain cannot change here.
type UpdateParams struct {
	Title         *string
	Body          *string
	Tags          *[]string
	Project       *string
	Lifecycle     *string
	ExpiresOn     *string
	Source        *string
	Links         *[]string
	Relationships *[]memory.Relationship
	Importance    *int
}

// Update applies partial edits to an existing memory, bumps its Updated date,
// rewrites its file in place, and reindexes it. It returns the updated memory
// and its (stable) vault-relative path.
func (e *Engine) Update(ctx context.Context, id string, p UpdateParams) (memory.Memory, string, error) {
	m, relPath, err := e.Get(ctx, id)
	if err != nil {
		return memory.Memory{}, "", err
	}
	original := m

	if p.Title != nil {
		m.Title = *p.Title
	}
	if p.Body != nil {
		m.Body = *p.Body
	}
	if p.Tags != nil {
		m.Tags = *p.Tags
	}
	if p.Project != nil {
		m.Project = *p.Project
	}
	if p.Source != nil {
		m.Source = *p.Source
	}
	if p.Links != nil {
		m.Links = *p.Links
	}
	if p.Relationships != nil {
		m.Relationships = *p.Relationships
	}
	if p.Importance != nil {
		m.Importance = *p.Importance
	}
	if p.Lifecycle != nil || p.ExpiresOn != nil {
		lifecycle := string(m.Lifecycle)
		if p.Lifecycle != nil {
			lifecycle = *p.Lifecycle
		}
		expires := m.ExpiresOn.String()
		if p.ExpiresOn != nil {
			expires = *p.ExpiresOn
		}
		if err := applyLifecycle(&m, lifecycle, expires); err != nil {
			return memory.Memory{}, "", err
		}
	}
	m.Updated = memory.Today()

	if err := m.Validate(); err != nil {
		return memory.Memory{}, "", err
	}
	// Rewrite in place to keep the path (and any inbound references) stable.
	if err := e.vault.WriteAt(relPath, m); err != nil {
		return memory.Memory{}, "", err
	}
	if err := e.index.Upsert(ctx, relPath, m); err != nil {
		if restoreErr := e.vault.WriteAt(relPath, original); restoreErr != nil {
			return memory.Memory{}, "", fmt.Errorf("recall: indexing updated memory failed: %w; restore failed: %v", err, restoreErr)
		}
		return memory.Memory{}, "", err
	}
	return m, relPath, nil
}

// ReindexStats summarizes a reindex run.
type ReindexStats struct {
	Indexed int // files (re)indexed from the vault
	Deleted int // index rows removed because their file is gone
}

// Reindex rebuilds the index from the vault: it re-reads every memory file,
// upserts it, then deletes index rows whose file no longer exists. This is how
// hand-edits and a deleted database are reconciled.
func (e *Engine) Reindex(ctx context.Context) (ReindexStats, error) {
	scanned, err := e.vault.ReadAll()
	if err != nil {
		return ReindexStats{}, err
	}
	present := make(map[string]struct{}, len(scanned))
	pathsByID := make(map[string]string, len(scanned))
	var stats ReindexStats
	for _, sm := range scanned {
		if err := sm.Memory.Validate(); err != nil {
			return stats, fmt.Errorf("recall: invalid memory %s: %w", sm.RelPath, err)
		}
		if existingPath, ok := pathsByID[sm.Memory.ID]; ok {
			return stats, fmt.Errorf("recall: duplicate memory id %s in %s and %s", sm.Memory.ID, existingPath, sm.RelPath)
		}
		pathsByID[sm.Memory.ID] = sm.RelPath
		if err := e.index.Upsert(ctx, sm.RelPath, sm.Memory); err != nil {
			return stats, err
		}
		present[sm.Memory.ID] = struct{}{}
		stats.Indexed++
	}

	indexed, err := e.index.ListIDs(ctx)
	if err != nil {
		return stats, err
	}
	for _, id := range indexed {
		if _, ok := present[id]; ok {
			continue
		}
		if err := e.index.Delete(ctx, id); err != nil {
			return stats, err
		}
		stats.Deleted++
	}
	return stats, nil
}

// requireDomain confirms a domain folder exists, so memories are routed only to
// known, self-described domains. Agents discover valid domains via ListDomains.
func (e *Engine) requireDomain(name string) error {
	if name == "" {
		return fmt.Errorf("%w: domain is required", ErrValidation)
	}
	if !e.vault.HasDomain(name) {
		return fmt.Errorf("%w: unknown domain %q (create it with `recall domain add` or pick an existing one)", ErrValidation, name)
	}
	return nil
}

// applyLifecycle resolves lifecycle + expires_on string inputs (via the memory
// package, which owns the rule) and sets them on the memory.
func applyLifecycle(m *memory.Memory, lifecycle, expiresOn string) error {
	lc, exp, err := memory.NormalizeLifecycle(lifecycle, expiresOn)
	if err != nil {
		return err
	}
	m.Lifecycle = lc
	m.ExpiresOn = exp
	return nil
}

func importanceOrDefault(importance int) int {
	if importance == 0 {
		return 3
	}
	return importance
}

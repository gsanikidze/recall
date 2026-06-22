// Package doctor audits a recall engine: project paths, vault/index drift,
// invalid memory files, and embedding coverage. CLI and API server both
// consume Run so audit logic stays in one place.
package doctor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"recall/internal/embedding"
	"recall/internal/recall"
)

// Options controls which audits Run performs.
type Options struct {
	Deep       bool   // audit vault/index drift and invalid memory files
	Embeddings bool   // report embedding coverage for indexed memories
	Provider   string // embedding provider for Embeddings
	Model      string // embedding model for Embeddings
}

// Report is the audit result. JSON tags are stable: CLI tests assert on them.
type Report struct {
	OK                bool             `json:"ok"`
	ProjectPath       string          `json:"project_path"`
	ConfigPath        string          `json:"config_path"`
	VaultPath         string          `json:"vault_path"`
	DBPath            string          `json:"db_path"`
	Domains           int             `json:"domains"`
	Memories          int             `json:"memories"`
	VaultMemories     int             `json:"vault_memories,omitempty"`
	IndexMemories     int             `json:"index_memories,omitempty"`
	InvalidFiles      []InvalidFile   `json:"invalid_files,omitempty"`
	StaleIndexIDs     []string        `json:"stale_index_ids,omitempty"`
	MissingIndexPaths []MissingIndex  `json:"missing_index_paths,omitempty"`
	Embeddings        *EmbeddingReady `json:"embeddings,omitempty"`
	Errors            []string        `json:"errors"`
}

// InvalidFile is a vault markdown file that failed to parse.
type InvalidFile struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

// MissingIndex is an indexed row whose vault file no longer exists.
type MissingIndex struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

// EmbeddingReady summarises embedding coverage for indexed memories.
type EmbeddingReady struct {
	Provider string  `json:"provider"`
	Model    string  `json:"model"`
	Embedded int     `json:"embedded"`
	Missing  int     `json:"missing"`
	Coverage float64 `json:"coverage"`
}

// Run performs the configured audits against the engine and returns a Report.
// projectPath, vaultPath, dbPath, configPath are filled by the caller so the
// API server can reuse paths it already knows; Run only touches the engine.
func Run(ctx context.Context, e *recall.Engine, opts Options, projectPath, vaultPath, dbPath, configPath string) Report {
	report := Report{OK: true, ProjectPath: projectPath, VaultPath: vaultPath, DBPath: dbPath, ConfigPath: configPath, Errors: []string{}}

	if projectPath != "" {
		if info, err := os.Stat(projectPath); err != nil || !info.IsDir() {
			report.OK = false
			report.Errors = append(report.Errors, fmt.Sprintf("project path missing or not directory: %s", projectPath))
		}
	}
	if vaultPath != "" {
		if info, err := os.Stat(vaultPath); err != nil || !info.IsDir() {
			report.OK = false
			report.Errors = append(report.Errors, fmt.Sprintf("vault missing or not directory: %s", vaultPath))
		}
	}
	if dbPath != "" {
		if _, err := os.Stat(dbPath); err != nil {
			report.OK = false
			report.Errors = append(report.Errors, fmt.Sprintf("db missing: %s", dbPath))
		}
	}

	if e == nil {
		report.OK = false
		report.Errors = append(report.Errors, "engine not open")
		return report
	}

	if domains, err := e.Vault().ListDomains(); err != nil {
		report.OK = false
		report.Errors = append(report.Errors, err.Error())
	} else {
		report.Domains = len(domains)
	}
	if count, err := e.MemoryCount(ctx); err != nil {
		report.OK = false
		report.Errors = append(report.Errors, err.Error())
	} else {
		report.Memories = count
	}

	if opts.Deep {
		auditDeep(ctx, e, &report)
	}
	if opts.Embeddings {
		auditEmbeddings(ctx, e, &report, opts.Provider, opts.Model)
	}
	return report
}

func auditDeep(ctx context.Context, e *recall.Engine, report *Report) {
	paths, err := e.Vault().Scan()
	if err != nil {
		report.OK = false
		report.Errors = append(report.Errors, err.Error())
		return
	}
	vaultIDs := map[string]string{}
	for _, rel := range paths {
		m, err := e.Vault().Read(rel)
		if err != nil {
			report.OK = false
			report.InvalidFiles = append(report.InvalidFiles, InvalidFile{Path: rel, Error: err.Error()})
			continue
		}
		vaultIDs[m.ID] = rel
	}

	ids, err := e.IndexedIDs(ctx)
	if err != nil {
		report.OK = false
		report.Errors = append(report.Errors, err.Error())
		return
	}
	report.VaultMemories = len(vaultIDs)
	report.IndexMemories = len(ids)

	for _, id := range ids {
		rel, err := e.IndexedPath(ctx, id)
		if err != nil {
			report.OK = false
			report.Errors = append(report.Errors, err.Error())
			continue
		}
		if _, ok := vaultIDs[id]; !ok {
			report.OK = false
			report.StaleIndexIDs = append(report.StaleIndexIDs, id)
			report.MissingIndexPaths = append(report.MissingIndexPaths, MissingIndex{ID: id, Path: rel})
		}
	}
	sort.Strings(report.StaleIndexIDs)
	sort.Slice(report.MissingIndexPaths, func(i, j int) bool { return report.MissingIndexPaths[i].ID < report.MissingIndexPaths[j].ID })
	sort.Slice(report.InvalidFiles, func(i, j int) bool { return report.InvalidFiles[i].Path < report.InvalidFiles[j].Path })
}

func auditEmbeddings(ctx context.Context, e *recall.Engine, report *Report, provider, model string) {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		provider = "ollama"
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = embedding.DefaultOllamaModel
	}
	ids, err := e.IndexedIDs(ctx)
	if err != nil {
		report.OK = false
		report.Errors = append(report.Errors, err.Error())
		return
	}
	embs, err := e.Embeddings(ctx, provider, model)
	if err != nil {
		report.OK = false
		report.Errors = append(report.Errors, err.Error())
		return
	}
	embeddedIDs := map[string]struct{}{}
	for _, emb := range embs {
		embeddedIDs[emb.MemoryID] = struct{}{}
	}
	missing := 0
	for _, id := range ids {
		if _, ok := embeddedIDs[id]; !ok {
			missing++
		}
	}
	coverage := 1.0
	if len(ids) > 0 {
		coverage = float64(len(ids)-missing) / float64(len(ids))
	}
	if missing > 0 {
		report.OK = false
	}
	report.Embeddings = &EmbeddingReady{Provider: provider, Model: model, Embedded: len(ids) - missing, Missing: missing, Coverage: coverage}
}

// JoinDBPath returns the conventional db path for a project root.
func JoinDBPath(projectPath string) string {
	return filepath.Join(projectPath, "db", "recall.sqlite")
}

// JoinVaultPath returns the conventional vault path for a project root.
func JoinVaultPath(projectPath string) string {
	return filepath.Join(projectPath, "vault")
}

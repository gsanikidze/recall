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
	Deep          bool   // audit vault/index drift and invalid memory files
	Embeddings    bool   // report embedding coverage for indexed memories
	Provider      string // embedding provider for Embeddings
	Model         string // embedding model for Embeddings
	Fix           bool   // run deterministic safe repairs before the final audit
	FixEmbeddings bool   // when Fix is true, generate missing embeddings too
}

// FixResult records one repair action run by doctor --fix.
type FixResult struct {
	Action   string `json:"action"`
	OK       bool   `json:"ok"`
	Message  string `json:"message,omitempty"`
	Indexed  int    `json:"indexed,omitempty"`
	Deleted  int    `json:"deleted,omitempty"`
	Embedded int    `json:"embedded,omitempty"`
	Skipped  int    `json:"skipped,omitempty"`
	Failed   int    `json:"failed,omitempty"`
}

// Report is the audit result. JSON tags are stable: CLI tests assert on them.
type Report struct {
	OK                  bool                 `json:"ok"`
	ProjectPath         string               `json:"project_path"`
	ConfigPath          string               `json:"config_path"`
	VaultPath           string               `json:"vault_path"`
	DBPath              string               `json:"db_path"`
	Domains             int                  `json:"domains"`
	Memories            int                  `json:"memories"`
	VaultMemories       int                  `json:"vault_memories,omitempty"`
	IndexMemories       int                  `json:"index_memories,omitempty"`
	InvalidFiles        []InvalidFile        `json:"invalid_files,omitempty"`
	StaleIndexIDs       []string             `json:"stale_index_ids,omitempty"`
	MissingIndexPaths   []MissingIndex       `json:"missing_index_paths,omitempty"`
	UnindexedVaultFiles []UnindexedVaultFile `json:"unindexed_vault_files,omitempty"`
	DuplicateVaultIDs   []DuplicateVaultID   `json:"duplicate_vault_ids,omitempty"`
	Embeddings          *EmbeddingReady      `json:"embeddings,omitempty"`
	Fixes               []FixResult          `json:"fixes,omitempty"`
	Suggestions         []Suggestion         `json:"suggestions,omitempty"`
	Errors              []string             `json:"errors"`
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

// UnindexedVaultFile is a valid vault memory file absent from SQLite.
type UnindexedVaultFile struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

// DuplicateVaultID is a memory id appearing in more than one vault file.
type DuplicateVaultID struct {
	ID    string   `json:"id"`
	Paths []string `json:"paths"`
}

// Suggestion is a copy-pasteable prompt the user can hand to an AI agent to
// fix a specific doctor issue. Severity is "error" (blocks core function) or
// "warning" (degraded but usable).
type Suggestion struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Severity string `json:"severity"`
	Prompt   string `json:"prompt"`
}

// EmbeddingReady summarises embedding coverage for indexed memories plus a
// live probe of the embedding backend (server reachable, model pulled).
type EmbeddingReady struct {
	Provider            string   `json:"provider"`
	Model               string   `json:"model"`
	ServerURL           string   `json:"server_url,omitempty"`
	Reachable           bool     `json:"reachable"`
	ModelAvailable      bool     `json:"model_available"`
	ServerError         string   `json:"server_error,omitempty"`
	AvailableModels     []string `json:"available_models,omitempty"`
	Embedded            int      `json:"embedded"`
	Missing             int      `json:"missing"`
	Coverage            float64  `json:"coverage"`
	MissingEmbeddingIDs []string `json:"missing_embedding_ids,omitempty"`
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

	if opts.Fix {
		report.Fixes = repair(ctx, e, opts)
	}
	if opts.Deep {
		auditDeep(ctx, e, &report)
	}
	if opts.Embeddings || opts.FixEmbeddings {
		auditEmbeddings(ctx, e, &report, opts.Provider, opts.Model)
	}
	report.Suggestions = buildSuggestions(&report)
	return report
}

func repair(ctx context.Context, e *recall.Engine, opts Options) []FixResult {
	var fixes []FixResult
	stats, err := e.Reindex(ctx)
	fix := FixResult{Action: "reindex"}
	if err != nil {
		fix.OK = false
		fix.Message = err.Error()
	} else {
		fix.OK = true
		fix.Indexed = stats.Indexed
		fix.Deleted = stats.Deleted
	}
	fixes = append(fixes, fix)

	if opts.FixEmbeddings {
		provider, model := normalizeEmbeddingProviderModel(opts.Provider, opts.Model)
		p, err := embedding.NewProvider(provider, model, "")
		embedFix := FixResult{Action: "embed"}
		if err != nil {
			embedFix.OK = false
			embedFix.Message = err.Error()
		} else {
			embedStats, err := e.EmbedAll(ctx, p, false)
			if err != nil {
				embedFix.OK = false
				embedFix.Message = err.Error()
			} else {
				embedFix.OK = true
				embedFix.Embedded = embedStats.Embedded
				embedFix.Skipped = embedStats.Skipped
				embedFix.Failed = embedStats.Failed
			}
		}
		fixes = append(fixes, embedFix)
	}
	return fixes
}

func normalizeEmbeddingProviderModel(provider, model string) (string, string) {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		provider = "ollama"
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = embedding.DefaultOllamaModel
	}
	return provider, model
}

func auditDeep(ctx context.Context, e *recall.Engine, report *Report) {
	paths, err := e.Vault().Scan()
	if err != nil {
		report.OK = false
		report.Errors = append(report.Errors, err.Error())
		return
	}
	vaultIDs := map[string]string{}
	vaultPathsByID := map[string][]string{}
	validVaultFiles := 0
	for _, rel := range paths {
		m, err := e.Vault().Read(rel)
		if err != nil {
			report.OK = false
			report.InvalidFiles = append(report.InvalidFiles, InvalidFile{Path: rel, Error: err.Error()})
			continue
		}
		validVaultFiles++
		vaultIDs[m.ID] = rel
		vaultPathsByID[m.ID] = append(vaultPathsByID[m.ID], rel)
	}

	ids, err := e.IndexedIDs(ctx)
	if err != nil {
		report.OK = false
		report.Errors = append(report.Errors, err.Error())
		return
	}
	report.VaultMemories = validVaultFiles
	report.IndexMemories = len(ids)
	indexedIDs := map[string]struct{}{}
	for _, id := range ids {
		indexedIDs[id] = struct{}{}
	}
	for id, rel := range vaultIDs {
		if _, ok := indexedIDs[id]; !ok {
			report.OK = false
			report.UnindexedVaultFiles = append(report.UnindexedVaultFiles, UnindexedVaultFile{ID: id, Path: rel})
		}
	}
	for id, vaultPaths := range vaultPathsByID {
		if len(vaultPaths) > 1 {
			report.OK = false
			sort.Strings(vaultPaths)
			report.DuplicateVaultIDs = append(report.DuplicateVaultIDs, DuplicateVaultID{ID: id, Paths: vaultPaths})
		}
	}

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
	sort.Slice(report.UnindexedVaultFiles, func(i, j int) bool { return report.UnindexedVaultFiles[i].Path < report.UnindexedVaultFiles[j].Path })
	sort.Slice(report.DuplicateVaultIDs, func(i, j int) bool { return report.DuplicateVaultIDs[i].ID < report.DuplicateVaultIDs[j].ID })
	sort.Slice(report.InvalidFiles, func(i, j int) bool { return report.InvalidFiles[i].Path < report.InvalidFiles[j].Path })
}

func auditEmbeddings(ctx context.Context, e *recall.Engine, report *Report, provider, model string) {
	provider, model = normalizeEmbeddingProviderModel(provider, model)

	ready := &EmbeddingReady{Provider: provider, Model: model}

	// Probe the backend if the provider supports it. Ollama exposes server
	// reachability + model availability; fake/test providers skip the probe
	// and report as reachable so existing tests keep passing.
	p, pErr := embedding.NewProvider(provider, model, "")
	if pErr != nil {
		report.OK = false
		report.Errors = append(report.Errors, pErr.Error())
		report.Embeddings = ready
		return
	}
	if prober, ok := p.(embedding.Prober); ok {
		probe, _ := prober.Probe(ctx)
		if probe != nil {
			ready.Reachable = probe.Reachable
			ready.ModelAvailable = probe.ModelAvailable
			ready.ServerError = probe.ServerError
			ready.AvailableModels = probe.AvailableModels
			if o, ok := p.(interface{ BaseURL() string }); ok {
				ready.ServerURL = o.BaseURL()
			}
			if !probe.Reachable || !probe.ModelAvailable {
				report.OK = false
			}
		}
	} else {
		ready.Reachable = true
		ready.ModelAvailable = true
	}

	ids, err := e.IndexedIDs(ctx)
	if err != nil {
		report.OK = false
		report.Errors = append(report.Errors, err.Error())
		report.Embeddings = ready
		return
	}
	embs, err := e.Embeddings(ctx, provider, model)
	if err != nil {
		report.OK = false
		report.Errors = append(report.Errors, err.Error())
		report.Embeddings = ready
		return
	}
	embeddedIDs := map[string]struct{}{}
	for _, emb := range embs {
		embeddedIDs[emb.MemoryID] = struct{}{}
	}
	missing := 0
	var missingIDs []string
	for _, id := range ids {
		if _, ok := embeddedIDs[id]; !ok {
			missing++
			if len(missingIDs) < 20 {
				missingIDs = append(missingIDs, id)
			}
		}
	}
	sort.Strings(missingIDs)
	coverage := 1.0
	if len(ids) > 0 {
		coverage = float64(len(ids)-missing) / float64(len(ids))
	}
	if missing > 0 {
		report.OK = false
	}
	ready.Embedded = len(ids) - missing
	ready.Missing = missing
	ready.Coverage = coverage
	ready.MissingEmbeddingIDs = missingIDs
	report.Embeddings = ready
}

// buildSuggestions inspects the report and returns copy-pasteable agent prompts
// for each issue found. Order: fatal path/db issues first, then vault/index
// drift, then embedding backend, then coverage gaps, then generic errors.
func buildSuggestions(r *Report) []Suggestion {
	var out []Suggestion

	// --- Project path ---
	if r.ProjectPath != "" {
		if info, err := os.Stat(r.ProjectPath); err != nil || !info.IsDir() {
			out = append(out, Suggestion{
				ID:       "project-missing",
				Title:    "Project directory missing",
				Severity: "error",
				Prompt: fmt.Sprintf(
					"The Recall project directory is missing or not a directory at %q. "+
						"Re-initialize it with `recall init --path %s --force`, or update the config at %q to point to the correct project path.",
					r.ProjectPath, r.ProjectPath, r.ConfigPath,
				),
			})
		}
	}

	// --- Vault path ---
	if r.VaultPath != "" {
		if info, err := os.Stat(r.VaultPath); err != nil || !info.IsDir() {
			out = append(out, Suggestion{
				ID:       "vault-missing",
				Title:    "Vault directory missing",
				Severity: "error",
				Prompt: fmt.Sprintf(
					"The Recall vault directory is missing at %q. "+
						"Recreate it (`mkdir -p %s`) and run `recall reindex` to rebuild the SQLite index from the vault, "+
						"or re-initialize the project with `recall init --path %s --force`.",
					r.VaultPath, r.VaultPath, r.ProjectPath,
				),
			})
		}
	}

	// --- DB path ---
	if r.DBPath != "" {
		if _, err := os.Stat(r.DBPath); err != nil {
			out = append(out, Suggestion{
				ID:       "db-missing",
				Title:    "SQLite database missing",
				Severity: "error",
				Prompt: fmt.Sprintf(
					"The Recall SQLite database is missing at %q. "+
						"Run `recall reindex` to rebuild it from the vault markdown files.",
					r.DBPath,
				),
			})
		}
	}

	// --- Invalid vault files ---
	if len(r.InvalidFiles) > 0 {
		var lines []string
		for _, f := range r.InvalidFiles {
			lines = append(lines, fmt.Sprintf("  - %s: %s", f.Path, f.Error))
		}
		out = append(out, Suggestion{
			ID:       "invalid-files",
			Title:    fmt.Sprintf("%d invalid vault file(s)", len(r.InvalidFiles)),
			Severity: "warning",
			Prompt: fmt.Sprintf(
				"The following Recall vault files failed to parse:\n%s\n"+
					"Inspect each file, fix the markdown/YAML frontmatter (common issues: missing or malformed `id`, `title`, `domain` fields), "+
					"then run `recall reindex` to rebuild the index.",
				strings.Join(lines, "\n"),
			),
		})
	}

	// --- Stale index / missing vault files ---
	if len(r.MissingIndexPaths) > 0 {
		out = append(out, Suggestion{
			ID:       "stale-index",
			Title:    fmt.Sprintf("%d stale index row(s)", len(r.MissingIndexPaths)),
			Severity: "warning",
			Prompt: fmt.Sprintf(
				"The Recall SQLite index has %d entries pointing to vault files that no longer exist. "+
					"Run `recall reindex` to rebuild the index from the current vault. "+
					"If those memories were deleted intentionally, reindexing will clean up the stale rows.",
				len(r.MissingIndexPaths),
			),
		})
	}

	// --- Vault/index drift: valid vault files absent from SQLite or duplicate ids ---
	if len(r.UnindexedVaultFiles) > 0 || len(r.DuplicateVaultIDs) > 0 {
		out = append(out, Suggestion{
			ID:       "vault-index-drift",
			Title:    "Vault/index drift detected",
			Severity: "warning",
			Prompt: fmt.Sprintf(
				"Recall found %d valid vault file(s) absent from SQLite and %d duplicate vault id(s). "+
					"Run `recall reindex` to rebuild the SQLite index from the vault. If duplicate ids remain, edit or remove duplicate vault files first, then run `recall reindex` again.",
				len(r.UnindexedVaultFiles), len(r.DuplicateVaultIDs),
			),
		})
	}

	// --- Embedding backend ---
	if r.Embeddings != nil {
		emb := r.Embeddings
		if !emb.Reachable {
			url := emb.ServerURL
			if url == "" {
				url = "http://127.0.0.1:11434"
			}
			out = append(out, Suggestion{
				ID:       "ollama-unreachable",
				Title:    "Ollama server unreachable",
				Severity: "error",
				Prompt: fmt.Sprintf(
					"The Ollama embedding server is not reachable at %s. "+
						"Start it with `ollama serve` in a separate terminal, "+
						"or install Ollama from https://ollama.com if it is not installed. "+
						"Then verify with `recall doctor --embeddings`.",
					url,
				),
			})
		} else if !emb.ModelAvailable {
			out = append(out, Suggestion{
				ID:       "model-not-pulled",
				Title:    fmt.Sprintf("Model %q not pulled", emb.Model),
				Severity: "error",
				Prompt: fmt.Sprintf(
					"The embedding model %q is not pulled in Ollama. "+
						"Run `ollama pull %s` to download it, then verify with `recall doctor --embeddings`.",
					emb.Model, emb.Model,
				),
			})
		}

		// --- Coverage gaps (only when backend is healthy) ---
		if emb.Reachable && emb.ModelAvailable && emb.Missing > 0 {
			out = append(out, Suggestion{
				ID:       "embeddings-missing",
				Title:    fmt.Sprintf("%d memories missing embeddings", emb.Missing),
				Severity: "warning",
				Prompt: fmt.Sprintf(
					"%d indexed memories are missing embedding vectors (coverage: %.0f%%). "+
						"Run `recall embed --provider %s --model %s` to generate embeddings for all indexed memories.",
					emb.Missing, emb.Coverage*100, emb.Provider, emb.Model,
				),
			})
		}
	}

	// --- Generic errors not covered above ---
	for _, e := range r.Errors {
		// Skip errors we already surfaced as specific suggestions
		if strings.Contains(e, "project path") || strings.Contains(e, "vault missing") || strings.Contains(e, "db missing") || strings.Contains(e, "engine not open") {
			continue
		}
		out = append(out, Suggestion{
			ID:       "error-" + fmt.Sprintf("%d", len(out)),
			Title:    "Error",
			Severity: "error",
			Prompt:   fmt.Sprintf("Recall doctor reported an error: %s. Investigate the cause and fix it, then re-run `recall doctor`.", e),
		})
	}

	return out
}

// JoinDBPath returns the conventional db path for a project root.
func JoinDBPath(projectPath string) string {
	return filepath.Join(projectPath, "db", "recall.sqlite")
}

// JoinVaultPath returns the conventional vault path for a project root.
func JoinVaultPath(projectPath string) string {
	return filepath.Join(projectPath, "vault")
}

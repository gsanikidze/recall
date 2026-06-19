package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"recall/internal/embedding"
	"recall/internal/recall"
)

type doctorReport struct {
	OK                bool                      `json:"ok"`
	ProjectPath       string                    `json:"project_path"`
	ConfigPath        string                    `json:"config_path"`
	VaultPath         string                    `json:"vault_path"`
	DBPath            string                    `json:"db_path"`
	Domains           int                       `json:"domains"`
	Memories          int                       `json:"memories"`
	VaultMemories     int                       `json:"vault_memories,omitempty"`
	IndexMemories     int                       `json:"index_memories,omitempty"`
	InvalidFiles      []doctorInvalidFile       `json:"invalid_files,omitempty"`
	StaleIndexIDs     []string                  `json:"stale_index_ids,omitempty"`
	MissingIndexPaths []doctorMissingIndexPath  `json:"missing_index_paths,omitempty"`
	Embeddings        *doctorEmbeddingReadiness `json:"embeddings,omitempty"`
	Errors            []string                  `json:"errors"`
}

type doctorInvalidFile struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

type doctorMissingIndexPath struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

type doctorEmbeddingReadiness struct {
	Provider string  `json:"provider"`
	Model    string  `json:"model"`
	Embedded int     `json:"embedded"`
	Missing  int     `json:"missing"`
	Coverage float64 `json:"coverage"`
}

func Doctor(args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "print JSON")
	deep := fs.Bool("deep", false, "audit vault/index drift and invalid memory files")
	embeddings := fs.Bool("embeddings", false, "report embedding coverage for indexed memories")
	provider := fs.String("provider", "ollama", "embedding provider for --embeddings")
	model := fs.String("model", embedding.DefaultOllamaModel, "embedding model for --embeddings")
	if err := fs.Parse(args); err != nil {
		return err
	}

	report := doctorReport{OK: true}
	cfgPath, _ := configPath()
	report.ConfigPath = cfgPath

	projectPath, err := currentProjectPath()
	if err != nil {
		report.OK = false
		report.Errors = append(report.Errors, err.Error())
		if *jsonOut {
			return printJSON(report)
		}
		return printDoctor(report)
	}
	report.ProjectPath = projectPath
	report.VaultPath = filepath.Join(projectPath, "vault")
	report.DBPath = filepath.Join(projectPath, "db", "recall.sqlite")

	if info, err := os.Stat(report.ProjectPath); err != nil || !info.IsDir() {
		report.OK = false
		report.Errors = append(report.Errors, fmt.Sprintf("project path missing or not directory: %s", report.ProjectPath))
	}
	if info, err := os.Stat(report.VaultPath); err != nil || !info.IsDir() {
		report.OK = false
		report.Errors = append(report.Errors, fmt.Sprintf("vault missing or not directory: %s", report.VaultPath))
	}

	e, err := openEngine()
	if err != nil {
		report.OK = false
		report.Errors = append(report.Errors, err.Error())
	} else {
		defer e.Close()
		ctx := context.Background()
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
		if _, err := os.Stat(report.DBPath); err != nil {
			report.OK = false
			report.Errors = append(report.Errors, fmt.Sprintf("db missing: %s", report.DBPath))
		}
		if *deep {
			auditDoctorDeep(ctx, e, &report)
		}
		if *embeddings {
			auditDoctorEmbeddings(ctx, e, &report, strings.TrimSpace(*provider), strings.TrimSpace(*model))
		}
	}

	if *jsonOut {
		return printJSON(report)
	}
	return printDoctor(report)
}

func auditDoctorDeep(ctx context.Context, e *recall.Engine, report *doctorReport) {
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
			report.InvalidFiles = append(report.InvalidFiles, doctorInvalidFile{Path: rel, Error: err.Error()})
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
			report.MissingIndexPaths = append(report.MissingIndexPaths, doctorMissingIndexPath{ID: id, Path: rel})
		}
	}
	sort.Strings(report.StaleIndexIDs)
	sort.Slice(report.MissingIndexPaths, func(i, j int) bool { return report.MissingIndexPaths[i].ID < report.MissingIndexPaths[j].ID })
	sort.Slice(report.InvalidFiles, func(i, j int) bool { return report.InvalidFiles[i].Path < report.InvalidFiles[j].Path })
}

func auditDoctorEmbeddings(ctx context.Context, e *recall.Engine, report *doctorReport, provider, model string) {
	if provider == "" {
		provider = "ollama"
	}
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
	report.Embeddings = &doctorEmbeddingReadiness{Provider: provider, Model: model, Embedded: len(ids) - missing, Missing: missing, Coverage: coverage}
}

func currentProjectPath() (string, error) {
	if override := strings.TrimSpace(os.Getenv("RECALL_PROJECT")); override != "" {
		return resolvePath(override)
	}
	if override := strings.TrimSpace(os.Getenv("RECALL_HOME")); override != "" {
		return resolvePath(override)
	}
	cfg, found, err := loadConfig()
	if err != nil {
		return "", err
	}
	if !found || cfg.ProjectPath == "" {
		return "", fmt.Errorf("recall is not initialized; run \"recall init\" first")
	}
	return cfg.ProjectPath, nil
}

func printDoctor(r doctorReport) error {
	status := "ok"
	if !r.OK {
		status = "failed"
	}
	fmt.Printf("recall doctor: %s\n", status)
	fmt.Printf("project: %s\n", r.ProjectPath)
	fmt.Printf("config:  %s\n", r.ConfigPath)
	fmt.Printf("vault:   %s\n", r.VaultPath)
	fmt.Printf("db:      %s\n", r.DBPath)
	fmt.Printf("domains: %d\n", r.Domains)
	fmt.Printf("memories: %d\n", r.Memories)
	if r.VaultMemories > 0 || r.IndexMemories > 0 || len(r.InvalidFiles) > 0 || len(r.StaleIndexIDs) > 0 {
		fmt.Printf("vault memories: %d\n", r.VaultMemories)
		fmt.Printf("index memories: %d\n", r.IndexMemories)
		for _, invalid := range r.InvalidFiles {
			fmt.Printf("invalid file: %s: %s\n", invalid.Path, invalid.Error)
		}
		for _, missing := range r.MissingIndexPaths {
			fmt.Printf("missing indexed file: %s %s\n", missing.ID, missing.Path)
		}
	}
	if r.Embeddings != nil {
		fmt.Printf("embeddings: %s/%s embedded=%d missing=%d coverage=%.2f\n", r.Embeddings.Provider, r.Embeddings.Model, r.Embeddings.Embedded, r.Embeddings.Missing, r.Embeddings.Coverage)
	}
	for _, err := range r.Errors {
		fmt.Printf("error:   %s\n", err)
	}
	return nil
}

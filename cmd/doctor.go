package cmd

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"recall/internal/doctor"
	"recall/internal/embedding"
)

func Doctor(args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "print JSON")
	deep := fs.Bool("deep", false, "audit vault/index drift and invalid memory files")
	embeddings := fs.Bool("embeddings", false, "report embedding coverage for indexed memories")
	fix := fs.Bool("fix", false, "run deterministic safe repairs before reporting")
	fixEmbeddings := fs.Bool("fix-embeddings", false, "with --fix, generate missing embeddings too")
	provider := fs.String("provider", "ollama", "embedding provider for --embeddings or --fix-embeddings")
	model := fs.String("model", embedding.DefaultOllamaModel, "embedding model for --embeddings or --fix-embeddings")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfgPath, _ := configPath()
	projectPath, err := currentProjectPath()
	if err != nil {
		report := doctor.Report{OK: false, ConfigPath: cfgPath, Errors: []string{err.Error()}}
		if *jsonOut {
			return printJSON(report)
		}
		return printDoctor(report)
	}
	vaultPath := doctor.JoinVaultPath(projectPath)
	dbPath := doctor.JoinDBPath(projectPath)

	e, err := openEngine()
	if err != nil {
		report := doctor.Run(context.Background(), nil, doctor.Options{}, projectPath, vaultPath, dbPath, cfgPath)
		report.Errors = append(report.Errors, err.Error())
		if *jsonOut {
			return printJSON(report)
		}
		return printDoctor(report)
	}
	defer e.Close()

	report := doctor.Run(context.Background(), e, doctor.Options{
		Deep: *deep, Embeddings: *embeddings, Provider: *provider, Model: *model,
		Fix: *fix, FixEmbeddings: *fixEmbeddings,
	}, projectPath, vaultPath, dbPath, cfgPath)

	if *jsonOut {
		return printJSON(report)
	}
	return printDoctor(report)
}

func printDoctor(r doctor.Report) error {
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
	if r.VaultMemories > 0 || r.IndexMemories > 0 || len(r.InvalidFiles) > 0 || len(r.StaleIndexIDs) > 0 || len(r.UnindexedVaultFiles) > 0 || len(r.DuplicateVaultIDs) > 0 {
		fmt.Printf("vault memories: %d\n", r.VaultMemories)
		fmt.Printf("index memories: %d\n", r.IndexMemories)
		for _, invalid := range r.InvalidFiles {
			fmt.Printf("invalid file: %s: %s\n", invalid.Path, invalid.Error)
		}
		for _, missing := range r.MissingIndexPaths {
			fmt.Printf("missing indexed file: %s %s\n", missing.ID, missing.Path)
		}
		for _, unindexed := range r.UnindexedVaultFiles {
			fmt.Printf("unindexed vault file: %s %s\n", unindexed.ID, unindexed.Path)
		}
		for _, dup := range r.DuplicateVaultIDs {
			fmt.Printf("duplicate vault id: %s %s\n", dup.ID, strings.Join(dup.Paths, ", "))
		}
	}
	for _, fix := range r.Fixes {
		fmt.Printf("fix: %s ok=%v", fix.Action, fix.OK)
		if fix.Indexed > 0 || fix.Deleted > 0 {
			fmt.Printf(" indexed=%d deleted=%d", fix.Indexed, fix.Deleted)
		}
		if fix.Embedded > 0 || fix.Skipped > 0 || fix.Failed > 0 {
			fmt.Printf(" embedded=%d skipped=%d failed=%d", fix.Embedded, fix.Skipped, fix.Failed)
		}
		if fix.Message != "" {
			fmt.Printf(" message=%s", fix.Message)
		}
		fmt.Println()
	}
	if r.Embeddings != nil {
		fmt.Printf("embeddings: %s/%s", r.Embeddings.Provider, r.Embeddings.Model)
		if r.Embeddings.ServerURL != "" {
			fmt.Printf(" server=%s", r.Embeddings.ServerURL)
		}
		fmt.Printf(" reachable=%v model_available=%v embedded=%d missing=%d coverage=%.2f\n",
			r.Embeddings.Reachable, r.Embeddings.ModelAvailable, r.Embeddings.Embedded, r.Embeddings.Missing, r.Embeddings.Coverage)
		if len(r.Embeddings.MissingEmbeddingIDs) > 0 {
			fmt.Printf("missing embedding ids (first %d of %d):\n", len(r.Embeddings.MissingEmbeddingIDs), r.Embeddings.Missing)
			for _, id := range r.Embeddings.MissingEmbeddingIDs {
				fmt.Printf("  %s\n", id)
			}
		}
		if r.Embeddings.ServerError != "" {
			fmt.Printf("embedding server: %s\n", r.Embeddings.ServerError)
		}
		if len(r.Embeddings.AvailableModels) > 0 {
			fmt.Printf("embedding models: %s\n", strings.Join(r.Embeddings.AvailableModels, ", "))
		}
	}
	for _, err := range r.Errors {
		fmt.Printf("error:   %s\n", err)
	}
	for _, s := range r.Suggestions {
		fmt.Printf("\n--- suggestion: %s [%s] ---\n%s\n", s.Title, s.Severity, s.Prompt)
	}
	return nil
}

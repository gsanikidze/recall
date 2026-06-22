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
	provider := fs.String("provider", "ollama", "embedding provider for --embeddings")
	model := fs.String("model", embedding.DefaultOllamaModel, "embedding model for --embeddings")
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
		fmt.Printf("embeddings: %s/%s", r.Embeddings.Provider, r.Embeddings.Model)
		if r.Embeddings.ServerURL != "" {
			fmt.Printf(" server=%s", r.Embeddings.ServerURL)
		}
		fmt.Printf(" reachable=%v model_available=%v embedded=%d missing=%d coverage=%.2f\n",
			r.Embeddings.Reachable, r.Embeddings.ModelAvailable, r.Embeddings.Embedded, r.Embeddings.Missing, r.Embeddings.Coverage)
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
	return nil
}

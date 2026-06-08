package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"recall/internal/index"
)

type doctorReport struct {
	OK          bool     `json:"ok"`
	ProjectPath string   `json:"project_path"`
	ConfigPath  string   `json:"config_path"`
	VaultPath   string   `json:"vault_path"`
	DBPath      string   `json:"db_path"`
	Domains     int      `json:"domains"`
	Memories    int      `json:"memories"`
	Errors      []string `json:"errors"`
}

func Doctor(args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "print JSON")
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
		if domains, err := e.Vault().ListDomains(); err != nil {
			report.OK = false
			report.Errors = append(report.Errors, err.Error())
		} else {
			report.Domains = len(domains)
		}
		if hits, err := e.Search(context.Background(), index.Filter{Limit: 1, IncludeExpired: true}); err != nil {
			report.OK = false
			report.Errors = append(report.Errors, err.Error())
		} else {
			report.Memories = len(hits)
		}
		if _, err := os.Stat(report.DBPath); err != nil {
			report.OK = false
			report.Errors = append(report.Errors, fmt.Sprintf("db missing: %s", report.DBPath))
		}
	}

	if *jsonOut {
		return printJSON(report)
	}
	return printDoctor(report)
}

func currentProjectPath() (string, error) {
	if override := os.Getenv("RECALL_PROJECT"); override != "" {
		return resolvePath(override)
	}
	if override := os.Getenv("RECALL_HOME"); override != "" {
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
	for _, err := range r.Errors {
		fmt.Printf("error:   %s\n", err)
	}
	return nil
}

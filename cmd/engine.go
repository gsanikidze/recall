package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"recall/internal/memory"
	"recall/internal/recall"
	"recall/internal/vault"
)

// openEngine loads the saved project path and opens the recall engine.
func openEngine() (*recall.Engine, error) {
	if override := strings.TrimSpace(os.Getenv("RECALL_PROJECT")); override != "" {
		path, err := resolvePath(override)
		if err != nil {
			return nil, err
		}
		return recall.Open(path)
	}
	if override := strings.TrimSpace(os.Getenv("RECALL_HOME")); override != "" {
		path, err := resolvePath(override)
		if err != nil {
			return nil, err
		}
		return recall.Open(path)
	}
	cfg, found, err := loadConfig()
	if err != nil {
		return nil, err
	}
	if !found || cfg.ProjectPath == "" {
		return nil, fmt.Errorf("recall is not initialized; run \"recall init\" first")
	}
	return recall.Open(cfg.ProjectPath)
}

// splitList parses a comma-separated flag value into a trimmed, non-empty slice.
func splitList(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var out []string
	for part := range strings.SplitSeq(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

type memoryJSONOutput struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Domain    string   `json:"domain"`
	Tags      []string `json:"tags"`
	Project   string   `json:"project"`
	Lifecycle string   `json:"lifecycle"`
	ExpiresOn string   `json:"expires_on"`
	Created   string   `json:"created"`
	Updated   string   `json:"updated"`
	Source    string   `json:"source"`
	Links     []string `json:"links"`
	Path      string   `json:"path"`
	Body      string   `json:"body"`
}

type domainOutput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func domainOutputs(domains []vault.Domain) []domainOutput {
	out := make([]domainOutput, 0, len(domains))
	for _, d := range domains {
		out = append(out, domainOutput{Name: d.Name, Description: d.Description})
	}
	return out
}

func memoryOutput(m memory.Memory, relPath string) memoryJSONOutput {
	tags := m.Tags
	if tags == nil {
		tags = []string{}
	}
	links := m.Links
	if links == nil {
		links = []string{}
	}
	return memoryJSONOutput{
		ID: m.ID, Title: m.Title, Domain: m.Domain, Tags: tags, Project: m.Project,
		Lifecycle: string(m.Lifecycle), ExpiresOn: m.ExpiresOn.String(),
		Created: m.Created.String(), Updated: m.Updated.String(), Source: m.Source,
		Links: links, Path: relPath, Body: m.Body,
	}
}

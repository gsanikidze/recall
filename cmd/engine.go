package cmd

import (
	"encoding/json"
	"os"
	"strings"

	"recall/internal/recall"
	"recall/internal/vault"
)

// openEngine loads the saved project path and opens the recall engine.
func openEngine() (*recall.Engine, error) {
	path, err := currentProjectPath()
	if err != nil {
		return nil, err
	}
	return recall.Open(path)
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

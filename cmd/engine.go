package cmd

import (
	"fmt"
	"strings"

	"recall/internal/recall"
)

// openEngine loads the saved project path and opens the recall engine.
func openEngine() (*recall.Engine, error) {
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

package main

import (
	"os"
	"strings"
	"testing"
)

func TestAgentInstructionArtifactsExist(t *testing.T) {
	cases := []struct {
		path     string
		requires []string
	}{
		{
			path: "docs/agent-instructions.md",
			requires: []string{
				"# Recall agent instructions",
				"Use Recall before answering questions about previous projects, decisions, people, opportunities, durable research, tools",
				"recall_search",
				"recall_get",
				"recall_list_domains",
				"recall_add",
				"Never store secrets, raw chat logs, generic Q&A, or temporary task progress.",
			},
		},
		{
			path: "docs/templates/AGENTS.md",
			requires: []string{
				"# Agent memory policy",
				"Recall MCP",
				"Read flow",
				"Write flow",
			},
		},
		{
			path: "skills/recall-memory/SKILL.md",
			requires: []string{
				"name: recall-memory",
				"description:",
				"Recall MCP",
				"Read flow",
				"Write flow",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			b, err := os.ReadFile(tc.path)
			if err != nil {
				t.Fatalf("read %s: %v", tc.path, err)
			}
			content := string(b)
			for _, required := range tc.requires {
				if !strings.Contains(content, required) {
					t.Fatalf("%s missing required text %q", tc.path, required)
				}
			}
		})
	}
}

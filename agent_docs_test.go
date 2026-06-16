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
			path: "AGENTS.md",
			requires: []string{
				"# Recall agent setup",
				"make build-nui",
				"hermes mcp add recall",
				"hermes mcp test recall",
				"hermes skills install",
				"/reload-mcp",
				"/reload-skills",
			},
		},
		{
			path: "docs/llm-setup.md",
			requires: []string{
				"# LLM setup for Recall MCP and skills",
				"RECALL_PROJECT",
				"hermes mcp add recall",
				"hermes mcp configure recall",
				"hermes skills install",
				"docs/agent-instructions.md",
				"skills/recall-memory/SKILL.md",
			},
		},
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
				"Hermes setup",
				"hermes mcp test recall",
			},
		},
		{
			path: "docs/templates/AGENTS.md",
			requires: []string{
				"# Agent memory policy",
				"Recall MCP",
				"Read flow",
				"Write flow",
				"Install / enable",
				"hermes mcp test recall",
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

// Package view provides the shared JSON representation of a memory used by all
// frontends (CLI, API server, MCP server). Keeping it in one place avoids
// three near-identical struct + converter pairs drifting apart.
package view

import (
	"recall/internal/memory"
)

// Memory is the JSON-serializable projection of a memory.Memory, with nil
// slices normalized to empty arrays and dates rendered as strings.
type Memory struct {
	ID            string                `json:"id"`
	Title         string                `json:"title"`
	Domain        string                `json:"domain"`
	Tags          []string              `json:"tags"`
	Project       string                `json:"project"`
	Lifecycle     string                `json:"lifecycle"`
	ExpiresOn     string                `json:"expires_on"`
	Created       string                `json:"created"`
	Updated       string                `json:"updated"`
	Source        string                `json:"source"`
	Links         []string              `json:"links"`
	Relationships []memory.Relationship `json:"relationships"`
	Importance    int                   `json:"importance"`
	Path          string                `json:"path"`
	Body          string                `json:"body"`
}

// FromMemory builds a view.Memory from a memory.Memory and its vault-relative
// path, normalizing nil slices to empty arrays.
func FromMemory(m memory.Memory, relPath string) Memory {
	tags := m.Tags
	if tags == nil {
		tags = []string{}
	}
	links := m.Links
	if links == nil {
		links = []string{}
	}
	relationships := m.Relationships
	if relationships == nil {
		relationships = []memory.Relationship{}
	}
	return Memory{
		ID: m.ID, Title: m.Title, Domain: m.Domain, Tags: tags, Project: m.Project,
		Lifecycle: string(m.Lifecycle), ExpiresOn: m.ExpiresOn.String(),
		Created: m.Created.String(), Updated: m.Updated.String(),
		Source: m.Source, Links: links, Relationships: relationships,
		Importance: m.Importance, Path: relPath, Body: m.Body,
	}
}

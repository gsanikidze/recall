// Package mcpserver exposes the recall engine to LLM agents over the Model
// Context Protocol on a local stdio transport. It registers six tools that map
// directly onto the engine. Tool descriptions steer agents to store only
// durable, decision-relevant facts and to route writes by reading domain
// descriptions first.
package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"

	"recall/internal/index"
	"recall/internal/recall"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Serve runs the MCP server on stdio until the connection closes.
func Serve(ctx context.Context, e *recall.Engine, version string) error {
	server := mcp.NewServer(&mcp.Implementation{Name: "recall", Version: version}, nil)
	register(server, e)
	return server.Run(ctx, &mcp.StdioTransport{})
}

// register wires every tool onto the server.
func register(server *mcp.Server, e *recall.Engine) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "recall_search",
		Description: "Search long-lived memory for relevant facts. Provide a natural-language " +
			"query and/or filters. Returns ranked, lightweight hits (id, title, snippet, path); " +
			"call recall_get to read full content. Expired memories are hidden unless include_expired is set.",
	}, searchHandler(e))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "recall_get",
		Description: "Fetch a single memory by id, returning its full Markdown content and vault path.",
	}, getHandler(e))

	mcp.AddTool(server, &mcp.Tool{
		Name: "recall_add",
		Description: "Store a new memory. Only store durable, decision-relevant facts worth recalling " +
			"later — not transient chatter. Choose `domain` by first calling recall_list_domains and " +
			"reading what each domain is for. Use lifecycle 'evergreen' for facts that don't decay " +
			"(tools, people) and 'expires' with expires_on for time-bound facts.",
	}, addHandler(e))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "recall_update",
		Description: "Update fields of an existing memory by id. Only the fields you provide are changed.",
	}, updateHandler(e))

	mcp.AddTool(server, &mcp.Tool{
		Name: "recall_list_domains",
		Description: "List memory domains and what each is for. Call this before recall_add to route a " +
			"new memory to the right domain.",
	}, listDomainsHandler(e))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "recall_reindex",
		Description: "Rebuild the search index from the Markdown vault, picking up any hand-edited files.",
	}, reindexHandler(e))
}

// ---- recall_search ----

type searchArgs struct {
	Query          string   `json:"query,omitempty" jsonschema:"full-text query over title and body"`
	Domain         string   `json:"domain,omitempty" jsonschema:"restrict to a single domain"`
	Tags           []string `json:"tags,omitempty" jsonschema:"match any of these tags"`
	Project        string   `json:"project,omitempty" jsonschema:"restrict to a project grouping key"`
	Lifecycle      string   `json:"lifecycle,omitempty" jsonschema:"evergreen or expires"`
	Since          string   `json:"since,omitempty" jsonschema:"updated on or after this date (YYYY-MM-DD)"`
	Until          string   `json:"until,omitempty" jsonschema:"updated on or before this date (YYYY-MM-DD)"`
	IncludeExpired bool     `json:"include_expired,omitempty" jsonschema:"include memories past their expiry"`
	Limit          int      `json:"limit,omitempty" jsonschema:"maximum number of hits (default 20)"`
}

type hitOut struct {
	ID      string  `json:"id"`
	Title   string  `json:"title"`
	Domain  string  `json:"domain"`
	Snippet string  `json:"snippet"`
	Path    string  `json:"path"`
	Score   float64 `json:"score"`
}

type searchOut struct {
	Hits []hitOut `json:"hits"`
}

func searchHandler(e *recall.Engine) func(context.Context, *mcp.CallToolRequest, searchArgs) (*mcp.CallToolResult, searchOut, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, a searchArgs) (*mcp.CallToolResult, searchOut, error) {
		hits, err := e.Search(ctx, index.Filter{
			Query:          a.Query,
			Domain:         a.Domain,
			Tags:           a.Tags,
			Project:        a.Project,
			Lifecycle:      a.Lifecycle,
			Since:          a.Since,
			Until:          a.Until,
			IncludeExpired: a.IncludeExpired,
			Limit:          a.Limit,
		})
		if err != nil {
			return nil, searchOut{}, err
		}
		out := searchOut{Hits: make([]hitOut, 0, len(hits))}
		for _, h := range hits {
			out.Hits = append(out.Hits, hitOut{
				ID: h.ID, Title: h.Title, Domain: h.Domain,
				Snippet: h.Snippet, Path: h.Path, Score: h.Score,
			})
		}
		return jsonResult(out), out, nil
	}
}

// ---- recall_get ----

type getArgs struct {
	ID string `json:"id" jsonschema:"the memory id"`
}

type getOut struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Domain    string   `json:"domain"`
	Tags      []string `json:"tags,omitempty"`
	Project   string   `json:"project,omitempty"`
	Lifecycle string   `json:"lifecycle"`
	ExpiresOn string   `json:"expires_on,omitempty"`
	Created   string   `json:"created"`
	Updated   string   `json:"updated"`
	Source    string   `json:"source,omitempty"`
	Links     []string `json:"links,omitempty"`
	Path      string   `json:"path"`
	Body      string   `json:"body"`
}

func getHandler(e *recall.Engine) func(context.Context, *mcp.CallToolRequest, getArgs) (*mcp.CallToolResult, getOut, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, a getArgs) (*mcp.CallToolResult, getOut, error) {
		m, relPath, err := e.Get(ctx, a.ID)
		if err != nil {
			return nil, getOut{}, err
		}
		out := getOut{
			ID: m.ID, Title: m.Title, Domain: m.Domain, Tags: m.Tags, Project: m.Project,
			Lifecycle: string(m.Lifecycle), ExpiresOn: m.ExpiresOn.String(),
			Created: m.Created.String(), Updated: m.Updated.String(),
			Source: m.Source, Links: m.Links, Path: relPath, Body: m.Body,
		}
		return jsonResult(out), out, nil
	}
}

// ---- recall_add ----

type addArgs struct {
	Title     string   `json:"title" jsonschema:"short headline for the fact"`
	Body      string   `json:"body" jsonschema:"the fact itself, written tersely (Markdown allowed)"`
	Domain    string   `json:"domain" jsonschema:"target domain folder; see recall_list_domains"`
	Tags      []string `json:"tags,omitempty" jsonschema:"free-form labels"`
	Project   string   `json:"project,omitempty" jsonschema:"project grouping key"`
	Lifecycle string   `json:"lifecycle,omitempty" jsonschema:"evergreen (default) or expires"`
	ExpiresOn string   `json:"expires_on,omitempty" jsonschema:"expiry date YYYY-MM-DD; required when lifecycle is expires"`
	Source    string   `json:"source,omitempty" jsonschema:"who or what produced this memory"`
	Links     []string `json:"links,omitempty" jsonschema:"ids of related memories"`
}

type addOut struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

func addHandler(e *recall.Engine) func(context.Context, *mcp.CallToolRequest, addArgs) (*mcp.CallToolResult, addOut, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, a addArgs) (*mcp.CallToolResult, addOut, error) {
		m, relPath, err := e.Add(ctx, recall.AddParams{
			Title: a.Title, Body: a.Body, Domain: a.Domain, Tags: a.Tags, Project: a.Project,
			Lifecycle: a.Lifecycle, ExpiresOn: a.ExpiresOn, Source: a.Source, Links: a.Links,
		})
		if err != nil {
			return nil, addOut{}, err
		}
		out := addOut{ID: m.ID, Path: relPath}
		return jsonResult(out), out, nil
	}
}

// ---- recall_update ----

type updateArgs struct {
	ID        string    `json:"id" jsonschema:"id of the memory to update"`
	Title     *string   `json:"title,omitempty"`
	Body      *string   `json:"body,omitempty"`
	Tags      *[]string `json:"tags,omitempty"`
	Project   *string   `json:"project,omitempty"`
	Lifecycle *string   `json:"lifecycle,omitempty"`
	ExpiresOn *string   `json:"expires_on,omitempty"`
	Source    *string   `json:"source,omitempty"`
	Links     *[]string `json:"links,omitempty"`
}

func updateHandler(e *recall.Engine) func(context.Context, *mcp.CallToolRequest, updateArgs) (*mcp.CallToolResult, addOut, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, a updateArgs) (*mcp.CallToolResult, addOut, error) {
		m, relPath, err := e.Update(ctx, a.ID, recall.UpdateParams{
			Title: a.Title, Body: a.Body, Tags: a.Tags, Project: a.Project,
			Lifecycle: a.Lifecycle, ExpiresOn: a.ExpiresOn, Source: a.Source, Links: a.Links,
		})
		if err != nil {
			return nil, addOut{}, err
		}
		out := addOut{ID: m.ID, Path: relPath}
		return jsonResult(out), out, nil
	}
}

// ---- recall_list_domains ----

type emptyArgs struct{}

type domainOut struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type domainsOut struct {
	Domains []domainOut `json:"domains"`
}

func listDomainsHandler(e *recall.Engine) func(context.Context, *mcp.CallToolRequest, emptyArgs) (*mcp.CallToolResult, domainsOut, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, _ emptyArgs) (*mcp.CallToolResult, domainsOut, error) {
		domains, err := e.Vault().ListDomains()
		if err != nil {
			return nil, domainsOut{}, err
		}
		out := domainsOut{Domains: make([]domainOut, 0, len(domains))}
		for _, d := range domains {
			out.Domains = append(out.Domains, domainOut{Name: d.Name, Description: d.Description})
		}
		return jsonResult(out), out, nil
	}
}

// ---- recall_reindex ----

type reindexOut struct {
	Indexed int `json:"indexed"`
	Deleted int `json:"deleted"`
}

func reindexHandler(e *recall.Engine) func(context.Context, *mcp.CallToolRequest, emptyArgs) (*mcp.CallToolResult, reindexOut, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, _ emptyArgs) (*mcp.CallToolResult, reindexOut, error) {
		stats, err := e.Reindex(ctx)
		if err != nil {
			return nil, reindexOut{}, err
		}
		out := reindexOut{Indexed: stats.Indexed, Deleted: stats.Deleted}
		return jsonResult(out), out, nil
	}
}

// jsonResult renders a value as pretty JSON text content for the tool result.
func jsonResult(v any) *mcp.CallToolResult {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		data = []byte(fmt.Sprintf("%v", v))
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}
}

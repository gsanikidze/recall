// Package mcpserver exposes the recall engine to LLM agents over the Model
// Context Protocol on a local stdio transport. It registers tools that map
// directly onto the engine. Tool descriptions steer agents to store only
// durable, decision-relevant facts and to route writes by reading domain
// descriptions first.
package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"recall/internal/doctor"
	"recall/internal/embedding"
	"recall/internal/index"
	"recall/internal/memory"
	"recall/internal/recall"
	"recall/internal/view"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Serve runs the MCP server on stdio until the connection closes.
// configPath is the recall config file path (may be empty if undiscoverable).
func Serve(ctx context.Context, e *recall.Engine, version, configPath string, switchers ...ProjectSwitcher) error {
	server := mcp.NewServer(&mcp.Implementation{Name: "recall", Version: version}, nil)
	register(server, e, configPath, switchers...)
	return server.Run(ctx, &mcp.StdioTransport{})
}

// ProjectSwitcher changes the saved Recall project directory.
type ProjectSwitcher func(context.Context, string) (ProjectOut, error)

type ProjectOut struct {
	ProjectPath string `json:"project_path"`
	VaultPath   string `json:"vault_path"`
	DBPath      string `json:"db_path"`
}

// register wires every tool onto the server.
func register(server *mcp.Server, e *recall.Engine, configPath string, switchers ...ProjectSwitcher) {
	var switcher ProjectSwitcher
	if len(switchers) > 0 {
		switcher = switchers[0]
	}
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
			"(tools, people) and 'expires' with expires_on for time-bound facts. Choose importance " +
			"automatically: 1 low, 2 useful, 3 default durable, 4 high-value, 5 critical operating fact/preference/path. " +
			"When a durable link to another memory is known, include relationships with target_id, type, and optional note.",
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

	mcp.AddTool(server, &mcp.Tool{
		Name: "recall_graph",
		Description: "Return typed memory relationships as a node/edge graph. " +
			"Optionally pass domain to include edges whose source memory is in that domain; " +
			"targets are still included so relationships remain visible.",
	}, graphHandler(e))

	mcp.AddTool(server, &mcp.Tool{
		Name: "recall_use_project",
		Description: "Change the saved Recall project directory. Existing files are preserved; missing " +
			"vault/ and db/ scaffold directories are created. New MCP sessions use the new directory.",
	}, useProjectHandler(switcher))

	mcp.AddTool(server, &mcp.Tool{
		Name: "recall_doctor",
		Description: "Check config, vault, SQLite index, domains, and embedding coverage. " +
			"Returns a health report with ok status, paths, counts, invalid files, stale index rows, " +
			"embedding backend probe, missing embedding IDs (first 20), and copy-pasteable fix suggestions. " +
			"Use deep=true for vault/index drift audit and invalid file detection. " +
			"Use embeddings=true to probe embedding backend and report coverage. " +
			"provider/model default to ollama/nomic-embed-text.",
	}, doctorHandler(e, configPath))
}

// ---- recall_use_project ----

type useProjectArgs struct {
	Path string `json:"path" jsonschema:"project root containing vault/ and db/, or where they should be created"`
}

func useProjectHandler(switcher ProjectSwitcher) func(context.Context, *mcp.CallToolRequest, useProjectArgs) (*mcp.CallToolResult, ProjectOut, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, a useProjectArgs) (*mcp.CallToolResult, ProjectOut, error) {
		if strings.TrimSpace(a.Path) == "" {
			return nil, ProjectOut{}, fmt.Errorf("path is required")
		}
		if switcher == nil {
			return nil, ProjectOut{}, fmt.Errorf("project switcher is not configured")
		}
		out, err := switcher(ctx, a.Path)
		if err != nil {
			return nil, ProjectOut{}, err
		}
		return jsonResult(out), out, nil
	}
}

// ---- recall_doctor ----

type doctorArgs struct {
	Deep       bool   `json:"deep,omitempty" jsonschema:"audit vault/index drift and invalid memory files"`
	Embeddings bool   `json:"embeddings,omitempty" jsonschema:"probe embedding backend and report coverage"`
	Provider   string `json:"provider,omitempty" jsonschema:"embedding provider for embeddings audit; default ollama"`
	Model      string `json:"model,omitempty" jsonschema:"embedding model for embeddings audit; default nomic-embed-text"`
}

func doctorHandler(e *recall.Engine, configPath string) func(context.Context, *mcp.CallToolRequest, doctorArgs) (*mcp.CallToolResult, doctor.Report, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, a doctorArgs) (*mcp.CallToolResult, doctor.Report, error) {
		projectPath := ""
		if e != nil {
			projectPath = e.ProjectPath()
		}
		vaultPath := doctor.JoinVaultPath(projectPath)
		dbPath := doctor.JoinDBPath(projectPath)
		report := doctor.Run(ctx, e, doctor.Options{
			Deep:       a.Deep,
			Embeddings: a.Embeddings,
			Provider:   a.Provider,
			Model:      a.Model,
		}, projectPath, vaultPath, dbPath, configPath)
		return jsonResult(report), report, nil
	}
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
	Mode           string   `json:"mode,omitempty" jsonschema:"keyword, semantic, or hybrid; default keyword"`
	Provider       string   `json:"provider,omitempty" jsonschema:"embedding provider for semantic/hybrid search; default ollama"`
	Model          string   `json:"model,omitempty" jsonschema:"embedding model for semantic/hybrid search; default nomic-embed-text"`
	BaseURL        string   `json:"base_url,omitempty" jsonschema:"Ollama base URL for semantic/hybrid search"`
}

type hitOut struct {
	ID            string  `json:"id"`
	Title         string  `json:"title"`
	Domain        string  `json:"domain"`
	Snippet       string  `json:"snippet"`
	Path          string  `json:"path"`
	Importance    int     `json:"importance"`
	Score         float64 `json:"score"`
	KeywordScore  float64 `json:"keyword_score,omitempty"`
	SemanticScore float64 `json:"semantic_score,omitempty"`
}

type searchOut struct {
	Hits []hitOut `json:"hits"`
}

func searchHandler(e *recall.Engine) func(context.Context, *mcp.CallToolRequest, searchArgs) (*mcp.CallToolResult, searchOut, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, a searchArgs) (*mcp.CallToolResult, searchOut, error) {
		mode, err := index.ParseSearchMode(a.Mode)
		if err != nil {
			return nil, searchOut{}, err
		}
		filter := index.Filter{
			Query:          a.Query,
			Domain:         a.Domain,
			Tags:           a.Tags,
			Project:        a.Project,
			Lifecycle:      a.Lifecycle,
			Since:          a.Since,
			Until:          a.Until,
			IncludeExpired: a.IncludeExpired,
			Limit:          a.Limit,
			Mode:           mode,
		}
		if mode == index.SearchModeSemantic || mode == index.SearchModeHybrid {
			providerName := strings.TrimSpace(a.Provider)
			if providerName == "" {
				providerName = "ollama"
			}
			model := strings.TrimSpace(a.Model)
			if model == "" {
				model = embedding.DefaultOllamaModel
			}
			baseURL := strings.TrimSpace(a.BaseURL)
			if baseURL == "" {
				baseURL = os.Getenv("RECALL_OLLAMA_URL")
			}
			provider, err := embedding.NewProvider(providerName, model, baseURL)
			if err != nil {
				return nil, searchOut{}, err
			}
			vectors, err := provider.Embed(ctx, []string{filter.Query})
			if err != nil {
				return nil, searchOut{}, err
			}
			if len(vectors) != 1 {
				return nil, searchOut{}, fmt.Errorf("mcp: provider returned %d query vectors, want 1", len(vectors))
			}
			filter.QueryVector = vectors[0]
			filter.Provider = provider.Name()
			filter.Model = provider.Model()
		}
		hits, err := e.Search(ctx, filter)
		if err != nil {
			return nil, searchOut{}, err
		}
		out := searchOut{Hits: make([]hitOut, 0, len(hits))}
		for _, h := range hits {
			out.Hits = append(out.Hits, hitOut{
				ID: h.ID, Title: h.Title, Domain: h.Domain,
				Snippet: h.Snippet, Path: h.Path, Importance: h.Importance, Score: h.Score,
				KeywordScore: h.KeywordScore, SemanticScore: h.SemanticScore,
			})
		}
		return jsonResult(out), out, nil
	}
}

// ---- recall_get ----

type getArgs struct {
	ID string `json:"id" jsonschema:"the memory id"`
}

func getHandler(e *recall.Engine) func(context.Context, *mcp.CallToolRequest, getArgs) (*mcp.CallToolResult, view.Memory, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, a getArgs) (*mcp.CallToolResult, view.Memory, error) {
		m, relPath, err := e.Get(ctx, a.ID)
		if err != nil {
			return nil, view.Memory{}, err
		}
		out := view.FromMemory(m, relPath)
		return jsonResult(out), out, nil
	}
}

// ---- recall_add ----

type addArgs struct {
	Title         string                `json:"title" jsonschema:"short headline for the fact"`
	Body          string                `json:"body" jsonschema:"the fact itself, written tersely (Markdown allowed)"`
	Domain        string                `json:"domain" jsonschema:"target domain folder; see recall_list_domains"`
	Tags          []string              `json:"tags,omitempty" jsonschema:"free-form labels"`
	Project       string                `json:"project,omitempty" jsonschema:"project grouping key"`
	Lifecycle     string                `json:"lifecycle,omitempty" jsonschema:"evergreen (default) or expires"`
	ExpiresOn     string                `json:"expires_on,omitempty" jsonschema:"expiry date YYYY-MM-DD; required when lifecycle is expires"`
	Source        string                `json:"source,omitempty" jsonschema:"who or what produced this memory"`
	Links         []string              `json:"links,omitempty" jsonschema:"legacy ids of related memories; prefer relationships for typed graph edges"`
	Relationships []memory.Relationship `json:"relationships,omitempty" jsonschema:"typed directed graph edges: target_id, type, optional note. Choose when durable context links this memory to another memory."`
	Importance    int                   `json:"importance,omitempty" jsonschema:"1-5; choose automatically based on durable value: 1 low, 3 default, 5 critical"`
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
			Relationships: a.Relationships, Importance: a.Importance,
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
	ID            string                 `json:"id" jsonschema:"id of the memory to update"`
	Title         *string                `json:"title,omitempty"`
	Body          *string                `json:"body,omitempty"`
	Tags          *[]string              `json:"tags,omitempty"`
	Project       *string                `json:"project,omitempty"`
	Lifecycle     *string                `json:"lifecycle,omitempty"`
	ExpiresOn     *string                `json:"expires_on,omitempty"`
	Source        *string                `json:"source,omitempty"`
	Links         *[]string              `json:"links,omitempty"`
	Relationships *[]memory.Relationship `json:"relationships,omitempty"`
	Importance    *int                   `json:"importance,omitempty"`
}

func updateHandler(e *recall.Engine) func(context.Context, *mcp.CallToolRequest, updateArgs) (*mcp.CallToolResult, addOut, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, a updateArgs) (*mcp.CallToolResult, addOut, error) {
		m, relPath, err := e.Update(ctx, a.ID, recall.UpdateParams{
			Title: a.Title, Body: a.Body, Tags: a.Tags, Project: a.Project,
			Lifecycle: a.Lifecycle, ExpiresOn: a.ExpiresOn, Source: a.Source,
			Links: a.Links, Relationships: a.Relationships, Importance: a.Importance,
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

// ---- recall_graph ----

type graphArgs struct {
	Domain string `json:"domain,omitempty" jsonschema:"restrict to edges whose source memory is in this domain"`
}

func graphHandler(e *recall.Engine) func(context.Context, *mcp.CallToolRequest, graphArgs) (*mcp.CallToolResult, recall.Graph, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, a graphArgs) (*mcp.CallToolResult, recall.Graph, error) {
		graph, err := e.Graph(ctx, strings.TrimSpace(a.Domain))
		if err != nil {
			return nil, recall.Graph{}, err
		}
		return jsonResult(graph), graph, nil
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

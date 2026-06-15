package mcpserver

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"recall/internal/embedding"
	"recall/internal/index"
	"recall/internal/memory"
	"recall/internal/recall"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// startTestServer wires a fresh engine to an MCP server over an in-memory
// transport and returns a connected client session.
func startTestServer(t *testing.T) *mcp.ClientSession {
	t.Helper()
	_, session := startTestServerWithEngine(t)
	return session
}

func startTestServerWithEngine(t *testing.T) (*recall.Engine, *mcp.ClientSession) {
	return startTestServerWithEngineAndSwitcher(t, nil)
}

func startTestServerWithEngineAndSwitcher(t *testing.T, switcher ProjectSwitcher) (*recall.Engine, *mcp.ClientSession) {
	t.Helper()
	ctx := context.Background()

	e, err := recall.Open(t.TempDir())
	if err != nil {
		t.Fatalf("recall.Open: %v", err)
	}
	if err := e.Vault().Scaffold(); err != nil {
		t.Fatalf("Scaffold: %v", err)
	}
	t.Cleanup(func() { _ = e.Close() })

	server := mcp.NewServer(&mcp.Implementation{Name: "recall", Version: "test"}, nil)
	register(server, e, switcher)

	serverT, clientT := mcp.NewInMemoryTransports()
	if _, err := server.Connect(ctx, serverT, nil); err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	session, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return e, session
}

// call invokes a tool and decodes its JSON text result into out.
func call(t *testing.T, s *mcp.ClientSession, name string, args map[string]any, out any) {
	t.Helper()
	res, err := s.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool %s: %v", name, err)
	}
	if res.IsError {
		t.Fatalf("tool %s returned error: %+v", name, res.Content)
	}
	if out == nil {
		return
	}
	text, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("tool %s: expected text content, got %T", name, res.Content[0])
	}
	if err := json.Unmarshal([]byte(text.Text), out); err != nil {
		t.Fatalf("tool %s: decoding result %q: %v", name, text.Text, err)
	}
}

func callExpectError(t *testing.T, s *mcp.ClientSession, name string, args map[string]any, want string) {
	t.Helper()
	res, err := s.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool transport error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected tool error from %s", name)
	}
	if want == "" {
		return
	}
	if len(res.Content) == 0 {
		t.Fatalf("tool error missing content")
	}
	text, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected text error content, got %T", res.Content[0])
	}
	if !strings.Contains(text.Text, want) {
		t.Fatalf("error %q missing %q", text.Text, want)
	}
}

func TestMCPToolsListed(t *testing.T) {
	s := startTestServer(t)
	res, err := s.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	want := map[string]bool{
		"recall_search": false, "recall_get": false, "recall_add": false,
		"recall_update": false, "recall_list_domains": false, "recall_reindex": false,
		"recall_use_project": false,
	}
	for _, tool := range res.Tools {
		if _, ok := want[tool.Name]; ok {
			want[tool.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("tool %q not registered", name)
		}
	}
}

func TestMCPUseProjectCallsSwitcher(t *testing.T) {
	_, s := startTestServerWithEngineAndSwitcher(t, func(ctx context.Context, path string) (projectOut, error) {
		if path != "/tmp/new-brain" {
			t.Fatalf("path = %q", path)
		}
		return projectOut{ProjectPath: path, VaultPath: path + "/vault", DBPath: path + "/db/recall.sqlite"}, nil
	})

	var out projectOut
	call(t, s, "recall_use_project", map[string]any{"path": "/tmp/new-brain"}, &out)
	if out.ProjectPath != "/tmp/new-brain" || out.VaultPath != "/tmp/new-brain/vault" || out.DBPath != "/tmp/new-brain/db/recall.sqlite" {
		t.Fatalf("use project out = %+v", out)
	}
}

func TestMCPUseProjectRejectsBlankPath(t *testing.T) {
	s := startTestServer(t)
	callExpectError(t, s, "recall_use_project", map[string]any{"path": " \t"}, "path is required")
}

func TestMCPAddSearchGetFlow(t *testing.T) {
	s := startTestServer(t)

	// list domains
	var domains domainsOut
	call(t, s, "recall_list_domains", map[string]any{}, &domains)
	if len(domains.Domains) == 0 {
		t.Fatal("no domains listed")
	}

	// add
	var added addOut
	call(t, s, "recall_add", map[string]any{
		"title":      "Kamal deploy",
		"body":       "Deploys run through **Kamal**.",
		"domain":     "tools",
		"tags":       []string{"deploy"},
		"importance": 5,
		"relationships": []map[string]any{
			{"target_id": "01TARGET000000000000000001", "type": "uses_tool", "note": "mcp edge"},
		},
	}, &added)
	if added.ID == "" || added.Path == "" {
		t.Fatalf("add returned empty: %+v", added)
	}

	// search
	var search searchOut
	call(t, s, "recall_search", map[string]any{"query": "kamal"}, &search)
	if len(search.Hits) != 1 || search.Hits[0].ID != added.ID || search.Hits[0].Importance != 5 {
		t.Fatalf("search = %+v", search)
	}

	// get
	var got getOut
	call(t, s, "recall_get", map[string]any{"id": added.ID}, &got)
	if got.Title != "Kamal deploy" || got.Domain != "tools" || got.Importance != 5 {
		t.Fatalf("get = %+v", got)
	}
	if len(got.Relationships) != 1 || got.Relationships[0].Type != memory.RelationshipUsesTool || got.Relationships[0].Note != "mcp edge" {
		t.Fatalf("get relationships = %+v", got.Relationships)
	}
	// Full Markdown body is returned to the agent, formatting intact.
	if got.Body == "" {
		t.Error("get returned empty body")
	}

	// update
	call(t, s, "recall_update", map[string]any{"id": added.ID, "body": "Now uses Kamal v2 and widgets.", "importance": 4}, nil)
	var search2 searchOut
	call(t, s, "recall_search", map[string]any{"query": "widgets"}, &search2)
	if len(search2.Hits) != 1 || search2.Hits[0].Importance != 4 {
		t.Fatalf("search after update = %+v", search2)
	}
}

func TestMCPSemanticAndHybridSearchModes(t *testing.T) {
	e, s := startTestServerWithEngine(t)
	ctx := context.Background()
	phone, _, err := e.Add(ctx, recall.AddParams{Title: "Phone Sync", Body: "iPhone Obsidian setup", Domain: "tools"})
	if err != nil {
		t.Fatalf("Add phone: %v", err)
	}
	if _, _, err := e.Add(ctx, recall.AddParams{Title: "Recall Policy", Body: "local first memory policy", Domain: "decisions"}); err != nil {
		t.Fatalf("Add policy: %v", err)
	}
	if _, err := e.EmbedAll(ctx, embedding.NewFakeProvider("fake-32", 32), false); err != nil {
		t.Fatalf("EmbedAll: %v", err)
	}

	var semantic searchOut
	call(t, s, "recall_search", map[string]any{"query": "phone sync", "mode": "semantic", "provider": "fake", "model": "fake-32"}, &semantic)
	if len(semantic.Hits) == 0 || semantic.Hits[0].ID != phone.ID || semantic.Hits[0].SemanticScore == 0 {
		t.Fatalf("semantic search = %+v", semantic)
	}

	var hybrid searchOut
	call(t, s, "recall_search", map[string]any{"query": "phone sync", "mode": "hybrid", "provider": "fake", "model": "fake-32"}, &hybrid)
	if len(hybrid.Hits) == 0 || hybrid.Hits[0].ID != phone.ID || hybrid.Hits[0].SemanticScore == 0 {
		t.Fatalf("hybrid search = %+v", hybrid)
	}
}

func TestMCPSearchRejectsUnknownMode(t *testing.T) {
	s := startTestServer(t)
	callExpectError(t, s, "recall_search", map[string]any{"query": "phone", "mode": "weird"}, "mode")
}

func TestMCPAddRejectsUnknownDomain(t *testing.T) {
	s := startTestServer(t)
	callExpectError(t, s, "recall_add", map[string]any{"title": "x", "body": "y", "domain": "nonexistent"}, "unknown domain")
}

func TestMCPSearchRespectsLimitCap(t *testing.T) {
	s := startTestServer(t)
	for i := 0; i < index.MaxLimit+5; i++ {
		call(t, s, "recall_add", map[string]any{
			"title":  "Limit memory",
			"body":   "same searchable body",
			"domain": "tools",
		}, nil)
	}
	var search searchOut
	call(t, s, "recall_search", map[string]any{"limit": index.MaxLimit + 500}, &search)
	if len(search.Hits) != index.MaxLimit {
		t.Fatalf("hits = %d, want %d", len(search.Hits), index.MaxLimit)
	}
}

func TestMCPSearchRejectsInvalidFilters(t *testing.T) {
	s := startTestServer(t)
	cases := []struct {
		name string
		args map[string]any
		want string
	}{
		{"invalid lifecycle", map[string]any{"lifecycle": "bad"}, "lifecycle"},
		{"invalid since", map[string]any{"since": "bad-date"}, "since"},
		{"invalid until", map[string]any{"until": "bad-date"}, "until"},
		{"since after until", map[string]any{"since": "2026-06-09", "until": "2026-06-08"}, "since"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			callExpectError(t, s, "recall_search", tc.args, tc.want)
		})
	}
}

func TestMCPAddRejectsBlankBody(t *testing.T) {
	s := startTestServer(t)
	callExpectError(t, s, "recall_add", map[string]any{"title": "x", "body": " \n	 ", "domain": "tools"}, "body is required")
}

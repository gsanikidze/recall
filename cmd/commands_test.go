package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"recall/internal/index"
	"recall/internal/memory"
	"recall/internal/recall"
)

func TestParseEmbedArgs(t *testing.T) {
	parsed, err := parseEmbedArgs([]string{"--provider", "fake", "--model", "fake-32", "--json"})
	if err != nil {
		t.Fatalf("parseEmbedArgs fake: %v", err)
	}
	if parsed.provider != "fake" || parsed.model != "fake-32" || !parsed.json {
		t.Fatalf("fake parsed = %+v", parsed)
	}

	parsed, err = parseEmbedArgs([]string{"--provider", "ollama", "--model", "nomic-embed-text", "--base-url", "http://127.0.0.1:11434"})
	if err != nil {
		t.Fatalf("parseEmbedArgs ollama: %v", err)
	}
	if parsed.provider != "ollama" || parsed.model != "nomic-embed-text" || parsed.baseURL != "http://127.0.0.1:11434" {
		t.Fatalf("ollama parsed = %+v", parsed)
	}

	parsed, err = parseEmbedArgs([]string{"--force"})
	if err != nil {
		t.Fatalf("parseEmbedArgs force: %v", err)
	}
	if parsed.provider != "ollama" || parsed.model != "nomic-embed-text" || !parsed.force {
		t.Fatalf("force/default parsed = %+v", parsed)
	}
}

func TestEmbedJSONFlowWithFakeProvider(t *testing.T) {
	project := filepath.Join(t.TempDir(), "brain")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := Init([]string{"--path", project, "--force"}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := Add([]string{"--title", "Phone Sync", "--domain", "tools", "--body", "iPhone sync setup"}); err != nil {
		t.Fatalf("Add first: %v", err)
	}
	if err := Add([]string{"--title", "Recall Policy", "--domain", "decisions", "--body", "local first memory policy"}); err != nil {
		t.Fatalf("Add second: %v", err)
	}

	out := captureStdout(t, func() {
		if err := Embed([]string{"--provider", "fake", "--model", "fake-32", "--json"}); err != nil {
			t.Fatalf("Embed json: %v", err)
		}
	})
	if !strings.Contains(out, `"provider": "fake"`) || !strings.Contains(out, `"model": "fake-32"`) || !strings.Contains(out, `"embedded": 2`) || !strings.Contains(out, `"skipped": 0`) {
		t.Fatalf("embed json output = %s", out)
	}

	out = captureStdout(t, func() {
		if err := Embed([]string{"--provider", "fake", "--model", "fake-32", "--json"}); err != nil {
			t.Fatalf("Embed json second: %v", err)
		}
	})
	if !strings.Contains(out, `"embedded": 0`) || !strings.Contains(out, `"skipped": 2`) {
		t.Fatalf("second embed json output = %s", out)
	}
}

func TestDoctorFixReindexesVaultDrift(t *testing.T) {
	project := filepath.Join(t.TempDir(), "brain")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := Init([]string{"--path", project, "--force"}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	today := memory.Today()
	unindexed := memory.Memory{
		ID:         memory.NewID(),
		Title:      "Unindexed",
		Domain:     "tools",
		Created:    today,
		Updated:    today,
		Importance: 3,
		Lifecycle:  memory.Evergreen,
		Body:       "only in vault",
	}
	data, err := unindexed.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	abs := filepath.Join(project, "vault", "tools", "unindexed.md")
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(abs, data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	out := captureStdout(t, func() {
		if err := Doctor([]string{"--deep", "--fix", "--json"}); err != nil {
			t.Fatalf("Doctor --fix: %v", err)
		}
	})
	if !strings.Contains(out, `"ok": true`) || !strings.Contains(out, `"reindex"`) || !strings.Contains(out, `"indexed": 1`) {
		t.Fatalf("doctor fix output = %s", out)
	}

	e, err := recall.Open(project)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer e.Close()
	if _, _, err := e.Get(context.Background(), unindexed.ID); err != nil {
		t.Fatalf("fixed index missing memory %s: %v", unindexed.ID, err)
	}
}

func TestDoctorFixEmbeddingsWithFakeProvider(t *testing.T) {
	project := filepath.Join(t.TempDir(), "brain")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := Init([]string{"--path", project, "--force"}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := Add([]string{"--title", "Needs embedding", "--domain", "tools", "--body", "semantic body"}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	out := captureStdout(t, func() {
		if err := Doctor([]string{"--embeddings", "--provider", "fake", "--model", "fake-32", "--fix", "--fix-embeddings", "--json"}); err != nil {
			t.Fatalf("Doctor --fix-embeddings: %v", err)
		}
	})
	if !strings.Contains(out, `"ok": true`) || !strings.Contains(out, `"missing": 0`) || !strings.Contains(out, `"embedded": 1`) || !strings.Contains(out, `"embed"`) {
		t.Fatalf("doctor fix embeddings output = %s", out)
	}
}

func TestParseSearchArgsAcceptsFlagsAfterQuery(t *testing.T) {
	parsed, err := parseSearchArgs([]string{"Smoke memory", "--domain", "tools", "--tag", "smoke", "--project", "recall", "--limit", "5", "--json"})
	if err != nil {
		t.Fatalf("parseSearchArgs: %v", err)
	}
	if parsed.filter.Query != "Smoke memory" {
		t.Fatalf("query = %q", parsed.filter.Query)
	}
	if parsed.filter.Domain != "tools" || parsed.filter.Project != "recall" || parsed.filter.Limit != 5 {
		t.Fatalf("filter = %+v", parsed.filter)
	}
	if len(parsed.filter.Tags) != 1 || parsed.filter.Tags[0] != "smoke" {
		t.Fatalf("tags = %v", parsed.filter.Tags)
	}
	if !parsed.json {
		t.Fatalf("json flag not captured")
	}
}

func TestParseSearchArgsSemanticAndHybridFlags(t *testing.T) {
	parsed, err := parseSearchArgs([]string{"phone setup", "--semantic", "--provider", "fake", "--model", "fake-32", "--base-url", "http://127.0.0.1:11434"})
	if err != nil {
		t.Fatalf("parse semantic search args: %v", err)
	}
	if parsed.filter.Query != "phone setup" || parsed.filter.Mode != index.SearchModeSemantic || parsed.provider != "fake" || parsed.model != "fake-32" || parsed.baseURL != "http://127.0.0.1:11434" {
		t.Fatalf("semantic parsed = %+v", parsed)
	}

	parsed, err = parseSearchArgs([]string{"phone setup", "--hybrid", "--provider", "fake", "--model", "fake-32"})
	if err != nil {
		t.Fatalf("parse hybrid search args: %v", err)
	}
	if parsed.filter.Query != "phone setup" || parsed.filter.Mode != index.SearchModeHybrid || parsed.provider != "fake" || parsed.model != "fake-32" {
		t.Fatalf("hybrid parsed = %+v", parsed)
	}

	parsed, err = parseSearchArgs([]string{"phone setup", "--mode", "hybrid", "--provider", "fake", "--model", "fake-32"})
	if err != nil {
		t.Fatalf("parse --mode hybrid search args: %v", err)
	}
	if parsed.filter.Query != "phone setup" || parsed.filter.Mode != index.SearchModeHybrid || parsed.provider != "fake" || parsed.model != "fake-32" {
		t.Fatalf("mode hybrid parsed = %+v", parsed)
	}

	parsed, err = parseSearchArgs([]string{"phone setup", "--mode", "semantic", "--provider", "fake", "--model", "fake-32"})
	if err != nil {
		t.Fatalf("parse --mode semantic search args: %v", err)
	}
	if parsed.filter.Query != "phone setup" || parsed.filter.Mode != index.SearchModeSemantic {
		t.Fatalf("mode semantic parsed = %+v", parsed)
	}

	parsed, err = parseSearchArgs([]string{"phone setup", "--mode", "keyword"})
	if err != nil {
		t.Fatalf("parse --mode keyword search args: %v", err)
	}
	if parsed.filter.Query != "phone setup" || parsed.filter.Mode != index.SearchModeKeyword {
		t.Fatalf("mode keyword parsed = %+v", parsed)
	}

	if _, err := parseSearchArgs([]string{"phone", "--semantic", "--hybrid"}); err == nil {
		t.Fatalf("semantic+hybrid parsed without mutual exclusion error")
	}
	if _, err := parseSearchArgs([]string{"phone", "--semantic", "--mode", "hybrid"}); err == nil {
		t.Fatalf("semantic+mode hybrid parsed without mutual exclusion error")
	}
	if _, err := parseSearchArgs([]string{"phone", "--mode", "weird"}); err == nil {
		t.Fatalf("unknown --mode parsed without error")
	}
	if _, err := parseSearchArgs([]string{"--semantic", "--provider", "fake", "--model", "fake-32"}); err == nil {
		t.Fatalf("semantic search parsed without query")
	}
}

func TestSearchSemanticJSONFlowWithFakeProvider(t *testing.T) {
	project := filepath.Join(t.TempDir(), "brain")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := Init([]string{"--path", project, "--force"}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := Add([]string{"--title", "Phone Sync", "--domain", "tools", "--body", "iPhone Obsidian sync setup"}); err != nil {
		t.Fatalf("Add phone: %v", err)
	}
	if err := Add([]string{"--title", "Recall Policy", "--domain", "decisions", "--body", "local first memory policy"}); err != nil {
		t.Fatalf("Add policy: %v", err)
	}
	if err := Embed([]string{"--provider", "fake", "--model", "fake-32"}); err != nil {
		t.Fatalf("Embed fake: %v", err)
	}

	out := captureStdout(t, func() {
		if err := Search([]string{"phone sync", "--semantic", "--provider", "fake", "--model", "fake-32", "--json"}); err != nil {
			t.Fatalf("Search semantic fake: %v", err)
		}
	})
	if !strings.Contains(out, `"hits"`) || !strings.Contains(out, `"semantic_score"`) || !strings.Contains(out, "Phone Sync") {
		t.Fatalf("semantic search json output = %s", out)
	}
}

func TestInitPathFlagAndEnvOverride(t *testing.T) {
	configHome := t.TempDir()
	project := filepath.Join(t.TempDir(), "brain")
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("RECALL_PROJECT", project)

	if err := Init([]string{"--path", project, "--force"}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	cfg, found, err := loadConfig()
	if err != nil || !found {
		t.Fatalf("loadConfig found=%v err=%v", found, err)
	}
	if cfg.ProjectPath != project {
		t.Fatalf("config project = %q, want %q", cfg.ProjectPath, project)
	}

	e, err := openEngine()
	if err != nil {
		t.Fatalf("openEngine with RECALL_PROJECT: %v", err)
	}
	defer e.Close()
	if got := filepath.Dir(e.Vault().Root()); got != project {
		t.Fatalf("engine project = %q, want %q", got, project)
	}
}

func TestUseUpdatesProjectConfigAndPreservesExistingFiles(t *testing.T) {
	configHome := t.TempDir()
	project := filepath.Join(t.TempDir(), "existing-brain")
	existing := filepath.Join(project, "notes.md")
	t.Setenv("XDG_CONFIG_HOME", configHome)

	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	if err := os.WriteFile(existing, []byte("keep me"), 0o644); err != nil {
		t.Fatalf("write existing file: %v", err)
	}

	out := captureStdout(t, func() {
		if err := Use([]string{project}); err != nil {
			t.Fatalf("Use: %v", err)
		}
	})
	if !strings.Contains(out, "project stored at: "+project) {
		t.Fatalf("use output = %s", out)
	}

	cfg, found, err := loadConfig()
	if err != nil || !found {
		t.Fatalf("loadConfig found=%v err=%v", found, err)
	}
	if cfg.ProjectPath != project {
		t.Fatalf("config project = %q, want %q", cfg.ProjectPath, project)
	}
	if _, err := os.Stat(existing); err != nil {
		t.Fatalf("existing file not preserved: %v", err)
	}
	for _, rel := range []string{filepath.Join("vault", "README.md"), "db"} {
		if _, err := os.Stat(filepath.Join(project, rel)); err != nil {
			t.Fatalf("missing scaffold %s: %v", rel, err)
		}
	}

	e, err := openEngine()
	if err != nil {
		t.Fatalf("openEngine: %v", err)
	}
	defer e.Close()
	if got := filepath.Dir(e.Vault().Root()); got != project {
		t.Fatalf("engine project = %q, want %q", got, project)
	}
}

func TestUseWarnsWhenExistingMarkdownIsOutsideVault(t *testing.T) {
	project := filepath.Join(t.TempDir(), "existing-brain")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	if err := os.WriteFile(filepath.Join(project, "meeting-notes.md"), []byte("root note"), 0o644); err != nil {
		t.Fatalf("write root markdown: %v", err)
	}

	out := captureStdout(t, func() {
		if err := Use([]string{project}); err != nil {
			t.Fatalf("Use: %v", err)
		}
	})
	if !strings.Contains(out, "warning:") || !strings.Contains(out, "vault/") || !strings.Contains(out, "meeting-notes.md") {
		t.Fatalf("use output should warn about root markdown outside vault, got %s", out)
	}
}

func TestUseRejectsDirectVaultDirectory(t *testing.T) {
	project := filepath.Join(t.TempDir(), "brain")
	directVault := filepath.Join(project, "vault")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := os.MkdirAll(directVault, 0o755); err != nil {
		t.Fatalf("mkdir vault: %v", err)
	}

	err := Use([]string{directVault})
	if err == nil || !strings.Contains(err.Error(), "project root") || !strings.Contains(err.Error(), project) {
		t.Fatalf("Use(vault) err = %v", err)
	}
}

func TestUseRequiresPath(t *testing.T) {
	if err := Use(nil); err == nil || !strings.Contains(err.Error(), "usage: recall use <path>") {
		t.Fatalf("Use(nil) err = %v", err)
	}
}

func TestParseDevArgs(t *testing.T) {
	parsed, err := parseDevArgs(nil)
	if err != nil {
		t.Fatalf("parseDevArgs defaults: %v", err)
	}
	if parsed.apiPort != 8888 || parsed.uiPort != 5173 || parsed.install {
		t.Fatalf("default dev args = %+v", parsed)
	}

	parsed, err = parseDevArgs([]string{"--api-port", "9999", "--ui-port", "5174", "--install"})
	if err != nil {
		t.Fatalf("parseDevArgs custom: %v", err)
	}
	if parsed.apiPort != 9999 || parsed.uiPort != 5174 || !parsed.install {
		t.Fatalf("custom dev args = %+v", parsed)
	}
	if parsed.apiURL() != "http://localhost:9999" {
		t.Fatalf("apiURL = %q", parsed.apiURL())
	}

	if _, err := parseDevArgs([]string{"extra"}); err == nil || !strings.Contains(err.Error(), "usage: recall dev") {
		t.Fatalf("parseDevArgs extra err = %v", err)
	}
}

func TestDoctorDeepJSONReportsVaultIndexDriftAndInvalidFiles(t *testing.T) {
	project := filepath.Join(t.TempDir(), "brain")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := Init([]string{"--path", project, "--force"}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	out := captureStdout(t, func() {
		if err := Add([]string{"--title", "Valid Memory", "--domain", "tools", "--body", "indexed and on disk", "--json"}); err != nil {
			t.Fatalf("Add valid: %v", err)
		}
	})
	validID := extractJSONField(out, "id")
	if validID == "" {
		t.Fatalf("missing valid id in %s", out)
	}

	out = captureStdout(t, func() {
		if err := Add([]string{"--title", "Missing File", "--domain", "tools", "--body", "indexed but removed", "--json"}); err != nil {
			t.Fatalf("Add missing: %v", err)
		}
	})
	missingID := extractJSONField(out, "id")
	missingPath := strings.TrimPrefix(extractJSONField(out, "path"), "vault/")
	if missingID == "" || missingPath == "" {
		t.Fatalf("missing id/path in %s", out)
	}
	if err := os.Remove(filepath.Join(project, "vault", missingPath)); err != nil {
		t.Fatalf("remove indexed file: %v", err)
	}
	invalidPath := filepath.Join(project, "vault", "tools", "broken.md")
	if err := os.WriteFile(invalidPath, []byte("---\nid: not-a-valid-memory\n---\n# broken"), 0o644); err != nil {
		t.Fatalf("write invalid memory: %v", err)
	}

	out = captureStdout(t, func() {
		if err := Doctor([]string{"--json", "--deep"}); err != nil {
			t.Fatalf("Doctor deep json: %v", err)
		}
	})
	for _, want := range []string{
		`"ok": false`,
		`"vault_memories": 1`,
		`"index_memories": 2`,
		`"invalid_files"`,
		`tools/broken.md`,
		`"stale_index_ids"`,
		missingID,
		`"missing_index_paths"`,
		missingPath,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("doctor deep output missing %q: %s", want, out)
		}
	}
}

func TestDoctorEmbeddingsJSONReportsCoverage(t *testing.T) {
	project := filepath.Join(t.TempDir(), "brain")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := Init([]string{"--path", project, "--force"}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := Add([]string{"--title", "One", "--domain", "tools", "--body", "first"}); err != nil {
		t.Fatalf("Add one: %v", err)
	}
	if err := Add([]string{"--title", "Two", "--domain", "tools", "--body", "second"}); err != nil {
		t.Fatalf("Add two: %v", err)
	}

	out := captureStdout(t, func() {
		if err := Doctor([]string{"--json", "--embeddings", "--provider", "fake", "--model", "fake-32"}); err != nil {
			t.Fatalf("Doctor embeddings missing: %v", err)
		}
	})
	for _, want := range []string{`"ok": false`, `"provider": "fake"`, `"model": "fake-32"`, `"embedded": 0`, `"missing": 2`, `"coverage": 0`} {
		if !strings.Contains(out, want) {
			t.Fatalf("doctor missing embeddings output missing %q: %s", want, out)
		}
	}

	if err := Embed([]string{"--provider", "fake", "--model", "fake-32"}); err != nil {
		t.Fatalf("Embed fake: %v", err)
	}
	out = captureStdout(t, func() {
		if err := Doctor([]string{"--json", "--embeddings", "--provider", "fake", "--model", "fake-32"}); err != nil {
			t.Fatalf("Doctor embeddings ready: %v", err)
		}
	})
	for _, want := range []string{`"ok": true`, `"embedded": 2`, `"missing": 0`, `"coverage": 1`} {
		if !strings.Contains(out, want) {
			t.Fatalf("doctor ready embeddings output missing %q: %s", want, out)
		}
	}
}

func TestAddSearchGetDeleteJSONFlow(t *testing.T) {
	project := filepath.Join(t.TempDir(), "brain")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := Init([]string{"--path", project, "--force"}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	out := captureStdout(t, func() {
		if err := Domain([]string{"list", "--json"}); err != nil {
			t.Fatalf("Domain list json: %v", err)
		}
	})
	if !strings.Contains(out, `"domains"`) || !strings.Contains(out, `"tools"`) {
		t.Fatalf("domain json output = %s", out)
	}

	out = captureStdout(t, func() {
		if err := Doctor([]string{"--json"}); err != nil {
			t.Fatalf("Doctor json: %v", err)
		}
	})
	if !strings.Contains(out, `"ok": true`) || !strings.Contains(out, `"project_path"`) {
		t.Fatalf("doctor json output = %s", out)
	}

	out = captureStdout(t, func() {
		if err := Add([]string{"--title", "Smoke Memory", "--domain", "tools", "--body", "Smoke memory text", "--importance", "5", "--relationships", `[{"target_id":"01TARGET000000000000000001","type":"uses_tool","note":"cli edge"}]`, "--json"}); err != nil {
			t.Fatalf("Add: %v", err)
		}
	})
	if !strings.Contains(out, `"id"`) || !strings.Contains(out, `"path"`) {
		t.Fatalf("add json output = %s", out)
	}
	id := extractJSONField(out, "id")
	if id == "" {
		t.Fatalf("missing id in %s", out)
	}

	out = captureStdout(t, func() {
		if err := Search([]string{"Smoke memory", "--domain", "tools", "--limit", "5", "--json"}); err != nil {
			t.Fatalf("Search flags-after-query: %v", err)
		}
	})
	if !strings.Contains(out, id) || !strings.Contains(out, `"hits"`) || !strings.Contains(out, `"importance": 5`) {
		t.Fatalf("search json output = %s", out)
	}

	out = captureStdout(t, func() {
		if err := Get([]string{id, "--json"}); err != nil {
			t.Fatalf("Get json: %v", err)
		}
	})
	if !strings.Contains(out, `"body"`) || !strings.Contains(out, "Smoke memory text") || !strings.Contains(out, `"importance": 5`) || !strings.Contains(out, `"type": "uses_tool"`) {
		t.Fatalf("get json output = %s", out)
	}

	out = captureStdout(t, func() {
		if err := Update([]string{id, "--title", "Updated Memory", "--body", "Updated body", "--tags", "edited,smoke", "--project", "recall", "--lifecycle", "evergreen", "--source", "cli-test", "--links", "01LINK000000000000000001", "--relationships", `[{"target_id":"01TARGET000000000000000002","type":"depends_on","note":"updated edge"}]`, "--importance", "4", "--json"}); err != nil {
			t.Fatalf("Update json: %v", err)
		}
	})
	for _, want := range []string{`"id": "` + id + `"`, `"path"`, `"title": "Updated Memory"`, `"body": "Updated body"`, `"project": "recall"`, `"source": "cli-test"`, `"importance": 4`, `"type": "depends_on"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("update json output missing %q: %s", want, out)
		}
	}

	out = captureStdout(t, func() {
		if err := Get([]string{id, "--json"}); err != nil {
			t.Fatalf("Get updated json: %v", err)
		}
	})
	if !strings.Contains(out, `"title": "Updated Memory"`) || !strings.Contains(out, "Updated body") || !strings.Contains(out, `"path"`) {
		t.Fatalf("get updated json output = %s", out)
	}

	if err := Add([]string{"--title", "Second Memory", "--domain", "tools", "--body", "Another memory"}); err != nil {
		t.Fatalf("Add second memory: %v", err)
	}
	out = captureStdout(t, func() {
		if err := Doctor([]string{"--json"}); err != nil {
			t.Fatalf("Doctor after multiple memories: %v", err)
		}
	})
	if !strings.Contains(out, `"memories": 2`) {
		t.Fatalf("doctor should count all memories, got %s", out)
	}

	out = captureStdout(t, func() {
		if err := Delete([]string{id, "--yes"}); err != nil {
			t.Fatalf("Delete: %v", err)
		}
	})
	if !strings.Contains(out, "deleted "+id) {
		t.Fatalf("delete output = %s", out)
	}

	out = captureStdout(t, func() {
		if err := Search([]string{"Smoke"}); err != nil {
			t.Fatalf("Search after delete: %v", err)
		}
	})
	if !strings.Contains(out, "no matches") {
		t.Fatalf("search after delete = %s", out)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

func extractJSONField(s, field string) string {
	needle := `"` + field + `": "`
	i := strings.Index(s, needle)
	if i < 0 {
		return ""
	}
	start := i + len(needle)
	end := strings.Index(s[start:], `"`)
	if end < 0 {
		return ""
	}
	return s[start : start+end]
}

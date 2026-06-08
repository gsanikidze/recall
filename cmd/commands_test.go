package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
		if err := Add([]string{"--title", "Smoke Memory", "--domain", "tools", "--body", "Smoke memory text", "--json"}); err != nil {
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
	if !strings.Contains(out, id) || !strings.Contains(out, `"hits"`) {
		t.Fatalf("search json output = %s", out)
	}

	out = captureStdout(t, func() {
		if err := Get([]string{id, "--json"}); err != nil {
			t.Fatalf("Get json: %v", err)
		}
	})
	if !strings.Contains(out, `"body"`) || !strings.Contains(out, "Smoke memory text") {
		t.Fatalf("get json output = %s", out)
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

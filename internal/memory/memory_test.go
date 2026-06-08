package memory

import (
	"strings"
	"testing"
)

func mustDate(t *testing.T, s string) Date {
	t.Helper()
	d, err := ParseDate(s)
	if err != nil {
		t.Fatalf("ParseDate(%q): %v", s, err)
	}
	return d
}

func sampleMemory(t *testing.T) Memory {
	return Memory{
		ID:        "01J8X3QH000000000000000000",
		Title:     "Production deploys use Kamal, not Compose",
		Domain:    "tools",
		Tags:      []string{"deploy", "infra"},
		Created:   mustDate(t, "2026-06-07"),
		Updated:   mustDate(t, "2026-06-07"),
		Lifecycle: Evergreen,
		Source:    "claude-code",
		Body:      "Production deploys run through Kamal; Compose is local-dev only.",
	}
}

func TestMarshalParseRoundTrip(t *testing.T) {
	orig := sampleMemory(t)
	orig.Project = "acme"
	orig.Links = []string{"01J8Y000000000000000000000"}

	data, err := orig.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	got, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v\n---\n%s", err, data)
	}

	if got.ID != orig.ID || got.Title != orig.Title || got.Domain != orig.Domain {
		t.Errorf("scalar fields differ:\n got %+v\nwant %+v", got, orig)
	}
	if got.Project != "acme" {
		t.Errorf("project = %q, want acme", got.Project)
	}
	if got.Lifecycle != Evergreen {
		t.Errorf("lifecycle = %q, want evergreen", got.Lifecycle)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "deploy" || got.Tags[1] != "infra" {
		t.Errorf("tags = %v, want [deploy infra]", got.Tags)
	}
	if len(got.Links) != 1 || got.Links[0] != orig.Links[0] {
		t.Errorf("links = %v, want %v", got.Links, orig.Links)
	}
	if got.Created.String() != "2026-06-07" {
		t.Errorf("created = %q, want 2026-06-07", got.Created.String())
	}
	if strings.TrimSpace(got.Body) != strings.TrimSpace(orig.Body) {
		t.Errorf("body = %q, want %q", got.Body, orig.Body)
	}
}

func TestMarshalOmitsEmptyOptionals(t *testing.T) {
	m := sampleMemory(t)
	data, err := m.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	s := string(data)
	for _, banned := range []string{"project:", "expires_on:", "links:"} {
		if strings.Contains(s, banned) {
			t.Errorf("expected %q to be omitted, got:\n%s", banned, s)
		}
	}
}

func TestExpiresRoundTrip(t *testing.T) {
	m := sampleMemory(t)
	m.Lifecycle = Expires
	m.ExpiresOn = mustDate(t, "2026-12-31")

	data, err := m.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(data), `expires_on: "2026-12-31"`) {
		t.Errorf("expected expires_on in output, got:\n%s", data)
	}

	got, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.ExpiresOn.String() != "2026-12-31" {
		t.Errorf("expires_on = %q, want 2026-12-31", got.ExpiresOn.String())
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Memory)
		wantErr bool
	}{
		{"valid evergreen", func(m *Memory) {}, false},
		{"missing id", func(m *Memory) { m.ID = "" }, true},
		{"missing title", func(m *Memory) { m.Title = "" }, true},
		{"missing domain", func(m *Memory) { m.Domain = "" }, true},
		{"missing created", func(m *Memory) { m.Created = Date{} }, true},
		{"missing updated", func(m *Memory) { m.Updated = Date{} }, true},
		{"blank body", func(m *Memory) { m.Body = " \n	 " }, true},
		{"bad lifecycle", func(m *Memory) { m.Lifecycle = "sometimes" }, true},
		{"expires without date", func(m *Memory) { m.Lifecycle = Expires }, true},
		{"evergreen with expiry", func(m *Memory) { m.ExpiresOn = mustDate(t, "2026-12-31") }, true},
		{"valid expires", func(m *Memory) {
			m.Lifecycle = Expires
			m.ExpiresOn = mustDate(t, "2026-12-31")
		}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := sampleMemory(t)
			tc.mutate(&m)
			err := m.Validate()
			if tc.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Production deploys use Kamal, not Compose": "production-deploys-use-kamal-not-compose",
		"  Hello   World!!  ":                       "hello-world",
		"C++ & Go":                                  "c-go",
		"":                                          "untitled",
		"---":                                       "untitled",
	}
	for in, want := range cases {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFilename(t *testing.T) {
	m := sampleMemory(t)
	want := "2026-06-07-production-deploys-use-kamal-not-compose.md"
	if got := m.Filename(); got != want {
		t.Errorf("Filename() = %q, want %q", got, want)
	}
}

func TestParseRejectsMissingFrontmatter(t *testing.T) {
	if _, err := Parse([]byte("no frontmatter here\n")); err == nil {
		t.Error("expected error for missing frontmatter")
	}
}

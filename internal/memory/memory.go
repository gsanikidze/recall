// Package memory defines the recall memory record: its in-memory representation,
// validation rules, and the Markdown-with-YAML-frontmatter file format that is
// recall's source of truth.
package memory

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/oklog/ulid/v2"
	"gopkg.in/yaml.v3"
)

// ErrValidation marks invalid user-supplied or stored memory data.
var ErrValidation = errors.New("memory: validation failed")

// Lifecycle controls how a memory ages.
type Lifecycle string

const (
	// Evergreen memories never decay (tools, people, durable facts).
	Evergreen Lifecycle = "evergreen"
	// Expires memories become irrelevant after ExpiresOn.
	Expires Lifecycle = "expires"
)

// RelationshipType classifies a directed edge from one memory to another.
type RelationshipType string

const (
	RelationshipRelatedTo        RelationshipType = "related_to"
	RelationshipAboutProject     RelationshipType = "about_project"
	RelationshipUsesTool         RelationshipType = "uses_tool"
	RelationshipDependsOn        RelationshipType = "depends_on"
	RelationshipDecidedBy        RelationshipType = "decided_by"
	RelationshipSupersedes       RelationshipType = "supersedes"
	RelationshipContradicts      RelationshipType = "contradicts"
	RelationshipReferencesPerson RelationshipType = "references_person"
)

var validRelationshipTypes = map[RelationshipType]struct{}{
	RelationshipRelatedTo:        {},
	RelationshipAboutProject:     {},
	RelationshipUsesTool:         {},
	RelationshipDependsOn:        {},
	RelationshipDecidedBy:        {},
	RelationshipSupersedes:       {},
	RelationshipContradicts:      {},
	RelationshipReferencesPerson: {},
}

var relationshipTypeNames = []string{
	string(RelationshipRelatedTo),
	string(RelationshipAboutProject),
	string(RelationshipUsesTool),
	string(RelationshipDependsOn),
	string(RelationshipDecidedBy),
	string(RelationshipSupersedes),
	string(RelationshipContradicts),
	string(RelationshipReferencesPerson),
}

// RelationshipTypeNames returns supported relationship type names.
func RelationshipTypeNames() []string {
	return append([]string(nil), relationshipTypeNames...)
}

// Relationship is a directed graph edge from this memory to TargetID.
type Relationship struct {
	TargetID string           `yaml:"target_id" json:"target_id"`
	Type     RelationshipType `yaml:"type" json:"type"`
	Note     string           `yaml:"note,omitempty" json:"note,omitempty"`
}

// Memory is a single recall record: one fact, stored as one Markdown file.
type Memory struct {
	ID            string         // ULID, links the MD file to its SQLite row
	Title         string         // short headline
	Domain        string         // folder name (tools, people, projects, ...)
	Tags          []string       // free-form labels
	Project       string         // optional grouping key (e.g. "acme")
	Created       Date           // creation date
	Updated       Date           // last-modified date
	Importance    int            // 1 (low) through 5 (critical), defaults to 3
	Lifecycle     Lifecycle      // evergreen | expires
	ExpiresOn     Date           // hard expiry date; required iff Lifecycle == Expires
	Source        string         // who/what produced it: agent name, person, or URL
	Links         []string       // legacy ids of related memories
	Relationships []Relationship // typed directed edges to other memories
	Body          string         // the fact itself, as Markdown
}

// frontmatter is the YAML header serialized at the top of each memory file.
// Optional fields use omitempty so emitted files stay clean.
type frontmatter struct {
	ID            string         `yaml:"id"`
	Title         string         `yaml:"title"`
	Domain        string         `yaml:"domain"`
	Tags          []string       `yaml:"tags,omitempty"`
	Project       string         `yaml:"project,omitempty"`
	Created       Date           `yaml:"created"`
	Updated       Date           `yaml:"updated"`
	Importance    int            `yaml:"importance,omitempty"`
	Lifecycle     string         `yaml:"lifecycle"`
	ExpiresOn     *Date          `yaml:"expires_on,omitempty"`
	Source        string         `yaml:"source,omitempty"`
	Links         []string       `yaml:"links,omitempty"`
	Relationships []Relationship `yaml:"relationships,omitempty"`
}

// NewID returns a fresh lexicographically-sortable ULID string.
func NewID() string {
	return ulid.Make().String()
}

// NormalizeLifecycle resolves the string lifecycle/expires_on inputs that
// frontends accept into typed values: an empty lifecycle defaults to evergreen,
// evergreen clears any expiry, and expires requires a valid expires_on date.
// Validate enforces the same invariants on a built Memory; this is the entry
// point for constructing one from raw input.
func NormalizeLifecycle(lifecycle, expiresOn string) (Lifecycle, Date, error) {
	if lifecycle == "" {
		lifecycle = string(Evergreen)
	}
	switch Lifecycle(lifecycle) {
	case Evergreen:
		return Evergreen, Date{}, nil
	case Expires:
		if expiresOn == "" {
			return "", Date{}, fmt.Errorf("%w: lifecycle 'expires' requires expires_on (YYYY-MM-DD)", ErrValidation)
		}
		d, err := ParseDate(expiresOn)
		if err != nil {
			return "", Date{}, fmt.Errorf("%w: %v", ErrValidation, err)
		}
		return Expires, d, nil
	default:
		return "", Date{}, fmt.Errorf("%w: lifecycle must be 'evergreen' or 'expires', got %q", ErrValidation, lifecycle)
	}
}

var fenceSplit = regexp.MustCompile(`(?m)^---[ \t]*\r?\n`)

// Parse reads a memory file (YAML frontmatter delimited by `---` fences,
// followed by the Markdown body).
func Parse(data []byte) (Memory, error) {
	text := string(data)
	// A memory file must open with a `---` fence.
	loc := fenceSplit.FindStringIndex(text)
	if loc == nil || loc[0] != 0 {
		return Memory{}, fmt.Errorf("memory: missing opening --- frontmatter fence")
	}
	rest := text[loc[1]:]

	// Find the closing fence.
	closing := fenceSplit.FindStringIndex(rest)
	if closing == nil {
		return Memory{}, fmt.Errorf("memory: missing closing --- frontmatter fence")
	}
	yamlPart := rest[:closing[0]]
	body := rest[closing[1]:]

	var fm frontmatter
	if err := yaml.Unmarshal([]byte(yamlPart), &fm); err != nil {
		return Memory{}, fmt.Errorf("memory: parsing frontmatter: %w", err)
	}

	m := Memory{
		ID:            fm.ID,
		Title:         fm.Title,
		Domain:        fm.Domain,
		Tags:          fm.Tags,
		Project:       fm.Project,
		Created:       fm.Created,
		Updated:       fm.Updated,
		Importance:    fm.Importance,
		Lifecycle:     Lifecycle(fm.Lifecycle),
		Source:        fm.Source,
		Links:         fm.Links,
		Relationships: fm.Relationships,
		Body:          strings.TrimSpace(body) + "\n",
	}
	if m.Importance == 0 {
		m.Importance = 3
	}
	if len(m.Relationships) == 0 && len(m.Links) > 0 {
		m.Relationships = make([]Relationship, 0, len(m.Links))
		for _, link := range m.Links {
			m.Relationships = append(m.Relationships, Relationship{TargetID: link, Type: RelationshipRelatedTo})
		}
	}
	if fm.ExpiresOn != nil {
		m.ExpiresOn = *fm.ExpiresOn
	}
	return m, nil
}

// Marshal renders the memory back to its on-disk Markdown+frontmatter form.
func (m Memory) Marshal() ([]byte, error) {
	fm := frontmatter{
		ID:            m.ID,
		Title:         m.Title,
		Domain:        m.Domain,
		Tags:          m.Tags,
		Project:       m.Project,
		Created:       m.Created,
		Updated:       m.Updated,
		Importance:    m.Importance,
		Lifecycle:     string(m.Lifecycle),
		Source:        m.Source,
		Links:         m.Links,
		Relationships: m.Relationships,
	}
	if !m.ExpiresOn.IsZero() {
		d := m.ExpiresOn
		fm.ExpiresOn = &d
	}

	var yamlBuf bytes.Buffer
	enc := yaml.NewEncoder(&yamlBuf)
	enc.SetIndent(2)
	if err := enc.Encode(fm); err != nil {
		return nil, fmt.Errorf("memory: encoding frontmatter: %w", err)
	}
	_ = enc.Close()

	var out bytes.Buffer
	out.WriteString("---\n")
	out.Write(yamlBuf.Bytes())
	out.WriteString("---\n\n")
	out.WriteString(strings.TrimSpace(m.Body))
	out.WriteString("\n")
	return out.Bytes(), nil
}

// Validate checks the invariants every stored memory must satisfy.
func (m Memory) Validate() error {
	if m.ID == "" {
		return fmt.Errorf("%w: id is required", ErrValidation)
	}
	if strings.TrimSpace(m.Title) == "" {
		return fmt.Errorf("%w: title is required", ErrValidation)
	}
	if strings.TrimSpace(m.Domain) == "" {
		return fmt.Errorf("%w: domain is required", ErrValidation)
	}
	if m.Created.IsZero() {
		return fmt.Errorf("%w: created date is required", ErrValidation)
	}
	if m.Updated.IsZero() {
		return fmt.Errorf("%w: updated date is required", ErrValidation)
	}
	if strings.TrimSpace(m.Body) == "" {
		return fmt.Errorf("%w: body is required", ErrValidation)
	}
	if m.Importance < 1 || m.Importance > 5 {
		return fmt.Errorf("%w: importance must be between 1 and 5, got %d", ErrValidation, m.Importance)
	}
	seenRelationships := make(map[string]struct{}, len(m.Relationships))
	for i, rel := range m.Relationships {
		if strings.TrimSpace(rel.TargetID) == "" {
			return fmt.Errorf("%w: relationship %d target_id is required", ErrValidation, i)
		}
		if rel.TargetID == m.ID {
			return fmt.Errorf("%w: relationship %d must not target its source memory", ErrValidation, i)
		}
		if _, ok := validRelationshipTypes[rel.Type]; !ok {
			return fmt.Errorf("%w: relationship %d type must be one of %s, got %q", ErrValidation, i, strings.Join(RelationshipTypeNames(), ", "), rel.Type)
		}
		if len(rel.Note) > 300 {
			return fmt.Errorf("%w: relationship %d note must be at most 300 chars", ErrValidation, i)
		}
		key := rel.TargetID + "\x00" + string(rel.Type)
		if _, exists := seenRelationships[key]; exists {
			return fmt.Errorf("%w: duplicate relationship target_id/type: %s/%s", ErrValidation, rel.TargetID, rel.Type)
		}
		seenRelationships[key] = struct{}{}
	}
	switch m.Lifecycle {
	case Evergreen:
		if !m.ExpiresOn.IsZero() {
			return fmt.Errorf("%w: evergreen memory must not set expires_on", ErrValidation)
		}
	case Expires:
		if m.ExpiresOn.IsZero() {
			return fmt.Errorf("%w: lifecycle 'expires' requires expires_on", ErrValidation)
		}
	default:
		return fmt.Errorf("%w: lifecycle must be 'evergreen' or 'expires', got %q", ErrValidation, m.Lifecycle)
	}
	return nil
}

var (
	nonSlug      = regexp.MustCompile(`[^a-z0-9]+`)
	trimDashes   = regexp.MustCompile(`^-+|-+$`)
	maxSlugChars = 60
)

// Slugify converts a title into a filename-safe, lowercase, dash-separated slug.
func Slugify(title string) string {
	s := strings.ToLower(title)
	s = nonSlug.ReplaceAllString(s, "-")
	s = trimDashes.ReplaceAllString(s, "")
	if len(s) > maxSlugChars {
		s = s[:maxSlugChars]
		s = trimDashes.ReplaceAllString(s, "")
	}
	if s == "" {
		s = "untitled"
	}
	return s
}

// Filename returns the memory's file name: YYYY-MM-DD-<slug>.md, dated by Created.
func (m Memory) Filename() string {
	date := m.Created
	if date.IsZero() {
		date = Today()
	}
	return fmt.Sprintf("%s-%s.md", date.String(), Slugify(m.Title))
}

// Package vault is recall's source of truth: the directory of Markdown memory
// files, organized into domain folders. It handles reading, writing, scanning,
// and the self-describing README files that tell humans and agents what each
// domain is for.
package vault

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// ErrInvalidDomain is returned when a domain name is not a safe vault folder.
var ErrInvalidDomain = errors.New("invalid domain")

// ErrDomainExists is returned when creating a domain that already exists.
var ErrDomainExists = errors.New("domain already exists")

// readmeName is the per-domain (and root) description/index file.
const readmeName = "README.md"

// Domain is a category folder inside the vault.
type Domain struct {
	Name        string // folder name, e.g. "tools"
	Description string // one-line purpose, shown to humans and agents
}

// PredefinedDomains are scaffolded by `recall init`. Users/agents may add more.
var PredefinedDomains = []Domain{
	{"tools", "Reusable tools, commands, libraries, and infra facts. Usually evergreen."},
	{"inbox", "Uncategorized memories that don't fit another domain yet. Triage and move them later."},
	{"people", "Facts about people: role, team, preferences, how to work with them."},
	{"projects", "Project-specific memories. Group related ones with the `project:` frontmatter field."},
	{"decisions", "Decisions made during conversations, together with their rationale."},
	{"research", "Findings, results, and references worth keeping."},
	{"goals", "User and agent goals."},
}

// domainNamePattern restricts domain folder names to safe single path segments.
var domainNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// Vault is a handle to a vault directory.
type Vault struct {
	root string
	mu   sync.Mutex
}

// Open returns a Vault rooted at the given directory. It does not create it.
func Open(root string) *Vault {
	return &Vault{root: root}
}

// Root returns the vault's absolute root directory.
func (v *Vault) Root() string { return v.root }

// DomainPath returns the absolute path to a domain folder.
func (v *Vault) DomainPath(name string) string {
	return filepath.Join(v.root, name)
}

// Scaffold creates the vault root, every predefined domain (folder + README),
// and the top-level README index. It is idempotent: existing folders/READMEs
// are left untouched, except the root index which is always regenerated.
func (v *Vault) Scaffold() error {
	if err := os.MkdirAll(v.root, 0o755); err != nil {
		return fmt.Errorf("vault: creating root: %w", err)
	}
	for _, d := range PredefinedDomains {
		if err := v.ensureDomain(d); err != nil {
			return err
		}
	}
	return v.writeIndex()
}

// AddDomain creates a new domain folder + README and refreshes the root index.
func (v *Vault) AddDomain(name, description string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	name = strings.ToLower(strings.TrimSpace(name))
	if !domainNamePattern.MatchString(name) {
		return fmt.Errorf("%w: vault domain name %q (use lowercase letters, digits, dashes)", ErrInvalidDomain, name)
	}
	if info, err := os.Lstat(v.DomainPath(name)); err == nil && info.IsDir() {
		return fmt.Errorf("%w: %q", ErrDomainExists, name)
	} else if err == nil {
		return fmt.Errorf("vault: domain path %q exists but is not a directory", name)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("vault: checking domain %q: %w", name, err)
	}
	if description == "" {
		description = "(no description)"
	}
	if err := v.ensureDomain(Domain{Name: name, Description: description}); err != nil {
		return err
	}
	return v.writeIndex()
}

// ensureDomain creates the folder and its README if missing. An existing README
// is not overwritten, so hand-edited descriptions survive.
func (v *Vault) ensureDomain(d Domain) error {
	dir := v.DomainPath(d.Name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("vault: creating domain %q: %w", d.Name, err)
	}
	readme := filepath.Join(dir, readmeName)
	if _, err := os.Stat(readme); err == nil {
		return nil // keep existing README
	}
	content := fmt.Sprintf("# %s\n\n%s\n", d.Name, d.Description)
	if err := os.WriteFile(readme, []byte(content), 0o644); err != nil {
		return fmt.Errorf("vault: writing README for %q: %w", d.Name, err)
	}
	return nil
}

// HasDomain reports whether a domain folder exists. It is a cheap existence
// check (a single stat), unlike ListDomains which reads every domain README.
// Names that don't satisfy domainNamePattern are always rejected, preventing
// path traversal via crafted domain values.
func (v *Vault) HasDomain(name string) bool {
	if !domainNamePattern.MatchString(name) {
		return false
	}
	info, err := os.Lstat(v.DomainPath(name))
	return err == nil && info.IsDir()
}

// ListDomains returns every domain folder in the vault with its description
// (parsed from the domain README), sorted by name.
func (v *Vault) ListDomains() ([]Domain, error) {
	entries, err := os.ReadDir(v.root)
	if err != nil {
		return nil, fmt.Errorf("vault: reading root: %w", err)
	}
	var domains []Domain
	for _, e := range entries {
		if !e.IsDir() || !domainNamePattern.MatchString(e.Name()) {
			continue
		}
		domains = append(domains, Domain{
			Name:        e.Name(),
			Description: v.readDomainDescription(e.Name()),
		})
	}
	sort.Slice(domains, func(i, j int) bool { return domains[i].Name < domains[j].Name })
	return domains, nil
}

// readDomainDescription extracts the one-line description from a domain README:
// the first non-empty line that is not a Markdown heading.
func (v *Vault) readDomainDescription(name string) string {
	data, err := os.ReadFile(filepath.Join(v.DomainPath(name), readmeName))
	if err != nil {
		return ""
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		return line
	}
	return ""
}

// writeIndex regenerates vault/README.md: a listing of every domain and its
// one-line purpose, so the vault is self-describing for humans and agents.
func (v *Vault) writeIndex() error {
	domains, err := v.ListDomains()
	if err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("# recall vault\n\n")
	b.WriteString("Long-lived memory, organized by domain. Each folder's README says what belongs there.\n\n")
	b.WriteString("## Domains\n\n")
	for _, d := range domains {
		fmt.Fprintf(&b, "- **%s/** — %s\n", d.Name, d.Description)
	}
	path := filepath.Join(v.root, readmeName)
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("vault: writing index: %w", err)
	}
	return nil
}

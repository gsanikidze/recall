package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"recall/internal/index"
	"recall/internal/recall"
)

// Add creates a memory from flags. The body may be passed with --body or piped
// on stdin.
func Add(args []string) error {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	title := fs.String("title", "", "memory title (required)")
	body := fs.String("body", "", "memory body; if empty, read from stdin")
	domain := fs.String("domain", "", "domain folder (required)")
	tags := fs.String("tags", "", "comma-separated tags")
	project := fs.String("project", "", "project grouping key")
	lifecycle := fs.String("lifecycle", "", "evergreen (default) or expires")
	expires := fs.String("expires", "", "expiry date YYYY-MM-DD (with --lifecycle expires)")
	source := fs.String("source", "", "who/what produced this memory")
	links := fs.String("links", "", "comma-separated related memory ids")
	if err := fs.Parse(args); err != nil {
		return err
	}

	text := *body
	if text == "" {
		piped, err := readStdin()
		if err != nil {
			return err
		}
		text = piped
	}
	if strings.TrimSpace(*title) == "" || strings.TrimSpace(text) == "" || *domain == "" {
		return fmt.Errorf("add: --title, --domain, and a body (--body or stdin) are required")
	}

	e, err := openEngine()
	if err != nil {
		return err
	}
	defer e.Close()

	m, relPath, err := e.Add(context.Background(), recall.AddParams{
		Title:     *title,
		Body:      text,
		Domain:    *domain,
		Tags:      splitList(*tags),
		Project:   *project,
		Lifecycle: *lifecycle,
		ExpiresOn: *expires,
		Source:    *source,
		Links:     splitList(*links),
	})
	if err != nil {
		return err
	}
	fmt.Printf("added %s\n%s\n", m.ID, filepath.Join("vault", relPath))
	return nil
}

// Search runs a query and prints ranked hits.
func Search(args []string) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	domain := fs.String("domain", "", "restrict to a domain")
	tags := fs.String("tag", "", "comma-separated tags (match any)")
	project := fs.String("project", "", "restrict to a project")
	lifecycle := fs.String("lifecycle", "", "evergreen or expires")
	since := fs.String("since", "", "updated on/after YYYY-MM-DD")
	until := fs.String("until", "", "updated on/before YYYY-MM-DD")
	includeExpired := fs.Bool("include-expired", false, "include expired memories")
	limit := fs.Int("limit", 20, "max results")
	if err := fs.Parse(args); err != nil {
		return err
	}

	e, err := openEngine()
	if err != nil {
		return err
	}
	defer e.Close()

	hits, err := e.Search(context.Background(), index.Filter{
		Query:          strings.Join(fs.Args(), " "),
		Domain:         *domain,
		Tags:           splitList(*tags),
		Project:        *project,
		Lifecycle:      *lifecycle,
		Since:          *since,
		Until:          *until,
		IncludeExpired: *includeExpired,
		Limit:          *limit,
	})
	if err != nil {
		return err
	}
	if len(hits) == 0 {
		fmt.Println("no matches")
		return nil
	}
	for _, h := range hits {
		fmt.Printf("%s  [%s]  %s\n    %s\n", h.ID, h.Domain, h.Title, h.Snippet)
	}
	return nil
}

// Get prints a memory's path and full Markdown content.
func Get(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: recall get <id>")
	}
	e, err := openEngine()
	if err != nil {
		return err
	}
	defer e.Close()

	_, relPath, err := e.Get(context.Background(), args[0])
	if err != nil {
		return err
	}
	data, err := os.ReadFile(filepath.Join(e.Vault().Root(), relPath))
	if err != nil {
		return err
	}
	fmt.Printf("# %s\n\n%s", filepath.Join("vault", relPath), data)
	return nil
}

// Domain manages domain folders: `recall domain list` and `recall domain add`.
func Domain(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: recall domain <list|add> ...")
	}
	e, err := openEngine()
	if err != nil {
		return err
	}
	defer e.Close()

	switch args[0] {
	case "list":
		domains, err := e.Vault().ListDomains()
		if err != nil {
			return err
		}
		for _, d := range domains {
			fmt.Printf("%-12s %s\n", d.Name+"/", d.Description)
		}
		return nil
	case "add":
		fs := flag.NewFlagSet("domain add", flag.ContinueOnError)
		desc := fs.String("desc", "", "what belongs in this domain")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 1 {
			return fmt.Errorf("usage: recall domain add <name> --desc \"...\"")
		}
		if err := e.Vault().AddDomain(fs.Arg(0), *desc); err != nil {
			return err
		}
		fmt.Printf("added domain %s\n", fs.Arg(0))
		return nil
	default:
		return fmt.Errorf("recall domain: unknown subcommand %q", args[0])
	}
}

// Reindex rebuilds the index from the vault.
func Reindex(args []string) error {
	e, err := openEngine()
	if err != nil {
		return err
	}
	defer e.Close()

	stats, err := e.Reindex(context.Background())
	if err != nil {
		return err
	}
	fmt.Printf("reindexed %d memories, removed %d stale rows\n", stats.Indexed, stats.Deleted)
	return nil
}

func readStdin() (string, error) {
	info, err := os.Stdin.Stat()
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeCharDevice != 0 {
		return "", nil // interactive terminal, nothing piped
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

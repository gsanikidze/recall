package cmd

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"recall/internal/embedding"
	"recall/internal/index"
	"recall/internal/memory"
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
	relationshipsJSON := fs.String("relationships", "", "JSON array of typed relationships [{target_id,type,note}]")
	importance := fs.Int("importance", 0, "importance rank 1-5 (default 3)")
	jsonOut := fs.Bool("json", false, "print JSON")
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
	relationships, err := parseRelationshipsFlag(*relationshipsJSON)
	if err != nil {
		return err
	}

	e, err := openEngine()
	if err != nil {
		return err
	}
	defer e.Close()

	m, relPath, err := e.Add(context.Background(), recall.AddParams{
		Title:         *title,
		Body:          text,
		Domain:        *domain,
		Tags:          splitList(*tags),
		Project:       *project,
		Lifecycle:     *lifecycle,
		ExpiresOn:     *expires,
		Source:        *source,
		Links:         splitList(*links),
		Relationships: relationships,
		Importance:    *importance,
	})
	if err != nil {
		return err
	}
	if *jsonOut {
		return printJSON(struct {
			ID   string `json:"id"`
			Path string `json:"path"`
		}{ID: m.ID, Path: filepath.Join("vault", relPath)})
	}
	fmt.Printf("added %s\n%s\n", m.ID, filepath.Join("vault", relPath))
	return nil
}

func parseRelationshipsFlag(raw string) ([]memory.Relationship, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var rels []memory.Relationship
	if err := json.Unmarshal([]byte(raw), &rels); err != nil {
		return nil, fmt.Errorf("add: --relationships must be JSON array of {target_id,type,note}: %w", err)
	}
	return rels, nil
}

type embedArgs struct {
	provider string
	model    string
	baseURL  string
	force    bool
	json     bool
}

func parseEmbedArgs(args []string) (embedArgs, error) {
	parsed := embedArgs{
		provider: "ollama",
		model:    embedding.DefaultOllamaModel,
		baseURL:  os.Getenv("RECALL_OLLAMA_URL"),
	}
	fs := flag.NewFlagSet("embed", flag.ContinueOnError)
	fs.StringVar(&parsed.provider, "provider", parsed.provider, "embedding provider: ollama or fake")
	fs.StringVar(&parsed.model, "model", parsed.model, "embedding model")
	fs.StringVar(&parsed.baseURL, "base-url", parsed.baseURL, "Ollama base URL")
	fs.BoolVar(&parsed.force, "force", false, "re-embed memories even when content hash matches")
	fs.BoolVar(&parsed.json, "json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return parsed, err
	}
	if fs.NArg() != 0 {
		return parsed, fmt.Errorf("usage: recall embed [--provider ollama|fake] [--model MODEL] [--base-url URL] [--force] [--json]")
	}
	parsed.provider = strings.TrimSpace(parsed.provider)
	parsed.model = strings.TrimSpace(parsed.model)
	if parsed.provider == "" {
		return parsed, fmt.Errorf("embed: --provider is required")
	}
	if parsed.model == "" {
		return parsed, fmt.Errorf("embed: --model is required")
	}
	return parsed, nil
}

func embeddingProviderFromArgs(args embedArgs) (embedding.Provider, error) {
	switch args.provider {
	case "ollama":
		return embedding.NewOllamaProvider(args.baseURL, args.model), nil
	case "fake":
		return embedding.NewFakeProvider(args.model, 32), nil
	default:
		return nil, fmt.Errorf("embed: unknown provider %q", args.provider)
	}
}

// Embed computes embedding vectors for indexed memories and stores them in SQLite.
func Embed(args []string) error {
	parsed, err := parseEmbedArgs(args)
	if err != nil {
		return err
	}
	provider, err := embeddingProviderFromArgs(parsed)
	if err != nil {
		return err
	}
	e, err := openEngine()
	if err != nil {
		return err
	}
	defer e.Close()

	stats, err := e.EmbedAll(context.Background(), provider, parsed.force)
	if err != nil {
		return err
	}
	if parsed.json {
		return printJSON(struct {
			Provider string `json:"provider"`
			Model    string `json:"model"`
			recall.EmbedStats
		}{Provider: provider.Name(), Model: provider.Model(), EmbedStats: stats})
	}
	fmt.Printf("embedded %d memories, skipped %d, failed %d\nprovider %s model %s\n", stats.Embedded, stats.Skipped, stats.Failed, provider.Name(), provider.Model())
	return nil
}

// Search runs a query and prints ranked hits.
func Search(args []string) error {
	parsed, err := parseSearchArgs(args)
	if err != nil {
		return err
	}

	e, err := openEngine()
	if err != nil {
		return err
	}
	defer e.Close()

	if parsed.filter.Mode == index.SearchModeSemantic || parsed.filter.Mode == index.SearchModeHybrid {
		provider, err := searchEmbeddingProvider(parsed)
		if err != nil {
			return err
		}
		vectors, err := provider.Embed(context.Background(), []string{parsed.filter.Query})
		if err != nil {
			return err
		}
		if len(vectors) != 1 {
			return fmt.Errorf("search: provider returned %d query vectors, want 1", len(vectors))
		}
		parsed.filter.QueryVector = vectors[0]
		parsed.filter.Provider = provider.Name()
		parsed.filter.Model = provider.Model()
	}

	hits, err := e.Search(context.Background(), parsed.filter)
	if err != nil {
		return err
	}
	if parsed.json {
		return printJSON(struct {
			Hits []index.Hit `json:"hits"`
		}{Hits: hits})
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

type searchArgs struct {
	filter   index.Filter
	provider string
	model    string
	baseURL  string
	json     bool
}

func searchEmbeddingProvider(args searchArgs) (embedding.Provider, error) {
	switch args.provider {
	case "ollama":
		return embedding.NewOllamaProvider(args.baseURL, args.model), nil
	case "fake":
		return embedding.NewFakeProvider(args.model, 32), nil
	default:
		return nil, fmt.Errorf("search: unknown provider %q", args.provider)
	}
}

func parseSearchMode(raw string) (index.SearchMode, error) {
	switch strings.TrimSpace(raw) {
	case "", "keyword":
		return index.SearchModeKeyword, nil
	case "semantic":
		return index.SearchModeSemantic, nil
	case "hybrid":
		return index.SearchModeHybrid, nil
	default:
		return "", fmt.Errorf("search: unknown mode %q", raw)
	}
}

func setSearchMode(current *index.SearchMode, next index.SearchMode) error {
	if *current != "" && *current != next {
		return fmt.Errorf("search: --mode, --semantic, and --hybrid are mutually exclusive")
	}
	*current = next
	return nil
}

func parseSearchArgs(args []string) (searchArgs, error) {
	parsed := searchArgs{
		filter:   index.Filter{Limit: 20},
		provider: "ollama",
		model:    embedding.DefaultOllamaModel,
		baseURL:  os.Getenv("RECALL_OLLAMA_URL"),
	}
	var query []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		next := func() (string, error) {
			if i+1 >= len(args) {
				return "", fmt.Errorf("search: %s requires a value", a)
			}
			i++
			return args[i], nil
		}
		switch a {
		case "--json":
			parsed.json = true
		case "--semantic":
			if err := setSearchMode(&parsed.filter.Mode, index.SearchModeSemantic); err != nil {
				return parsed, err
			}
		case "--hybrid":
			if err := setSearchMode(&parsed.filter.Mode, index.SearchModeHybrid); err != nil {
				return parsed, err
			}
		case "--mode":
			v, err := next()
			if err != nil {
				return parsed, err
			}
			mode, err := parseSearchMode(v)
			if err != nil {
				return parsed, err
			}
			if err := setSearchMode(&parsed.filter.Mode, mode); err != nil {
				return parsed, err
			}
		case "--provider":
			v, err := next()
			if err != nil {
				return parsed, err
			}
			parsed.provider = strings.TrimSpace(v)
		case "--model":
			v, err := next()
			if err != nil {
				return parsed, err
			}
			parsed.model = strings.TrimSpace(v)
		case "--base-url":
			v, err := next()
			if err != nil {
				return parsed, err
			}
			parsed.baseURL = strings.TrimSpace(v)
		case "--include-expired":
			parsed.filter.IncludeExpired = true
		case "--domain":
			v, err := next()
			if err != nil {
				return parsed, err
			}
			parsed.filter.Domain = v
		case "--tag", "--tags":
			v, err := next()
			if err != nil {
				return parsed, err
			}
			parsed.filter.Tags = append(parsed.filter.Tags, splitList(v)...)
		case "--project":
			v, err := next()
			if err != nil {
				return parsed, err
			}
			parsed.filter.Project = v
		case "--lifecycle":
			v, err := next()
			if err != nil {
				return parsed, err
			}
			parsed.filter.Lifecycle = v
		case "--since":
			v, err := next()
			if err != nil {
				return parsed, err
			}
			parsed.filter.Since = v
		case "--until":
			v, err := next()
			if err != nil {
				return parsed, err
			}
			parsed.filter.Until = v
		case "--limit":
			v, err := next()
			if err != nil {
				return parsed, err
			}
			n, err := strconv.Atoi(v)
			if err != nil {
				return parsed, fmt.Errorf("search: --limit must be an integer")
			}
			parsed.filter.Limit = n
		default:
			if strings.HasPrefix(a, "--") {
				return parsed, fmt.Errorf("search: unknown flag %s", a)
			}
			query = append(query, a)
		}
	}
	parsed.filter.Query = strings.Join(query, " ")
	if parsed.filter.Mode == index.SearchModeSemantic || parsed.filter.Mode == index.SearchModeHybrid {
		if strings.TrimSpace(parsed.filter.Query) == "" {
			return parsed, fmt.Errorf("search: semantic and hybrid search require a query")
		}
		if parsed.provider == "" {
			return parsed, fmt.Errorf("search: --provider is required")
		}
		if parsed.model == "" {
			return parsed, fmt.Errorf("search: --model is required")
		}
	}
	return parsed, nil
}

// Get prints a memory's path and full Markdown content.
func Get(args []string) error {
	jsonOut := false
	var ids []string
	for _, a := range args {
		if a == "--json" {
			jsonOut = true
			continue
		}
		ids = append(ids, a)
	}
	if len(ids) != 1 {
		return fmt.Errorf("usage: recall get <id> [--json]")
	}
	e, err := openEngine()
	if err != nil {
		return err
	}
	defer e.Close()

	m, relPath, err := e.Get(context.Background(), ids[0])
	if err != nil {
		return err
	}
	if jsonOut {
		return printJSON(memoryOutput(m, relPath))
	}
	data, err := os.ReadFile(filepath.Join(e.Vault().Root(), relPath))
	if err != nil {
		return err
	}
	fmt.Printf("# %s\n\n%s", filepath.Join("vault", relPath), data)
	return nil
}

// Delete removes a memory by id.
func Delete(args []string) error {
	jsonOut := false
	yes := false
	var ids []string
	for _, a := range args {
		switch a {
		case "--json":
			jsonOut = true
		case "--yes", "-y":
			yes = true
		default:
			ids = append(ids, a)
		}
	}
	if len(ids) != 1 {
		return fmt.Errorf("usage: recall delete <id> --yes")
	}
	if !yes {
		return fmt.Errorf("delete requires --yes")
	}
	e, err := openEngine()
	if err != nil {
		return err
	}
	defer e.Close()
	if err := e.Delete(context.Background(), ids[0]); err != nil {
		return err
	}
	if jsonOut {
		return printJSON(struct {
			ID      string `json:"id"`
			Deleted bool   `json:"deleted"`
		}{ID: ids[0], Deleted: true})
	}
	fmt.Printf("deleted %s\n", ids[0])
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
		jsonOut := false
		for _, a := range args[1:] {
			if a == "--json" {
				jsonOut = true
				continue
			}
			return fmt.Errorf("usage: recall domain list [--json]")
		}
		domains, err := e.Vault().ListDomains()
		if err != nil {
			return err
		}
		if jsonOut {
			return printJSON(struct {
				Domains []domainOutput `json:"domains"`
			}{Domains: domainOutputs(domains)})
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

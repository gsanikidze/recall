// Package apiserver provides a JSON REST API that wraps the recall engine,
// intended for consumption by the local web UI. All endpoints are under /api/.
package apiserver

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	fibercors "github.com/gofiber/fiber/v2/middleware/cors"

	"recall/internal/embedding"
	"recall/internal/index"
	"recall/internal/memory"
	"recall/internal/recall"
	"recall/internal/vault"
	"recall/internal/view"
)

// allowedOrigins is the strict CORS allowlist (Vite dev server only).
// In production the UI is same-origin, so no Origin header is sent.
var allowedOrigins = map[string]bool{
	"http://localhost:5173": true,
	"http://127.0.0.1:5173": true,
}

// New creates a Fiber app with all /api/* routes registered, CORS and
// DNS-rebinding protection applied.
func New(e *recall.Engine) *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	s := &server{engine: e}

	api := app.Group("/api",
		dnsRebindGuard,
		fibercors.New(fibercors.Config{
			AllowOriginsFunc: func(origin string) bool { return allowedOrigins[origin] },
			AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
			AllowHeaders:     "Content-Type",
		}),
	)

	api.Get("/domains", s.listDomains)
	api.Get("/status", s.status)
	api.Post("/domains", s.createDomain)
	api.Get("/graph", s.getGraph)
	api.Get("/memories", s.listMemories)
	api.Get("/memories/:id", s.getMemory)
	api.Post("/memories", s.createMemory)
	api.Put("/memories/:id", s.updateMemory)
	api.Delete("/memories/:id", s.deleteMemory)
	api.Post("/reindex", s.reindex)

	return app
}

type server struct{ engine *recall.Engine }

// ---- response types ----

type domainJSON struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type hitJSON struct {
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

type statusJSON struct {
	ProjectPath string `json:"project_path"`
	VaultPath   string `json:"vault_path"`
	DBPath      string `json:"db_path"`
}

// ---- handlers ----

func (s *server) status(c *fiber.Ctx) error {
	projectPath := s.engine.ProjectPath()
	return c.JSON(statusJSON{
		ProjectPath: projectPath,
		VaultPath:   s.engine.Vault().Root(),
		DBPath:      filepath.Join(projectPath, "db", "recall.sqlite"),
	})
}

func (s *server) listDomains(c *fiber.Ctx) error {
	domains, err := s.engine.Vault().ListDomains()
	if err != nil {
		return errResp(c, err)
	}
	out := make([]domainJSON, len(domains))
	for i, d := range domains {
		out[i] = domainJSON{Name: d.Name, Description: d.Description}
	}
	return c.JSON(fiber.Map{"domains": out})
}

func (s *server) createDomain(c *fiber.Ctx) error {
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	name := strings.ToLower(strings.TrimSpace(body.Name))
	description := strings.TrimSpace(body.Description)
	if err := s.engine.Vault().AddDomain(name, description); err != nil {
		return validationOrErrResp(c, err)
	}
	if description == "" {
		description = "(no description)"
	}
	return c.Status(fiber.StatusCreated).JSON(domainJSON{Name: name, Description: description})
}

func (s *server) getGraph(c *fiber.Ctx) error {
	graph, err := s.engine.Graph(c.Context(), c.Query("domain"))
	if err != nil {
		return errResp(c, err)
	}
	return c.JSON(graph)
}

func (s *server) listMemories(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 200)
	if limit <= 0 {
		limit = 200
	}
	var tags []string
	for part := range strings.SplitSeq(c.Query("tags"), ",") {
		if p := strings.TrimSpace(part); p != "" {
			tags = append(tags, p)
		}
	}
	mode, err := index.ParseSearchMode(c.Query("mode", "keyword"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	filter := index.Filter{
		Query:          c.Query("q"),
		Domain:         c.Query("domain"),
		Tags:           tags,
		Project:        c.Query("project"),
		Lifecycle:      c.Query("lifecycle"),
		Since:          c.Query("since"),
		Until:          c.Query("until"),
		IncludeExpired: c.QueryBool("include_expired"),
		Limit:          limit,
		Mode:           mode,
	}
	if mode == index.SearchModeSemantic || mode == index.SearchModeHybrid {
		providerName := strings.TrimSpace(c.Query("provider", "ollama"))
		model := strings.TrimSpace(c.Query("model", embedding.DefaultOllamaModel))
		baseURL := strings.TrimSpace(c.Query("base_url", os.Getenv("RECALL_OLLAMA_URL")))
		provider, err := embedding.NewProvider(providerName, model, baseURL)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		vectors, err := provider.Embed(c.Context(), []string{filter.Query})
		if err != nil {
			return errResp(c, err)
		}
		if len(vectors) != 1 {
			return errResp(c, fmt.Errorf("api: provider returned %d query vectors, want 1", len(vectors)))
		}
		filter.QueryVector = vectors[0]
		filter.Provider = provider.Name()
		filter.Model = provider.Model()
	}
	hits, err := s.engine.Search(c.Context(), filter)
	if err != nil {
		return validationOrErrResp(c, err)
	}
	out := make([]hitJSON, len(hits))
	for i, h := range hits {
		out[i] = hitJSON{ID: h.ID, Title: h.Title, Domain: h.Domain, Snippet: h.Snippet, Path: h.Path, Importance: h.Importance, Score: h.Score, KeywordScore: h.KeywordScore, SemanticScore: h.SemanticScore}
	}
	return c.JSON(fiber.Map{"memories": out})
}

func (s *server) getMemory(c *fiber.Ctx) error {
	m, relPath, err := s.engine.Get(c.Context(), c.Params("id"))
	if errors.Is(err, recall.ErrNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	if err != nil {
		return errResp(c, err)
	}
	return c.JSON(view.FromMemory(m, relPath))
}

func (s *server) createMemory(c *fiber.Ctx) error {
	var body struct {
		Title         string                `json:"title"`
		Body          string                `json:"body"`
		Domain        string                `json:"domain"`
		Tags          []string              `json:"tags"`
		Project       string                `json:"project"`
		Lifecycle     string                `json:"lifecycle"`
		ExpiresOn     string                `json:"expires_on"`
		Source        string                `json:"source"`
		Links         []string              `json:"links"`
		Relationships []memory.Relationship `json:"relationships"`
		Importance    int                   `json:"importance"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	m, relPath, err := s.engine.Add(c.Context(), recall.AddParams{
		Title: body.Title, Body: body.Body, Domain: body.Domain,
		Tags: body.Tags, Project: body.Project, Lifecycle: body.Lifecycle,
		ExpiresOn: body.ExpiresOn, Source: body.Source, Links: body.Links,
		Relationships: body.Relationships, Importance: body.Importance,
	})
	if err != nil {
		return validationOrErrResp(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": m.ID, "path": relPath})
}

func (s *server) updateMemory(c *fiber.Ctx) error {
	// Pointer fields: absent JSON key → nil → no change. Present key → non-nil → applied.
	var body struct {
		Title         *string                `json:"title"`
		Body          *string                `json:"body"`
		Tags          *[]string              `json:"tags"`
		Project       *string                `json:"project"`
		Lifecycle     *string                `json:"lifecycle"`
		ExpiresOn     *string                `json:"expires_on"`
		Source        *string                `json:"source"`
		Links         *[]string              `json:"links"`
		Relationships *[]memory.Relationship `json:"relationships"`
		Importance    *int                   `json:"importance"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	m, relPath, err := s.engine.Update(c.Context(), c.Params("id"), recall.UpdateParams{
		Title: body.Title, Body: body.Body, Tags: body.Tags, Project: body.Project,
		Lifecycle: body.Lifecycle, ExpiresOn: body.ExpiresOn, Source: body.Source,
		Links: body.Links, Relationships: body.Relationships, Importance: body.Importance,
	})
	if errors.Is(err, recall.ErrNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	if err != nil {
		return validationOrErrResp(c, err)
	}
	return c.JSON(view.FromMemory(m, relPath))
}

func (s *server) deleteMemory(c *fiber.Ctx) error {
	err := s.engine.Delete(c.Context(), c.Params("id"))
	if errors.Is(err, recall.ErrNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	if err != nil {
		return errResp(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (s *server) reindex(c *fiber.Ctx) error {
	stats, err := s.engine.Reindex(c.Context())
	if err != nil {
		return errResp(c, err)
	}
	return c.JSON(fiber.Map{"indexed": stats.Indexed, "deleted": stats.Deleted})
}

// ---- helpers ----

func errResp(c *fiber.Ctx, err error) error {
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
}

func validationOrErrResp(c *fiber.Ctx, err error) error {
	status := fiber.StatusInternalServerError
	if errors.Is(err, vault.ErrDomainExists) {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
	}
	if errors.Is(err, recall.ErrValidation) || errors.Is(err, memory.ErrValidation) || errors.Is(err, index.ErrInvalidFilter) || errors.Is(err, vault.ErrInvalidDomain) {
		status = fiber.StatusUnprocessableEntity
	}
	return c.Status(status).JSON(fiber.Map{"error": err.Error()})
}

// dnsRebindGuard rejects requests whose Host header is not a loopback address,
// preventing DNS-rebinding attacks on the unauthenticated local API.
func dnsRebindGuard(c *fiber.Ctx) error {
	if !isSafeHost(c.Hostname()) {
		return c.Status(fiber.StatusForbidden).SendString("forbidden")
	}
	return c.Next()
}

func isSafeHost(host string) bool {
	// Fiber's c.Hostname() returns host+port from fasthttp; strip the port.
	h, _, err := net.SplitHostPort(host)
	if err != nil {
		h = host // no port present
	}
	h = strings.Trim(h, "[]")
	switch h {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	return false
}

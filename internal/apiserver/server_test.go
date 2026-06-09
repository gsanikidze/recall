package apiserver

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"recall/internal/memory"
	"recall/internal/recall"
)

func newTestApp(t *testing.T) (*recall.Engine, *fiber.App) {
	t.Helper()
	e, err := recall.Open(t.TempDir())
	if err != nil {
		t.Fatalf("recall.Open: %v", err)
	}
	t.Cleanup(func() { _ = e.Close() })
	if err := e.Vault().Scaffold(); err != nil {
		t.Fatalf("Scaffold: %v", err)
	}
	return e, New(e)
}

func doReq(t *testing.T, app *fiber.App, method, path, body string) *http.Response {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, bytes.NewBufferString(body))
		r.Header.Set("Content-Type", "application/json")
	}
	r.Host = "localhost"
	resp, err := app.Test(r)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	return resp
}

func decodeJSON(t *testing.T, resp *http.Response, out any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func TestDNSRebindGuardAllowsLoopbackHosts(t *testing.T) {
	_, app := newTestApp(t)
	for _, host := range []string{"localhost", "127.0.0.1", "[::1]"} {
		t.Run(host, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/domains", nil)
			req.Host = host
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Test: %v", err)
			}
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want 200", resp.StatusCode)
			}
		})
	}
}

func TestDNSRebindGuardRejectsNonLoopbackHost(t *testing.T) {
	_, app := newTestApp(t)
	req := httptest.NewRequest(http.MethodGet, "/api/domains", nil)
	req.Host = "example.com"
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

func TestCORSAllowlistForViteOrigins(t *testing.T) {
	_, app := newTestApp(t)
	for _, origin := range []string{"http://localhost:5173", "http://127.0.0.1:5173"} {
		t.Run(origin, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodOptions, "/api/memories", nil)
			req.Host = "localhost"
			req.Header.Set("Origin", origin)
			req.Header.Set("Access-Control-Request-Method", "POST")
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Test: %v", err)
			}
			if got := resp.Header.Get("Access-Control-Allow-Origin"); got != origin {
				t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, origin)
			}
		})
	}
}

func TestMemoryCRUDHappyPath(t *testing.T) {
	_, app := newTestApp(t)

	create := doReq(t, app, http.MethodPost, "/api/memories", `{"title":"API memory","body":"api body","domain":"tools","tags":["api"],"importance":5,"relationships":[{"target_id":"01TARGET000000000000000001","type":"uses_tool","note":"api edge"}]}`)
	if create.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", create.StatusCode)
	}
	var created struct {
		ID string `json:"id"`
	}
	decodeJSON(t, create, &created)
	if created.ID == "" {
		t.Fatal("created id empty")
	}

	get := doReq(t, app, http.MethodGet, "/api/memories/"+created.ID, "")
	if get.StatusCode != http.StatusOK {
		t.Fatalf("get status = %d, want 200", get.StatusCode)
	}
	var got memoryJSON
	decodeJSON(t, get, &got)
	if got.Title != "API memory" || got.Body == "" || got.Path == "" || got.Importance != 5 {
		t.Fatalf("get body = %+v", got)
	}
	if len(got.Relationships) != 1 || got.Relationships[0].Type != memory.RelationshipUsesTool || got.Relationships[0].Note != "api edge" {
		t.Fatalf("get relationships = %+v", got.Relationships)
	}

	list := doReq(t, app, http.MethodGet, "/api/memories?q=api", "")
	if list.StatusCode != http.StatusOK {
		t.Fatalf("list status = %d, want 200", list.StatusCode)
	}
	var listed struct {
		Memories []hitJSON `json:"memories"`
	}
	decodeJSON(t, list, &listed)
	if len(listed.Memories) != 1 || listed.Memories[0].ID != created.ID || listed.Memories[0].Importance != 5 {
		t.Fatalf("list = %+v", listed)
	}

	update := doReq(t, app, http.MethodPut, "/api/memories/"+created.ID, `{"body":"updated widgets","importance":4}`)
	if update.StatusCode != http.StatusOK {
		t.Fatalf("update status = %d, want 200", update.StatusCode)
	}
	var updated memoryJSON
	decodeJSON(t, update, &updated)
	if updated.Importance != 4 {
		t.Fatalf("updated importance = %d, want 4", updated.Importance)
	}

	deleteResp := doReq(t, app, http.MethodDelete, "/api/memories/"+created.ID, "")
	if deleteResp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", deleteResp.StatusCode)
	}
}

func TestInvalidJSONReturns400(t *testing.T) {
	_, app := newTestApp(t)
	resp := doReq(t, app, http.MethodPost, "/api/memories", `{`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestCreateDomainEndpointAddsDomain(t *testing.T) {
	_, app := newTestApp(t)

	create := doReq(t, app, http.MethodPost, "/api/domains", `{"name":"personal-notes","description":"Private notes"}`)
	if create.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(create.Body)
		t.Fatalf("create status = %d, want 201 body=%s", create.StatusCode, body)
	}
	var created domainJSON
	decodeJSON(t, create, &created)
	if created.Name != "personal-notes" || created.Description != "Private notes" {
		t.Fatalf("created = %+v", created)
	}

	list := doReq(t, app, http.MethodGet, "/api/domains", "")
	var listed struct {
		Domains []domainJSON `json:"domains"`
	}
	decodeJSON(t, list, &listed)
	found := false
	for _, d := range listed.Domains {
		if d.Name == "personal-notes" && d.Description == "Private notes" {
			found = true
		}
	}
	if !found {
		t.Fatalf("created domain not listed: %+v", listed.Domains)
	}
}

func TestCreateDomainValidationReturns422(t *testing.T) {
	_, app := newTestApp(t)
	resp := doReq(t, app, http.MethodPost, "/api/domains", `{"name":"Bad Name"}`)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 422 body=%s", resp.StatusCode, body)
	}
}

func TestCreateDomainDuplicateReturns409(t *testing.T) {
	_, app := newTestApp(t)
	first := doReq(t, app, http.MethodPost, "/api/domains", `{"name":"personal-notes","description":"First"}`)
	if first.StatusCode != http.StatusCreated {
		t.Fatalf("first status = %d, want 201", first.StatusCode)
	}
	duplicate := doReq(t, app, http.MethodPost, "/api/domains", `{"name":"personal-notes","description":"Second"}`)
	if duplicate.StatusCode != http.StatusConflict {
		body, _ := io.ReadAll(duplicate.Body)
		t.Fatalf("duplicate status = %d, want 409 body=%s", duplicate.StatusCode, body)
	}
}

func TestUnknownDomainReturns422(t *testing.T) {
	_, app := newTestApp(t)
	resp := doReq(t, app, http.MethodPost, "/api/memories", `{"title":"x","body":"y","domain":"nope"}`)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}

func TestMissingMemoryReturns404(t *testing.T) {
	_, app := newTestApp(t)
	resp := doReq(t, app, http.MethodGet, "/api/memories/missing", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestReindexEndpointReturnsStats(t *testing.T) {
	e, app := newTestApp(t)
	_, _, err := e.Add(context.Background(), recall.AddParams{Title: "Needs reindex", Body: "body", Domain: "tools"})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	resp := doReq(t, app, http.MethodPost, "/api/reindex", "{}")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var stats struct {
		Indexed int `json:"indexed"`
	}
	decodeJSON(t, resp, &stats)
	if stats.Indexed != 1 {
		t.Fatalf("indexed = %d, want 1", stats.Indexed)
	}
}

func TestSearchInvalidFilterReturns422(t *testing.T) {
	_, app := newTestApp(t)
	resp := doReq(t, app, http.MethodGet, "/api/memories?lifecycle=bad", "")
	if resp.StatusCode != http.StatusUnprocessableEntity {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 422 body=%s", resp.StatusCode, body)
	}
}

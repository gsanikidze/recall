package embedding

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOllamaProviderEmbedsOnePrompt(t *testing.T) {
	var gotPath string
	var gotBody struct {
		Model  string `json:"model"`
		Prompt string `json:"prompt"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string][]float32{"embedding": {0.1, 0.2, 0.3}})
	}))
	defer server.Close()

	provider := NewOllamaProvider(server.URL, "nomic-embed-text")
	vectors, err := provider.Embed(context.Background(), []string{"hello recall"})
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if gotPath != "/api/embeddings" {
		t.Fatalf("path = %q, want /api/embeddings", gotPath)
	}
	if gotBody.Model != "nomic-embed-text" || gotBody.Prompt != "hello recall" {
		t.Fatalf("request body = %+v", gotBody)
	}
	assertVectorEqual(t, vectors[0], []float32{0.1, 0.2, 0.3})
	if provider.Name() != "ollama" {
		t.Fatalf("Name = %q, want ollama", provider.Name())
	}
	if provider.Model() != "nomic-embed-text" {
		t.Fatalf("Model = %q", provider.Model())
	}
	if provider.Dimension() != 0 {
		t.Fatalf("Dimension = %d, want 0 before first fixed-dim contract", provider.Dimension())
	}
}

func TestOllamaProviderBatchesByRequestPerPrompt(t *testing.T) {
	var prompts []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Prompt string `json:"prompt"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		prompts = append(prompts, req.Prompt)
		_ = json.NewEncoder(w).Encode(map[string][]float32{"embedding": {float32(len(prompts))}})
	}))
	defer server.Close()

	provider := NewOllamaProvider(server.URL, "embed-model")
	vectors, err := provider.Embed(context.Background(), []string{"first", "second"})
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(prompts) != 2 || prompts[0] != "first" || prompts[1] != "second" {
		t.Fatalf("prompts = %#v", prompts)
	}
	assertVectorEqual(t, vectors[0], []float32{1})
	assertVectorEqual(t, vectors[1], []float32{2})
}

func TestOllamaProviderReturnsUsefulErrors(t *testing.T) {
	t.Run("non-200", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "model missing", http.StatusNotFound)
		}))
		defer server.Close()

		provider := NewOllamaProvider(server.URL, "missing")
		_, err := provider.Embed(context.Background(), []string{"x"})
		if err == nil {
			t.Fatal("Embed succeeded, want error")
		}
		if want := "ollama embeddings: status 404"; !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want substring %q", err.Error(), want)
		}
	})

	t.Run("empty vector", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string][]float32{"embedding": {}})
		}))
		defer server.Close()

		provider := NewOllamaProvider(server.URL, "empty")
		_, err := provider.Embed(context.Background(), []string{"x"})
		if err == nil {
			t.Fatal("Embed succeeded, want error")
		}
		if want := "empty embedding"; !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want substring %q", err.Error(), want)
		}
	})
}

package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	DefaultOllamaBaseURL = "http://127.0.0.1:11434"
	DefaultOllamaModel   = "nomic-embed-text"
)

type OllamaProvider struct {
	baseURL string
	model   string
	client  *http.Client
}

func NewOllamaProvider(baseURL, model string) OllamaProvider {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = DefaultOllamaBaseURL
	}
	if strings.TrimSpace(model) == "" {
		model = DefaultOllamaModel
	}
	return OllamaProvider{
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (p OllamaProvider) Name() string { return "ollama" }

func (p OllamaProvider) Model() string { return p.model }

func (p OllamaProvider) Dimension() int { return 0 }

// BaseURL returns the configured Ollama server URL.
func (p OllamaProvider) BaseURL() string { return p.baseURL }

// ProbeResult reports whether the Ollama server is reachable and whether the
// configured model is installed locally. AvailableModels is the full list of
// pulled models, useful for surfacing "did you pull X?" hints in the UI.
type ProbeResult struct {
	Reachable       bool
	ServerError     string
	ModelAvailable  bool
	AvailableModels []string `json:",omitempty"`
}

// Prober is an optional capability a Provider may implement. Doctor uses it to
// audit whether the embedding backend is actually up and the model is pulled.
// Providers that do not talk to a remote service (e.g. FakeProvider) should
// omit this interface; doctor then treats them as always reachable.
type Prober interface {
	Probe(ctx context.Context) (*ProbeResult, error)
}

// Probe pings the Ollama server's /api/tags endpoint and verifies the
// configured model is present in the local model list. A short timeout is
// enforced so a dead server surfaces as a clear error rather than hanging the
// doctor audit.
func (p OllamaProvider) Probe(ctx context.Context) (*ProbeResult, error) {
	result := &ProbeResult{}
	probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, p.baseURL+"/api/tags", nil)
	if err != nil {
		result.ServerError = fmt.Sprintf("build request: %v", err)
		return result, nil
	}
	resp, err := p.client.Do(req)
	if err != nil {
		result.ServerError = fmt.Sprintf("server unreachable: %v", err)
		return result, nil
	}
	defer resp.Body.Close()
	result.Reachable = resp.StatusCode >= 200 && resp.StatusCode < 300
	if !result.Reachable {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		result.ServerError = fmt.Sprintf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
		return result, nil
	}
	var decoded struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		result.Reachable = true
		result.ServerError = fmt.Sprintf("decode /api/tags: %v", err)
		return result, nil
	}
	for _, m := range decoded.Models {
		result.AvailableModels = append(result.AvailableModels, m.Name)
		if baseModel(m.Name) == baseModel(p.model) {
			result.ModelAvailable = true
		}
	}
	if !result.ModelAvailable {
		result.ServerError = fmt.Sprintf("model %q not pulled; run `ollama pull %s`", p.model, p.model)
	}
	return result, nil
}

// baseModel strips the :tag suffix from an Ollama model name so that
// "nomic-embed-text" matches "nomic-embed-text:latest".
func baseModel(name string) string {
	if i := strings.IndexByte(name, ':'); i > 0 {
		return name[:i]
	}
	return name
}

func (p OllamaProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	vectors := make([][]float32, 0, len(texts))
	for _, text := range texts {
		vector, err := p.embedOne(ctx, text)
		if err != nil {
			return nil, err
		}
		vectors = append(vectors, vector)
	}
	return vectors, nil
}

func (p OllamaProvider) embedOne(ctx context.Context, text string) ([]float32, error) {
	body, err := json.Marshal(struct {
		Model  string `json:"model"`
		Prompt string `json:"prompt"`
	}{Model: p.model, Prompt: text})
	if err != nil {
		return nil, fmt.Errorf("ollama embeddings: encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("ollama embeddings: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama embeddings: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("ollama embeddings: status %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}

	var decoded struct {
		Embedding []float32 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("ollama embeddings: decode response: %w", err)
	}
	if len(decoded.Embedding) == 0 {
		return nil, fmt.Errorf("ollama embeddings: empty embedding")
	}
	return decoded.Embedding, nil
}

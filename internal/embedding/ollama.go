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

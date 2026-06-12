package recall

import (
	"strings"
	"testing"

	"recall/internal/memory"
)

func TestEmbedTextIncludesDurableFields(t *testing.T) {
	m := memory.Memory{
		Title:   "Recall vector search",
		Domain:  "projects",
		Project: "recall",
		Tags:    []string{"semantic", "local-first"},
		Source:  "Hermes plan",
		Body:    "Use **Ollama** embeddings for local semantic search.",
	}

	text := EmbedText(m)
	for _, want := range []string{
		"Title: Recall vector search",
		"Domain: projects",
		"Project: recall",
		"Tags: semantic, local-first",
		"Source: Hermes plan",
		"Use **Ollama** embeddings for local semantic search.",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("EmbedText missing %q in:\n%s", want, text)
		}
	}
}

func TestEmbedTextNormalizesWhitespace(t *testing.T) {
	m := memory.Memory{
		Title:  "  Recall\t search  ",
		Domain: " tools ",
		Body:   "hello\n\n\tworld   again",
	}

	text := EmbedText(m)
	if strings.Contains(text, "\t") || strings.Contains(text, "  ") {
		t.Fatalf("EmbedText did not normalize whitespace: %q", text)
	}
	if !strings.Contains(text, "Title: Recall search") || !strings.Contains(text, "hello world again") {
		t.Fatalf("EmbedText lost normalized content: %q", text)
	}
}

func TestEmbedContentHashChangesWhenBodyChanges(t *testing.T) {
	base := memory.Memory{Title: "Recall", Domain: "projects", Body: "local memory"}
	changed := base
	changed.Body = "local semantic memory"

	if EmbedContentHash(base) == EmbedContentHash(changed) {
		t.Fatal("hash did not change when body changed")
	}
}

func TestEmbedContentHashIgnoresWhitespaceOnlyDifferences(t *testing.T) {
	compact := memory.Memory{Title: "Recall", Domain: "projects", Body: "local memory"}
	spaced := memory.Memory{Title: " Recall ", Domain: "projects", Body: "local\n\t memory"}

	if EmbedContentHash(compact) != EmbedContentHash(spaced) {
		t.Fatalf("hash changed for whitespace-only differences: %s != %s", EmbedContentHash(compact), EmbedContentHash(spaced))
	}
}

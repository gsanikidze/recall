package recall

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"recall/internal/memory"
)

func EmbedText(m memory.Memory) string {
	var parts []string
	appendField := func(label, value string) {
		value = normalizeEmbedWhitespace(value)
		if value != "" {
			parts = append(parts, label+": "+value)
		}
	}

	appendField("Title", m.Title)
	appendField("Domain", m.Domain)
	appendField("Project", m.Project)
	if len(m.Tags) > 0 {
		tags := make([]string, 0, len(m.Tags))
		for _, tag := range m.Tags {
			if normalized := normalizeEmbedWhitespace(tag); normalized != "" {
				tags = append(tags, normalized)
			}
		}
		appendField("Tags", strings.Join(tags, ", "))
	}
	appendField("Source", m.Source)

	body := normalizeEmbedWhitespace(m.Body)
	if body != "" {
		if len(parts) > 0 {
			parts = append(parts, "")
		}
		parts = append(parts, body)
	}
	return strings.Join(parts, "\n")
}

func EmbedContentHash(m memory.Memory) string {
	sum := sha256.Sum256([]byte(EmbedText(m)))
	return hex.EncodeToString(sum[:])
}

func normalizeEmbedWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

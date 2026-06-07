package index

import (
	"regexp"
	"strings"
)

var (
	reCodeFence = regexp.MustCompile("(?s)```.*?```")
	reInlineCode = regexp.MustCompile("`([^`]*)`")
	reImage     = regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`)
	reLink      = regexp.MustCompile(`\[([^\]]+)\]\([^)]*\)`)
	reWikiLink  = regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	reHeading   = regexp.MustCompile(`(?m)^#{1,6}\s+`)
	reQuote     = regexp.MustCompile(`(?m)^>\s?`)
	reListItem  = regexp.MustCompile(`(?m)^[\s]*[-*+]\s+`)
	reEmphasis  = regexp.MustCompile(`[*_~]{1,3}`)
	reWhitespace = regexp.MustCompile(`[ \t]+`)
	reBlankLines = regexp.MustCompile(`\n{3,}`)
)

// StripMarkdown reduces Markdown to plain prose for the SQLite index: the DB is
// for LLMs, so it stores clean text without heading marks, emphasis, link, or
// code syntax. Formatting is preserved only in the source MD file.
func StripMarkdown(md string) string {
	s := md
	s = reCodeFence.ReplaceAllString(s, " ")
	s = reImage.ReplaceAllString(s, " ")
	s = reLink.ReplaceAllString(s, "$1")
	s = reWikiLink.ReplaceAllString(s, "$1")
	s = reInlineCode.ReplaceAllString(s, "$1")
	s = reHeading.ReplaceAllString(s, "")
	s = reQuote.ReplaceAllString(s, "")
	s = reListItem.ReplaceAllString(s, "")
	s = reEmphasis.ReplaceAllString(s, "")
	s = reWhitespace.ReplaceAllString(s, " ")
	s = reBlankLines.ReplaceAllString(s, "\n\n")

	var lines []string
	for line := range strings.SplitSeq(s, "\n") {
		lines = append(lines, strings.TrimSpace(line))
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

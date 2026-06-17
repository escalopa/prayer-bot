package domain

import (
	"strings"
	"unicode/utf8"
)

// StripMarkdown removes MarkdownV2 formatting for surfaces that do not support parse_mode (e.g. poll questions).
func StripMarkdown(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == '\\' && i+size < len(s) {
			next, nextSize := utf8.DecodeRuneInString(s[i+size:])
			if isMarkdownV2Reserved(next) {
				b.WriteRune(next)
				i += size + nextSize
				continue
			}
		}
		if r != '*' && r != '_' && r != '`' {
			b.WriteRune(r)
		}
		i += size
	}

	return b.String()
}

func isMarkdownV2Reserved(r rune) bool {
	switch r {
	case '_', '*', '[', ']', '(', ')', '~', '`', '>', '#', '+', '-', '=', '|', '{', '}', '.', '!', '\\':
		return true
	default:
		return false
	}
}

package domain

import (
	"fmt"
	"strings"
)

// MarkdownRaw is markdown already produced by FormatMarkdown and must not be escaped again.
type MarkdownRaw string

// EscapeMarkdown escapes characters that are special in Telegram legacy Markdown.
func EscapeMarkdown(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '*', '_', '`', '[':
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// FormatMarkdown applies a printf-style template after escaping string arguments for Markdown.
func FormatMarkdown(template string, args ...any) string {
	escaped := make([]any, len(args))
	for i, arg := range args {
		switch v := arg.(type) {
		case MarkdownRaw:
			escaped[i] = string(v)
		case string:
			escaped[i] = EscapeMarkdown(v)
		default:
			escaped[i] = arg
		}
	}
	return fmt.Sprintf(template, escaped...)
}

// StripMarkdown removes Markdown formatting for surfaces that do not support parse_mode (e.g. poll questions).
func StripMarkdown(s string) string {
	s = strings.ReplaceAll(s, "*", "")
	s = strings.ReplaceAll(s, `\(`, "(")
	s = strings.ReplaceAll(s, `\)`, ")")
	return s
}

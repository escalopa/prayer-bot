package domain

import "testing"

func TestEscapeMarkdown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in, want string
	}{
		{"plain", "plain"},
		{"*bold*", "\\*bold\\*"},
	}
	for _, tt := range tests {
		if got := EscapeMarkdown(tt.in); got != tt.want {
			t.Errorf("EscapeMarkdown(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestFormatMarkdown(t *testing.T) {
	t.Parallel()

	const template = `⏳ *%s* after *%s* (%s)`
	got := FormatMarkdown(template, "Dhuhr", "20m", "11:45")
	want := `⏳ *Dhuhr* after *20m* (11:45)`
	if got != want {
		t.Errorf("FormatMarkdown() = %q, want %q", got, want)
	}
}

func TestFormatMarkdownRaw(t *testing.T) {
	t.Parallel()

	inner := FormatMarkdown("*%s*", "bold")
	got := FormatMarkdown("before %s after", MarkdownRaw(inner))
	want := "before *bold* after"
	if got != want {
		t.Errorf("FormatMarkdown(Raw) = %q, want %q", got, want)
	}
}

func TestStripMarkdown(t *testing.T) {
	t.Parallel()

	in := `⏳ *Dhuhr* after *20m* (11:45)`
	want := `⏳ Dhuhr after 20m (11:45)`
	if got := StripMarkdown(in); got != want {
		t.Errorf("StripMarkdown() = %q, want %q", got, want)
	}
}

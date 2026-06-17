package domain

import "testing"

func TestStripMarkdown(t *testing.T) {
	t.Parallel()

	in := `⏳ *Dhuhr* after *20m* \(11:45\)`
	want := `⏳ Dhuhr after 20m (11:45)`
	if got := StripMarkdown(in); got != want {
		t.Errorf("StripMarkdown() = %q, want %q", got, want)
	}
}

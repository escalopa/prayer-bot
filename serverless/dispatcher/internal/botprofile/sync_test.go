package botprofile

import "testing"

func TestCityFromUsername(t *testing.T) {
	tests := []struct {
		username string
		want     string
	}{
		{username: "kazan_prayer_bot", want: "Kazan"},
		{username: "innopolis_test_bot", want: "Innopolis"},
		{username: "bot", want: "Bot"},
	}

	for _, tt := range tests {
		if got := cityFromUsername(tt.username); got != tt.want {
			t.Fatalf("cityFromUsername(%q) = %q, want %q", tt.username, got, tt.want)
		}
	}
}

func TestProfilePrefix(t *testing.T) {
	if got := profilePrefix("kazan_prayer_bot"); got != "" {
		t.Fatalf("profilePrefix(kazan_prayer_bot) = %q, want empty", got)
	}

	if got := profilePrefix("kazan_test_bot"); got != "[TEST] " {
		t.Fatalf("profilePrefix(kazan_test_bot) = %q, want %q", got, "[TEST] ")
	}
}

package assets

import _ "embed"

// ProfilePhoto is the static JPEG uploaded as the bot's Telegram profile photo.
//
//go:embed profile.jpg
var ProfilePhoto []byte

// WelcomePhoto is sent with the localized onboarding message on /start.
//
//go:embed welcome.jpg
var WelcomePhoto []byte

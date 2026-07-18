package botprofile

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image/jpeg"
	"net/http"
	"testing"
	"unicode/utf8"

	botapi "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/escalopa/prayer-bot/global/internal/assets"
	"github.com/escalopa/prayer-bot/global/internal/i18n"
)

func TestLocalizedCommandsAreCompleteAndWithinTelegramLimits(t *testing.T) {
	for _, locale := range i18n.Supported() {
		items := commands(locale)
		if len(items) != 11 {
			t.Fatalf("%s has %d commands, want 11", locale.Code, len(items))
		}
		seen := make(map[string]bool)
		for _, item := range items {
			if seen[item.Command] {
				t.Errorf("%s duplicates /%s", locale.Code, item.Command)
			}
			seen[item.Command] = true
			length := utf8.RuneCountInString(item.Description)
			if length < 1 || length > 256 {
				t.Errorf("%s /%s description has invalid length %d", locale.Code, item.Command, length)
			}
		}
	}
}

func TestMiniAppMenuButtonUsesDeploymentURL(t *testing.T) {
	button := miniAppMenuButton("https://example.run.app/app/")
	if button.Type != models.MenuButtonTypeWebApp || button.Text != "🕌 Prayer App" || button.WebApp.URL != "https://example.run.app/app/" {
		t.Fatalf("unexpected Mini App menu button: %+v", button)
	}
	payload, err := json.Marshal(button)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"type":"web_app","text":"🕌 Prayer App","web_app":{"url":"https://example.run.app/app/"}}`
	if string(payload) != want {
		t.Fatalf("Mini App menu button JSON = %s, want %s", payload, want)
	}

	current := models.MenuButton{
		Type:   models.MenuButtonTypeWebApp,
		WebApp: &button,
	}
	if !menuButtonEqual(current, button) {
		t.Fatal("identical Mini App menu buttons should compare equal")
	}
	differentButton := miniAppMenuButton("https://different.example/app/")
	if menuButtonEqual(current, differentButton) {
		t.Fatal("different Mini App URLs should require an update")
	}
}

func TestStableIdentityRemovesLocalizedGlobalMetadata(t *testing.T) {
	originalDelay := profileSyncDelay
	profileSyncDelay = 0
	t.Cleanup(func() { profileSyncDelay = originalDelay })

	client := newFakeIdentityClient()
	client.currentNames[""] = "Old name"
	client.currentShortDescriptions[""] = "Old short description"
	client.currentDescriptions[""] = "Old description"
	client.currentCommands[""] = []models.BotCommand{{Command: "old", Description: "Old command"}}
	for _, locale := range i18n.Supported() {
		if locale.Code == "en" {
			continue
		}
		client.currentNames[locale.Code] = "Localized name"
		client.currentShortDescriptions[locale.Code] = "Localized short description"
		client.currentDescriptions[locale.Code] = "Localized description"
		client.currentCommands[locale.Code] = []models.BotCommand{{Command: "old", Description: "Old command"}}
	}
	if err := syncStableIdentity(context.Background(), client); err != nil {
		t.Fatal(err)
	}

	if len(client.names) != len(i18n.Supported()) || client.names[0].LanguageCode != "" || client.names[0].Name != i18n.Resolve("en").BotName {
		t.Fatalf("unexpected name operations: %+v", client.names)
	}
	if len(client.shortDescriptions) != len(i18n.Supported()) || client.shortDescriptions[0].LanguageCode != "" || client.shortDescriptions[0].ShortDescription == "" {
		t.Fatalf("unexpected short-description operations: %+v", client.shortDescriptions)
	}
	if len(client.descriptions) != len(i18n.Supported()) || client.descriptions[0].LanguageCode != "" || client.descriptions[0].Description == "" {
		t.Fatalf("unexpected description operations: %+v", client.descriptions)
	}
	if len(client.commandSets) != 1 || client.commandSets[0].LanguageCode != "" || len(client.commandDeletes) != len(i18n.Supported())-1 {
		t.Fatalf("unexpected command operations: set=%+v delete=%+v", client.commandSets, client.commandDeletes)
	}
	for index := 1; index < len(client.names); index++ {
		if client.names[index].LanguageCode == "" || client.names[index].Name != "" ||
			client.shortDescriptions[index].ShortDescription != "" || client.descriptions[index].Description != "" {
			t.Fatalf("localized metadata was not cleared at index %d", index)
		}
	}
}

func TestStableIdentitySkipsUnchangedMetadata(t *testing.T) {
	originalDelay := profileSyncDelay
	profileSyncDelay = 0
	t.Cleanup(func() { profileSyncDelay = originalDelay })

	english := i18n.Resolve("en")
	client := newFakeIdentityClient()
	client.currentNames[""] = english.BotName
	client.currentShortDescriptions[""] = english.ShortDescription
	client.currentDescriptions[""] = english.Description
	client.currentCommands[""] = commands(english)

	if err := syncStableIdentity(context.Background(), client); err != nil {
		t.Fatal(err)
	}
	if len(client.names)+len(client.shortDescriptions)+len(client.descriptions)+len(client.commandSets)+len(client.commandDeletes) != 0 {
		t.Fatalf("unchanged profile generated writes: %+v", client)
	}
}

func TestRateLimitRetryAfterRecognizesWrappedTelegramError(t *testing.T) {
	err := fmt.Errorf("set bot name: %w", &botapi.TooManyRequestsError{
		Message:    "Too Many Requests",
		RetryAfter: 76903,
	})
	retryAfter, ok := RateLimitRetryAfter(err)
	if !ok || retryAfter != 76903 {
		t.Fatalf("RateLimitRetryAfter() = %d, %t", retryAfter, ok)
	}
	if _, ok := RateLimitRetryAfter(fmt.Errorf("unauthorized")); ok {
		t.Fatal("non-rate-limit error was classified as rate limited")
	}
}

func TestProfilePhotoRateLimitIsRecognized(t *testing.T) {
	result := profilePhotoAPIResponse{Description: "Too Many Requests"}
	result.Parameters.RetryAfter = 3600
	err := profilePhotoResponseError(http.StatusTooManyRequests, result)

	retryAfter, ok := RateLimitRetryAfter(err)
	if !ok || retryAfter != 3600 {
		t.Fatalf("RateLimitRetryAfter() = %d, %t", retryAfter, ok)
	}
}

func TestTransientProfileSyncRetriesThenSucceeds(t *testing.T) {
	originalDelay := transientSyncRetryDelay
	transientSyncRetryDelay = 0
	t.Cleanup(func() { transientSyncRetryDelay = originalDelay })

	attempts := 0
	err := retryTransientSync(context.Background(), func() error {
		attempts++
		if attempts < transientSyncMaxAttempts {
			return emptyTelegramResponseError(t)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if attempts != transientSyncMaxAttempts {
		t.Fatalf("profile sync attempts = %d, want %d", attempts, transientSyncMaxAttempts)
	}
}

func TestTransientProfileSyncExhaustionIsSkippable(t *testing.T) {
	originalDelay := transientSyncRetryDelay
	transientSyncRetryDelay = 0
	t.Cleanup(func() { transientSyncRetryDelay = originalDelay })

	attempts := 0
	err := retryTransientSync(context.Background(), func() error {
		attempts++
		return emptyTelegramResponseError(t)
	})
	if !IsTransientFailure(err) {
		t.Fatalf("expected a skippable transient failure, got %v", err)
	}
	if attempts != transientSyncMaxAttempts {
		t.Fatalf("profile sync attempts = %d, want %d", attempts, transientSyncMaxAttempts)
	}
}

func TestPermanentProfileSyncFailureIsNotRetried(t *testing.T) {
	permanent := errors.New("unauthorized")
	attempts := 0
	err := retryTransientSync(context.Background(), func() error {
		attempts++
		return permanent
	})
	if !errors.Is(err, permanent) || IsTransientFailure(err) {
		t.Fatalf("expected permanent failure, got %v", err)
	}
	if attempts != 1 {
		t.Fatalf("permanent profile sync attempts = %d, want 1", attempts)
	}
}

func TestRateLimitedProfileSyncIsNotRetried(t *testing.T) {
	limited := &botapi.TooManyRequestsError{Message: "Too Many Requests", RetryAfter: 60}
	attempts := 0
	err := retryTransientSync(context.Background(), func() error {
		attempts++
		return limited
	})
	retryAfter, ok := RateLimitRetryAfter(err)
	if !ok || retryAfter != 60 {
		t.Fatalf("expected rate limit to be preserved, got %v", err)
	}
	if attempts != 1 {
		t.Fatalf("rate-limited profile sync attempts = %d, want 1", attempts)
	}
}

func emptyTelegramResponseError(t *testing.T) error {
	t.Helper()
	var response any
	err := json.Unmarshal(nil, &response)
	if err == nil {
		t.Fatal("empty Telegram response unexpectedly decoded")
	}
	return fmt.Errorf("decode Telegram response: %w", err)
}

func TestProfilePhotoComparisonAllowsJPEGRecompression(t *testing.T) {
	decoded, err := jpeg.Decode(bytes.NewReader(assets.ProfilePhoto))
	if err != nil {
		t.Fatal(err)
	}
	var recompressed bytes.Buffer
	if err := jpeg.Encode(&recompressed, decoded, &jpeg.Options{Quality: 65}); err != nil {
		t.Fatal(err)
	}
	equal, err := profilePhotosEqual(recompressed.Bytes(), assets.ProfilePhoto)
	if err != nil {
		t.Fatal(err)
	}
	if !equal {
		t.Fatal("JPEG recompression should not trigger another profile photo upload")
	}

	equal, err = profilePhotosEqual(assets.WelcomePhoto, assets.ProfilePhoto)
	if err != nil {
		t.Fatal(err)
	}
	if equal {
		t.Fatal("visually different images should trigger a profile photo update")
	}
}

type fakeIdentityClient struct {
	currentNames             map[string]string
	currentShortDescriptions map[string]string
	currentDescriptions      map[string]string
	currentCommands          map[string][]models.BotCommand
	names                    []botapi.SetMyNameParams
	shortDescriptions        []botapi.SetMyShortDescriptionParams
	descriptions             []botapi.SetMyDescriptionParams
	commandSets              []botapi.SetMyCommandsParams
	commandDeletes           []botapi.DeleteMyCommandsParams
}

func newFakeIdentityClient() *fakeIdentityClient {
	return &fakeIdentityClient{
		currentNames:             make(map[string]string),
		currentShortDescriptions: make(map[string]string),
		currentDescriptions:      make(map[string]string),
		currentCommands:          make(map[string][]models.BotCommand),
	}
}

func (f *fakeIdentityClient) GetMyName(_ context.Context, params *botapi.GetMyNameParams) (models.BotName, error) {
	return models.BotName{Name: f.currentNames[params.LanguageCode]}, nil
}

func (f *fakeIdentityClient) SetMyName(_ context.Context, params *botapi.SetMyNameParams) (bool, error) {
	f.names = append(f.names, *params)
	f.currentNames[params.LanguageCode] = params.Name
	return true, nil
}

func (f *fakeIdentityClient) GetMyShortDescription(_ context.Context, params *botapi.GetMyShortDescriptionParams) (models.BotShortDescription, error) {
	return models.BotShortDescription{ShortDescription: f.currentShortDescriptions[params.LanguageCode]}, nil
}

func (f *fakeIdentityClient) SetMyShortDescription(_ context.Context, params *botapi.SetMyShortDescriptionParams) (bool, error) {
	f.shortDescriptions = append(f.shortDescriptions, *params)
	f.currentShortDescriptions[params.LanguageCode] = params.ShortDescription
	return true, nil
}

func (f *fakeIdentityClient) GetMyDescription(_ context.Context, params *botapi.GetMyDescriptionParams) (models.BotDescription, error) {
	return models.BotDescription{Description: f.currentDescriptions[params.LanguageCode]}, nil
}

func (f *fakeIdentityClient) SetMyDescription(_ context.Context, params *botapi.SetMyDescriptionParams) (bool, error) {
	f.descriptions = append(f.descriptions, *params)
	f.currentDescriptions[params.LanguageCode] = params.Description
	return true, nil
}

func (f *fakeIdentityClient) GetMyCommands(_ context.Context, params *botapi.GetMyCommandsParams) ([]models.BotCommand, error) {
	return append([]models.BotCommand(nil), f.currentCommands[params.LanguageCode]...), nil
}

func (f *fakeIdentityClient) SetMyCommands(_ context.Context, params *botapi.SetMyCommandsParams) (bool, error) {
	f.commandSets = append(f.commandSets, *params)
	f.currentCommands[params.LanguageCode] = append([]models.BotCommand(nil), params.Commands...)
	return true, nil
}

func (f *fakeIdentityClient) DeleteMyCommands(_ context.Context, params *botapi.DeleteMyCommandsParams) (bool, error) {
	f.commandDeletes = append(f.commandDeletes, *params)
	delete(f.currentCommands, params.LanguageCode)
	return true, nil
}

var _ identityClient = (*fakeIdentityClient)(nil)

package botprofile

import (
	"context"
	"encoding/json"
	"testing"
	"unicode/utf8"

	botapi "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

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
}

func TestStableIdentityRemovesLocalizedGlobalMetadata(t *testing.T) {
	originalDelay := profileSyncDelay
	profileSyncDelay = 0
	t.Cleanup(func() { profileSyncDelay = originalDelay })

	client := &fakeIdentityClient{}
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

type fakeIdentityClient struct {
	names             []botapi.SetMyNameParams
	shortDescriptions []botapi.SetMyShortDescriptionParams
	descriptions      []botapi.SetMyDescriptionParams
	commandSets       []botapi.SetMyCommandsParams
	commandDeletes    []botapi.DeleteMyCommandsParams
}

func (f *fakeIdentityClient) SetMyName(_ context.Context, params *botapi.SetMyNameParams) (bool, error) {
	f.names = append(f.names, *params)
	return true, nil
}

func (f *fakeIdentityClient) SetMyShortDescription(_ context.Context, params *botapi.SetMyShortDescriptionParams) (bool, error) {
	f.shortDescriptions = append(f.shortDescriptions, *params)
	return true, nil
}

func (f *fakeIdentityClient) SetMyDescription(_ context.Context, params *botapi.SetMyDescriptionParams) (bool, error) {
	f.descriptions = append(f.descriptions, *params)
	return true, nil
}

func (f *fakeIdentityClient) SetMyCommands(_ context.Context, params *botapi.SetMyCommandsParams) (bool, error) {
	f.commandSets = append(f.commandSets, *params)
	return true, nil
}

func (f *fakeIdentityClient) DeleteMyCommands(_ context.Context, params *botapi.DeleteMyCommandsParams) (bool, error) {
	f.commandDeletes = append(f.commandDeletes, *params)
	return true, nil
}

var _ identityClient = (*fakeIdentityClient)(nil)

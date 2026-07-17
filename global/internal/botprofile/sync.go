// Package botprofile synchronizes Telegram-facing metadata at deploy time.
package botprofile

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	botapi "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/escalopa/prayer-bot/global/internal/assets"
	"github.com/escalopa/prayer-bot/global/internal/i18n"
)

var startDescriptions = map[string]string{
	"en": "Open the bot and show the main menu",
	"ar": "فتح البوت وعرض القائمة الرئيسية",
	"es": "Abrir el bot y mostrar el menú principal",
	"fr": "Ouvrir le bot et afficher le menu principal",
	"ru": "Открыть бота и показать главное меню",
	"tr": "Botu aç ve ana menüyü göster",
	"uz": "Botni ochish va bosh menyuni ko‘rsatish",
	"tt": "Ботны ачу һәм төп менюны күрсәтү",
}

var profileSyncDelay = 50 * time.Millisecond

// Sync applies one stable Telegram-facing identity and removes old localized
// profile variants. User-selected languages belong to chat data and must never
// mutate global bot metadata. The avatar is uploaded only when the bot does not
// yet have a profile photo.
func Sync(ctx context.Context, client *botapi.Bot, token, miniAppURL string) error {
	if err := syncStableIdentity(ctx, client); err != nil {
		return err
	}
	if _, err := client.SetChatMenuButton(ctx, &botapi.SetChatMenuButtonParams{
		MenuButton: miniAppMenuButton(miniAppURL),
	}); err != nil {
		return fmt.Errorf("set Mini App menu button: %w", err)
	}

	me, err := client.GetMe(ctx)
	if err != nil {
		return fmt.Errorf("get bot identity: %w", err)
	}
	photos, err := client.GetUserProfilePhotos(ctx, &botapi.GetUserProfilePhotosParams{UserID: me.ID, Limit: 1})
	if err != nil {
		return fmt.Errorf("get bot profile photos: %w", err)
	}
	if photos.TotalCount == 0 {
		if err := setProfilePhoto(ctx, token, assets.ProfilePhoto); err != nil {
			return err
		}
	}
	return nil
}

func miniAppMenuButton(url string) models.MenuButtonWebApp {
	return models.MenuButtonWebApp{
		Type: models.MenuButtonTypeWebApp,
		Text: "🕌 Prayer App", WebApp: models.WebAppInfo{URL: url},
	}
}

type identityClient interface {
	SetMyName(context.Context, *botapi.SetMyNameParams) (bool, error)
	SetMyShortDescription(context.Context, *botapi.SetMyShortDescriptionParams) (bool, error)
	SetMyDescription(context.Context, *botapi.SetMyDescriptionParams) (bool, error)
	SetMyCommands(context.Context, *botapi.SetMyCommandsParams) (bool, error)
	DeleteMyCommands(context.Context, *botapi.DeleteMyCommandsParams) (bool, error)
}

func syncStableIdentity(ctx context.Context, client identityClient) error {
	english := i18n.Resolve("en")
	if _, err := client.SetMyName(ctx, &botapi.SetMyNameParams{Name: english.BotName}); err != nil {
		return fmt.Errorf("set default bot name: %w", err)
	}
	time.Sleep(profileSyncDelay)
	if _, err := client.SetMyShortDescription(ctx, &botapi.SetMyShortDescriptionParams{
		ShortDescription: english.ShortDescription,
	}); err != nil {
		return fmt.Errorf("set default short description: %w", err)
	}
	time.Sleep(profileSyncDelay)
	if _, err := client.SetMyDescription(ctx, &botapi.SetMyDescriptionParams{
		Description: english.Description,
	}); err != nil {
		return fmt.Errorf("set default description: %w", err)
	}
	time.Sleep(profileSyncDelay)
	if _, err := client.SetMyCommands(ctx, &botapi.SetMyCommandsParams{
		Commands: commands(english),
	}); err != nil {
		return fmt.Errorf("set default commands: %w", err)
	}
	time.Sleep(profileSyncDelay)

	// Earlier releases published localized profile metadata. Telegram selects
	// those values from the viewer's Telegram language, not the per-chat bot
	// preference, so remove them to keep the public identity consistent.
	for _, locale := range i18n.Supported() {
		if locale.Code == english.Code {
			continue
		}
		if err := clearLocalizedIdentity(ctx, client, locale.Code); err != nil {
			return err
		}
	}
	return nil
}

func clearLocalizedIdentity(ctx context.Context, client identityClient, languageCode string) error {
	if _, err := client.SetMyName(ctx, &botapi.SetMyNameParams{LanguageCode: languageCode}); err != nil {
		return fmt.Errorf("remove localized bot name for %q: %w", languageCode, err)
	}
	time.Sleep(profileSyncDelay)
	if _, err := client.SetMyShortDescription(ctx, &botapi.SetMyShortDescriptionParams{LanguageCode: languageCode}); err != nil {
		return fmt.Errorf("remove localized short description for %q: %w", languageCode, err)
	}
	time.Sleep(profileSyncDelay)
	if _, err := client.SetMyDescription(ctx, &botapi.SetMyDescriptionParams{LanguageCode: languageCode}); err != nil {
		return fmt.Errorf("remove localized description for %q: %w", languageCode, err)
	}
	time.Sleep(profileSyncDelay)
	if _, err := client.DeleteMyCommands(ctx, &botapi.DeleteMyCommandsParams{LanguageCode: languageCode}); err != nil {
		return fmt.Errorf("remove localized commands for %q: %w", languageCode, err)
	}
	time.Sleep(profileSyncDelay)
	return nil
}

func commands(locale i18n.Locale) []models.BotCommand {
	order := []string{"start", "location", "today", "tomorrow", "next", "settings", "remind", "language", "feedback", "privacy", "help"}
	result := make([]models.BotCommand, 0, len(order))
	for _, command := range order {
		description := locale.Commands[command]
		if command == "start" {
			description = startDescriptions[locale.Code]
		}
		result = append(result, models.BotCommand{Command: command, Description: description})
	}
	return result
}

func setProfilePhoto(ctx context.Context, token string, photo []byte) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("photo", `{"type":"static","photo":"attach://profile"}`); err != nil {
		return fmt.Errorf("build profile photo request: %w", err)
	}
	part, err := writer.CreateFormFile("profile", "profile.jpg")
	if err != nil {
		return fmt.Errorf("build profile photo upload: %w", err)
	}
	if _, err := part.Write(photo); err != nil {
		return fmt.Errorf("write profile photo upload: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("finish profile photo upload: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.telegram.org/bot"+token+"/setMyProfilePhoto", &body)
	if err != nil {
		return fmt.Errorf("create profile photo request: %w", err)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	httpClient := &http.Client{Timeout: 30 * time.Second}
	response, err := httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("upload bot profile photo: %w", err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read profile photo response: %w", err)
	}
	var result struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("decode profile photo response (status %d): %w", response.StatusCode, err)
	}
	if response.StatusCode != http.StatusOK || !result.OK {
		return fmt.Errorf("set bot profile photo failed (status %d): %s", response.StatusCode, result.Description)
	}
	return nil
}

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

// Sync applies the default English profile, all supported localized profiles,
// the localized command menus and the generated avatar. The avatar is uploaded
// only when the bot does not yet have a profile photo.
func Sync(ctx context.Context, client *botapi.Bot, token string) error {
	english := i18n.Resolve("en")
	if err := syncLocale(ctx, client, english, ""); err != nil {
		return err
	}
	for _, locale := range i18n.Supported() {
		if locale.Code == "en" {
			continue
		}
		if err := syncLocale(ctx, client, locale, locale.Code); err != nil {
			return err
		}
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

func syncLocale(ctx context.Context, client *botapi.Bot, locale i18n.Locale, languageCode string) error {
	if _, err := client.SetMyName(ctx, &botapi.SetMyNameParams{Name: locale.BotName, LanguageCode: languageCode}); err != nil {
		return fmt.Errorf("set bot name for %q: %w", languageCode, err)
	}
	time.Sleep(50 * time.Millisecond)
	if _, err := client.SetMyShortDescription(ctx, &botapi.SetMyShortDescriptionParams{
		ShortDescription: locale.ShortDescription, LanguageCode: languageCode,
	}); err != nil {
		return fmt.Errorf("set short description for %q: %w", languageCode, err)
	}
	time.Sleep(50 * time.Millisecond)
	if _, err := client.SetMyDescription(ctx, &botapi.SetMyDescriptionParams{
		Description: locale.Description, LanguageCode: languageCode,
	}); err != nil {
		return fmt.Errorf("set description for %q: %w", languageCode, err)
	}
	time.Sleep(50 * time.Millisecond)
	if _, err := client.SetMyCommands(ctx, &botapi.SetMyCommandsParams{
		Commands: commands(locale), LanguageCode: languageCode,
	}); err != nil {
		return fmt.Errorf("set commands for %q: %w", languageCode, err)
	}
	time.Sleep(50 * time.Millisecond)
	return nil
}

func commands(locale i18n.Locale) []models.BotCommand {
	order := []string{"start", "location", "today", "tomorrow", "next", "settings", "remind", "language", "privacy", "help"}
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

// Package botprofile synchronizes Telegram-facing metadata at deploy time.
package botprofile

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"math/bits"
	"mime/multipart"
	"net/http"
	"strings"
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

const profilePhotoHashTolerance = 16

type profilePhotoAPIResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
	Parameters  struct {
		RetryAfter int `json:"retry_after"`
	} `json:"parameters"`
}

// Sync applies one stable Telegram-facing identity and removes old localized
// profile variants. User-selected languages belong to chat data and must never
// mutate global bot metadata. The avatar is uploaded only when the bot does not
// yet have a profile photo.
func Sync(ctx context.Context, client *botapi.Bot, token, miniAppURL string) error {
	if err := syncStableIdentity(ctx, client); err != nil {
		return err
	}
	desiredMenuButton := miniAppMenuButton(miniAppURL)
	currentMenuButton, err := client.GetChatMenuButton(ctx, nil)
	if err != nil {
		return fmt.Errorf("get Mini App menu button: %w", err)
	}
	if !menuButtonEqual(currentMenuButton, desiredMenuButton) {
		if _, err := client.SetChatMenuButton(ctx, &botapi.SetChatMenuButtonParams{
			MenuButton: desiredMenuButton,
		}); err != nil {
			return fmt.Errorf("set Mini App menu button: %w", err)
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
	photoMatches := false
	if photos.TotalCount > 0 {
		currentPhoto, err := downloadCurrentProfilePhoto(ctx, client, token, photos)
		if err != nil {
			return err
		}
		photoMatches, err = profilePhotosEqual(currentPhoto, assets.ProfilePhoto)
		if err != nil {
			return fmt.Errorf("compare bot profile photo: %w", err)
		}
	}
	if !photoMatches {
		if err := setProfilePhoto(ctx, token, assets.ProfilePhoto); err != nil {
			return err
		}
	}
	return nil
}

// RateLimitRetryAfter reports Telegram's requested retry delay through wrapped
// errors so deploy-time profile throttling can be treated as a non-fatal skip.
func RateLimitRetryAfter(err error) (int, bool) {
	var rateLimit *botapi.TooManyRequestsError
	if !errors.As(err, &rateLimit) {
		return 0, false
	}
	return rateLimit.RetryAfter, true
}

func miniAppMenuButton(url string) models.MenuButtonWebApp {
	return models.MenuButtonWebApp{
		Type: models.MenuButtonTypeWebApp,
		Text: "🕌 Prayer App", WebApp: models.WebAppInfo{URL: url},
	}
}

func menuButtonEqual(current models.MenuButton, desired models.MenuButtonWebApp) bool {
	return current.Type == models.MenuButtonTypeWebApp &&
		current.WebApp != nil &&
		current.WebApp.Text == desired.Text &&
		current.WebApp.WebApp.URL == desired.WebApp.URL
}

type identityClient interface {
	GetMyName(context.Context, *botapi.GetMyNameParams) (models.BotName, error)
	SetMyName(context.Context, *botapi.SetMyNameParams) (bool, error)
	GetMyShortDescription(context.Context, *botapi.GetMyShortDescriptionParams) (models.BotShortDescription, error)
	SetMyShortDescription(context.Context, *botapi.SetMyShortDescriptionParams) (bool, error)
	GetMyDescription(context.Context, *botapi.GetMyDescriptionParams) (models.BotDescription, error)
	SetMyDescription(context.Context, *botapi.SetMyDescriptionParams) (bool, error)
	GetMyCommands(context.Context, *botapi.GetMyCommandsParams) ([]models.BotCommand, error)
	SetMyCommands(context.Context, *botapi.SetMyCommandsParams) (bool, error)
	DeleteMyCommands(context.Context, *botapi.DeleteMyCommandsParams) (bool, error)
}

func syncStableIdentity(ctx context.Context, client identityClient) error {
	english := i18n.Resolve("en")
	if err := syncName(ctx, client, "", english.BotName); err != nil {
		return err
	}
	if err := syncShortDescription(ctx, client, "", english.ShortDescription); err != nil {
		return err
	}
	if err := syncDescription(ctx, client, "", english.Description); err != nil {
		return err
	}
	if err := syncCommands(ctx, client, "", commands(english)); err != nil {
		return err
	}

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
	if err := syncName(ctx, client, languageCode, ""); err != nil {
		return err
	}
	if err := syncShortDescription(ctx, client, languageCode, ""); err != nil {
		return err
	}
	if err := syncDescription(ctx, client, languageCode, ""); err != nil {
		return err
	}
	return syncCommands(ctx, client, languageCode, nil)
}

func syncName(ctx context.Context, client identityClient, languageCode, desired string) error {
	current, err := client.GetMyName(ctx, &botapi.GetMyNameParams{LanguageCode: languageCode})
	if err != nil {
		return fmt.Errorf("get bot name for %q: %w", languageCode, err)
	}
	if current.Name == desired {
		return nil
	}
	if _, err := client.SetMyName(ctx, &botapi.SetMyNameParams{Name: desired, LanguageCode: languageCode}); err != nil {
		return fmt.Errorf("set bot name for %q: %w", languageCode, err)
	}
	time.Sleep(profileSyncDelay)
	return nil
}

func syncShortDescription(ctx context.Context, client identityClient, languageCode, desired string) error {
	current, err := client.GetMyShortDescription(ctx, &botapi.GetMyShortDescriptionParams{LanguageCode: languageCode})
	if err != nil {
		return fmt.Errorf("get short description for %q: %w", languageCode, err)
	}
	if current.ShortDescription == desired {
		return nil
	}
	if _, err := client.SetMyShortDescription(ctx, &botapi.SetMyShortDescriptionParams{
		ShortDescription: desired,
		LanguageCode:     languageCode,
	}); err != nil {
		return fmt.Errorf("set short description for %q: %w", languageCode, err)
	}
	time.Sleep(profileSyncDelay)
	return nil
}

func syncDescription(ctx context.Context, client identityClient, languageCode, desired string) error {
	current, err := client.GetMyDescription(ctx, &botapi.GetMyDescriptionParams{LanguageCode: languageCode})
	if err != nil {
		return fmt.Errorf("get description for %q: %w", languageCode, err)
	}
	if current.Description == desired {
		return nil
	}
	if _, err := client.SetMyDescription(ctx, &botapi.SetMyDescriptionParams{
		Description:  desired,
		LanguageCode: languageCode,
	}); err != nil {
		return fmt.Errorf("set description for %q: %w", languageCode, err)
	}
	time.Sleep(profileSyncDelay)
	return nil
}

func syncCommands(ctx context.Context, client identityClient, languageCode string, desired []models.BotCommand) error {
	current, err := client.GetMyCommands(ctx, &botapi.GetMyCommandsParams{LanguageCode: languageCode})
	if err != nil {
		return fmt.Errorf("get commands for %q: %w", languageCode, err)
	}
	if commandsEqual(current, desired) {
		return nil
	}
	if len(desired) == 0 {
		if _, err := client.DeleteMyCommands(ctx, &botapi.DeleteMyCommandsParams{LanguageCode: languageCode}); err != nil {
			return fmt.Errorf("delete commands for %q: %w", languageCode, err)
		}
	} else if _, err := client.SetMyCommands(ctx, &botapi.SetMyCommandsParams{
		Commands:     desired,
		LanguageCode: languageCode,
	}); err != nil {
		return fmt.Errorf("set commands for %q: %w", languageCode, err)
	}
	time.Sleep(profileSyncDelay)
	return nil
}

func commandsEqual(current, desired []models.BotCommand) bool {
	if len(current) != len(desired) {
		return false
	}
	for index := range current {
		if current[index] != desired[index] {
			return false
		}
	}
	return true
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

func downloadCurrentProfilePhoto(ctx context.Context, client *botapi.Bot, token string, photos *models.UserProfilePhotos) ([]byte, error) {
	if len(photos.Photos) == 0 || len(photos.Photos[0]) == 0 {
		return nil, fmt.Errorf("get bot profile photos returned no photo sizes")
	}
	largest := photos.Photos[0][0]
	for _, candidate := range photos.Photos[0][1:] {
		if candidate.Width*candidate.Height > largest.Width*largest.Height {
			largest = candidate
		}
	}
	file, err := client.GetFile(ctx, &botapi.GetFileParams{FileID: largest.FileID})
	if err != nil {
		return nil, fmt.Errorf("get bot profile photo file: %w", err)
	}
	if strings.TrimSpace(file.FilePath) == "" {
		return nil, fmt.Errorf("get bot profile photo file returned an empty path")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://api.telegram.org/file/bot"+token+"/"+strings.TrimLeft(file.FilePath, "/"), nil)
	if err != nil {
		return nil, fmt.Errorf("create bot profile photo download: %w", err)
	}
	response, err := (&http.Client{Timeout: 30 * time.Second}).Do(request)
	if err != nil {
		return nil, fmt.Errorf("download bot profile photo: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download bot profile photo returned HTTP %d", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 10<<20))
	if err != nil {
		return nil, fmt.Errorf("read bot profile photo: %w", err)
	}
	return data, nil
}

func profilePhotosEqual(current, desired []byte) (bool, error) {
	currentHash, err := profilePhotoHash(current)
	if err != nil {
		return false, fmt.Errorf("decode current photo: %w", err)
	}
	desiredHash, err := profilePhotoHash(desired)
	if err != nil {
		return false, fmt.Errorf("decode desired photo: %w", err)
	}
	return bits.OnesCount64(currentHash^desiredHash) <= profilePhotoHashTolerance, nil
}

// profilePhotoHash calculates a difference hash from normalized sample points.
// Telegram may recompress an uploaded JPEG, so comparing source bytes would
// incorrectly treat the same visible avatar as a change on every deployment.
func profilePhotoHash(data []byte) (uint64, error) {
	decoded, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return 0, err
	}
	return differenceHash(decoded), nil
}

func differenceHash(source image.Image) uint64 {
	bounds := source.Bounds()
	var hash uint64
	var bit uint
	for y := 0; y < 8; y++ {
		sampleY := bounds.Min.Y + (2*y+1)*bounds.Dy()/16
		for x := 0; x < 8; x++ {
			leftX := bounds.Min.X + (2*x+1)*bounds.Dx()/18
			rightX := bounds.Min.X + (2*x+3)*bounds.Dx()/18
			if grayscale(source.At(leftX, sampleY).RGBA()) > grayscale(source.At(rightX, sampleY).RGBA()) {
				hash |= 1 << bit
			}
			bit++
		}
	}
	return hash
}

func grayscale(red, green, blue, _ uint32) uint32 {
	return (299*red + 587*green + 114*blue) / 1000
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
	var result profilePhotoAPIResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("decode profile photo response (status %d): %w", response.StatusCode, err)
	}
	if response.StatusCode != http.StatusOK || !result.OK {
		return profilePhotoResponseError(response.StatusCode, result)
	}
	return nil
}

func profilePhotoResponseError(status int, result profilePhotoAPIResponse) error {
	if status == http.StatusTooManyRequests {
		return fmt.Errorf("set bot profile photo: %w", &botapi.TooManyRequestsError{
			Message:    result.Description,
			RetryAfter: result.Parameters.RetryAfter,
		})
	}
	return fmt.Errorf("set bot profile photo failed (status %d): %s", status, result.Description)
}

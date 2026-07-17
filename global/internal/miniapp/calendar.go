package miniapp

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/escalopa/prayer-bot/global/internal/calendarfile"
	"github.com/escalopa/prayer-bot/global/internal/i18n"
	"github.com/escalopa/prayer-bot/global/internal/store"
)

const calendarTokenLifetime = 5 * time.Minute

type calendarLinkRequest struct {
	Days int `json:"days"`
}

type calendarLinkResponse struct {
	Path     string `json:"path"`
	Filename string `json:"filename"`
}

type calendarTokenPayload struct {
	ChatID    int64 `json:"chat_id"`
	Days      int   `json:"days"`
	ExpiresAt int64 `json:"expires_at"`
}

func (h *Handler) calendarLink(w http.ResponseWriter, r *http.Request, identity Identity) error {
	var request calendarLinkRequest
	if err := decodeJSON(w, r, &request); err != nil || !validCalendarDays(request.Days) {
		return badRequest("invalid_calendar_range")
	}
	profile, err := h.store.Profile(r.Context(), identity.UserID)
	if store.IsNotFound(err) {
		return conflict("location_required")
	}
	if err != nil {
		return fmt.Errorf("load profile: %w", err)
	}
	location, err := time.LoadLocation(profile.Timezone)
	if err != nil {
		return fmt.Errorf("load profile timezone: %w", err)
	}
	token, err := signCalendarToken(h.botToken, calendarTokenPayload{
		ChatID: identity.UserID, Days: request.Days,
		ExpiresAt: h.now().Add(calendarTokenLifetime).Unix(),
	})
	if err != nil {
		return fmt.Errorf("sign calendar token: %w", err)
	}
	return writeJSON(w, calendarLinkResponse{
		Path:     "/api/miniapp/calendar.ics?token=" + url.QueryEscape(token),
		Filename: calendarfile.Filename(h.now().In(location), request.Days),
	})
}

func (h *Handler) calendarDownload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	payload, err := verifyCalendarToken(h.botToken, r.URL.Query().Get("token"), h.now())
	if err != nil {
		http.Error(w, "invalid or expired calendar link", http.StatusUnauthorized)
		return
	}
	chat, err := h.store.Chat(r.Context(), payload.ChatID)
	if err != nil {
		if !store.IsNotFound(err) {
			h.logger.Error("Calendar export chat lookup failed", "error", err)
		}
		http.NotFound(w, r)
		return
	}
	profile, err := h.store.Profile(r.Context(), payload.ChatID)
	if err != nil {
		if !store.IsNotFound(err) {
			h.logger.Error("Calendar export profile lookup failed", "error", err)
		}
		http.NotFound(w, r)
		return
	}
	location, err := time.LoadLocation(profile.Timezone)
	if err != nil {
		h.logger.Error("Calendar export timezone lookup failed", "timezone", profile.Timezone, "error", err)
		http.Error(w, "calendar generation failed", http.StatusInternalServerError)
		return
	}
	start := h.now().In(location)
	data, err := calendarfile.Generate(
		r.Context(), h.calculator, profile, i18n.Resolve(chat.LanguageCode), start, payload.Days, h.now(),
	)
	if err != nil {
		h.logger.Error("Calendar export generation failed", "days", payload.Days, "error", err)
		http.Error(w, "calendar generation failed", http.StatusInternalServerError)
		return
	}
	filename := calendarfile.Filename(start, payload.Days)
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("Access-Control-Allow-Origin", "https://web.telegram.org")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func signCalendarToken(secret string, payload calendarTokenPayload) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	aead, err := calendarTokenCipher(secret)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := aead.Seal(nonce, nonce, data, []byte("global-prayer-calendar-v1"))
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

func verifyCalendarToken(secret, token string, now time.Time) (calendarTokenPayload, error) {
	sealed, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return calendarTokenPayload{}, fmt.Errorf("decode token")
	}
	aead, err := calendarTokenCipher(secret)
	if err != nil || len(sealed) <= aead.NonceSize() {
		return calendarTokenPayload{}, fmt.Errorf("malformed token")
	}
	nonce, ciphertext := sealed[:aead.NonceSize()], sealed[aead.NonceSize():]
	data, err := aead.Open(nil, nonce, ciphertext, []byte("global-prayer-calendar-v1"))
	if err != nil {
		return calendarTokenPayload{}, fmt.Errorf("invalid token")
	}
	var payload calendarTokenPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return calendarTokenPayload{}, fmt.Errorf("decode payload")
	}
	expiresAt := time.Unix(payload.ExpiresAt, 0)
	if payload.ChatID <= 0 || !validCalendarDays(payload.Days) || !expiresAt.After(now) ||
		expiresAt.After(now.Add(2*calendarTokenLifetime)) {
		return calendarTokenPayload{}, fmt.Errorf("invalid payload")
	}
	return payload, nil
}

func calendarTokenCipher(secret string) (cipher.AEAD, error) {
	key := sha256.Sum256([]byte("global-prayer-calendar-key-v1\x00" + secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func validCalendarDays(days int) bool { return days == 7 || days == 30 }

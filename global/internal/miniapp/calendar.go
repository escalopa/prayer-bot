package miniapp

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/escalopa/prayer-bot/global/internal/calendarfile"
	"github.com/escalopa/prayer-bot/global/internal/i18n"
	"github.com/escalopa/prayer-bot/global/internal/store"
)

const rollingCalendarDays = 30

type calendarSubscriptionResponse struct {
	Enabled bool   `json:"enabled"`
	Path    string `json:"path,omitempty"`
}

func (h *Handler) createCalendarSubscription(w http.ResponseWriter, r *http.Request, identity Identity) error {
	if _, err := h.store.Profile(r.Context(), identity.UserID); store.IsNotFound(err) {
		return conflict("location_required")
	} else if err != nil {
		return fmt.Errorf("load profile: %w", err)
	}
	feedToken, uidNamespace, err := newCalendarCredentials()
	if err != nil {
		return fmt.Errorf("create calendar credentials: %w", err)
	}
	subscription, err := h.store.EnableCalendarSubscription(
		r.Context(), identity.UserID, feedToken, uidNamespace,
	)
	if err != nil {
		return fmt.Errorf("enable calendar subscription: %w", err)
	}
	return writeJSON(w, calendarSubscriptionResponse{
		Enabled: true,
		Path:    "/api/miniapp/calendar.ics?token=" + url.QueryEscape(subscription.FeedToken),
	})
}

func (h *Handler) disableCalendarSubscription(w http.ResponseWriter, r *http.Request, identity Identity) error {
	if err := h.store.DisableCalendarSubscription(r.Context(), identity.UserID); err != nil {
		return fmt.Errorf("disable calendar subscription: %w", err)
	}
	return writeJSON(w, calendarSubscriptionResponse{Enabled: false})
}

func (h *Handler) calendarDownload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, max-age=300")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	subscription, err := h.store.CalendarSubscriptionByToken(r.Context(), r.URL.Query().Get("token"))
	if err != nil || !subscription.Enabled {
		if err != nil && !store.IsNotFound(err) {
			h.logger.Error("Calendar subscription lookup failed", "error", err)
		}
		http.Error(w, "invalid calendar subscription", http.StatusUnauthorized)
		return
	}
	chat, err := h.store.Chat(r.Context(), subscription.ChatID)
	if err != nil {
		if !store.IsNotFound(err) {
			h.logger.Error("Calendar feed chat lookup failed", "error", err)
		}
		http.NotFound(w, r)
		return
	}
	profile, err := h.store.Profile(r.Context(), subscription.ChatID)
	if err != nil {
		if !store.IsNotFound(err) {
			h.logger.Error("Calendar feed profile lookup failed", "error", err)
		}
		http.NotFound(w, r)
		return
	}
	location, err := time.LoadLocation(profile.Timezone)
	if err != nil {
		h.logger.Error("Calendar feed timezone lookup failed", "timezone", profile.Timezone, "error", err)
		http.Error(w, "calendar generation failed", http.StatusInternalServerError)
		return
	}
	start := h.now().In(location)
	createdAt := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, location)
	data, err := calendarfile.Generate(
		r.Context(),
		h.calculator,
		profile,
		i18n.Resolve(chat.LanguageCode),
		start,
		rollingCalendarDays,
		createdAt,
		subscription.UIDNamespace,
	)
	if err != nil {
		h.logger.Error("Calendar feed generation failed", "error", err)
		http.Error(w, "calendar generation failed", http.StatusInternalServerError)
		return
	}
	digest := sha256.Sum256(data)
	etag := `"` + hex.EncodeToString(digest[:]) + `"`
	w.Header().Set("ETag", etag)
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", `inline; filename="prayer-times.ics"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func newCalendarCredentials() (string, string, error) {
	feedToken := make([]byte, 32)
	if _, err := rand.Read(feedToken); err != nil {
		return "", "", err
	}
	uidNamespace := make([]byte, 16)
	if _, err := rand.Read(uidNamespace); err != nil {
		return "", "", err
	}
	return hex.EncodeToString(feedToken), hex.EncodeToString(uidNamespace), nil
}

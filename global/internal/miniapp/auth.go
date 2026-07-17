package miniapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

var ErrInvalidInitData = errors.New("invalid Telegram Mini App init data")

type Identity struct {
	UserID       int64
	FirstName    string
	LanguageCode string
}

type initDataUser struct {
	ID           int64  `json:"id"`
	IsBot        bool   `json:"is_bot"`
	FirstName    string `json:"first_name"`
	LanguageCode string `json:"language_code"`
}

// ValidateInitData implements Telegram's HMAC validation for WebApp.initData
// and rejects stale sessions. The raw query string must come from Telegram.WebApp.initData.
func ValidateInitData(raw, botToken string, now time.Time, maxAge time.Duration) (Identity, error) {
	if raw == "" || botToken == "" || len(raw) > 16<<10 {
		return Identity{}, ErrInvalidInitData
	}
	values, err := url.ParseQuery(raw)
	if err != nil {
		return Identity{}, ErrInvalidInitData
	}
	for _, value := range values {
		if len(value) != 1 {
			return Identity{}, ErrInvalidInitData
		}
	}

	providedHex := values.Get("hash")
	provided, err := hex.DecodeString(providedHex)
	if err != nil || len(provided) != sha256.Size {
		return Identity{}, ErrInvalidInitData
	}
	delete(values, "hash")

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+values.Get(key))
	}
	secretMAC := hmac.New(sha256.New, []byte("WebAppData"))
	_, _ = secretMAC.Write([]byte(botToken))
	signatureMAC := hmac.New(sha256.New, secretMAC.Sum(nil))
	_, _ = signatureMAC.Write([]byte(strings.Join(parts, "\n")))
	if !hmac.Equal(provided, signatureMAC.Sum(nil)) {
		return Identity{}, ErrInvalidInitData
	}

	authUnix, err := strconv.ParseInt(values.Get("auth_date"), 10, 64)
	if err != nil || authUnix <= 0 {
		return Identity{}, ErrInvalidInitData
	}
	authTime := time.Unix(authUnix, 0)
	if authTime.After(now.Add(5*time.Minute)) || now.Sub(authTime) > maxAge {
		return Identity{}, fmt.Errorf("%w: expired", ErrInvalidInitData)
	}

	var user initDataUser
	if err := json.Unmarshal([]byte(values.Get("user")), &user); err != nil || user.ID <= 0 || user.IsBot {
		return Identity{}, ErrInvalidInitData
	}
	return Identity{UserID: user.ID, FirstName: user.FirstName, LanguageCode: user.LanguageCode}, nil
}

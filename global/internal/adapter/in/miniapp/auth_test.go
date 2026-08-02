package miniapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestValidateInitDataAcceptsTelegramSignature(t *testing.T) {
	now := time.Date(2026, time.July, 17, 12, 0, 0, 0, time.UTC)
	raw := signedInitData(t, "test-token", now, initDataUser{ID: 1385434843, FirstName: "Ahmed", LanguageCode: "ar"})

	identity, err := ValidateInitData(raw, "test-token", now.Add(time.Minute), 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if identity.UserID != 1385434843 || identity.FirstName != "Ahmed" || identity.LanguageCode != "ar" {
		t.Fatalf("unexpected identity: %+v", identity)
	}
}

func TestValidateInitDataRejectsTamperingAndExpiry(t *testing.T) {
	now := time.Date(2026, time.July, 17, 12, 0, 0, 0, time.UTC)
	raw := signedInitData(t, "test-token", now, initDataUser{ID: 42, FirstName: "A"})
	if _, err := ValidateInitData(strings.Replace(raw, "first_name%22%3A%22A", "first_name%22%3A%22B", 1), "test-token", now, time.Hour); err == nil {
		t.Fatal("expected tampered init data to fail")
	}
	if _, err := ValidateInitData(raw, "test-token", now.Add(2*time.Hour), time.Hour); err == nil {
		t.Fatal("expected expired init data to fail")
	}
}

func signedInitData(t *testing.T, token string, authTime time.Time, user initDataUser) string {
	t.Helper()
	encodedUser, err := json.Marshal(user)
	if err != nil {
		t.Fatal(err)
	}
	values := url.Values{
		"auth_date": {strconv.FormatInt(authTime.Unix(), 10)},
		"query_id":  {"AAE-test"},
		"user":      {string(encodedUser)},
	}
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
	_, _ = secretMAC.Write([]byte(token))
	signatureMAC := hmac.New(sha256.New, secretMAC.Sum(nil))
	_, _ = signatureMAC.Write([]byte(strings.Join(parts, "\n")))
	values.Set("hash", hex.EncodeToString(signatureMAC.Sum(nil)))
	return values.Encode()
}

package botprofile

import (
	"bytes"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Telegram rejects multipart/form-data requests that carry no fields with a
// bodyless HTTP 400. The bot library encodes every call as multipart and
// sends parameterless methods (getMyName, getMe, getChatMenuButton, ...) with
// a multipart Content-Type but a completely empty body, so those calls always
// fail. Rewriting the empty envelope into an empty JSON object keeps them
// valid; requests that carry any field pass through untouched.
type emptyMultipartFix struct {
	base http.RoundTripper
}

// An empty multipart body is either no bytes at all or just the closing
// boundary delimiter.
var closingBoundaryOnly = regexp.MustCompile(`\A\s*--[^\r\n]+--\s*\z`)

func (t emptyMultipartFix) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.ContentLength >= 0 && request.ContentLength <= 128 &&
		strings.HasPrefix(request.Header.Get("Content-Type"), "multipart/form-data") {
		var body []byte
		if request.Body != nil {
			read, err := io.ReadAll(request.Body)
			_ = request.Body.Close()
			if err != nil {
				return nil, err
			}
			body = read
		}
		if len(body) == 0 || closingBoundaryOnly.Match(body) {
			request.Header.Set("Content-Type", "application/json")
			body = []byte("{}")
		}
		request.Body = io.NopCloser(bytes.NewReader(body))
		request.ContentLength = int64(len(body))
	}
	return t.base.RoundTrip(request)
}

// HTTPClient returns the HTTP client every profile-sync Telegram call must go
// through. See emptyMultipartFix for why a plain client is not enough.
func HTTPClient() *http.Client {
	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: emptyMultipartFix{base: http.DefaultTransport},
	}
}

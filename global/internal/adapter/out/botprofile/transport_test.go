package botprofile

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEmptyMultipartIsRewrittenToJSON(t *testing.T) {
	var gotContentType, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotContentType, gotBody = r.Header.Get("Content-Type"), string(body)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	empty := &bytes.Buffer{}
	writer := multipart.NewWriter(empty)
	_ = writer.Close()
	request, _ := http.NewRequest(http.MethodPost, server.URL, bytes.NewReader(empty.Bytes()))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if _, err := HTTPClient().Do(request); err != nil {
		t.Fatal(err)
	}
	if gotContentType != "application/json" || gotBody != "{}" {
		t.Fatalf("empty multipart was not rewritten: content-type=%q body=%q", gotContentType, gotBody)
	}
}

func TestBodylessMultipartIsRewrittenToJSON(t *testing.T) {
	var gotContentType, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotContentType, gotBody = r.Header.Get("Content-Type"), string(body)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	// The bot library sends parameterless methods as a multipart Content-Type
	// with a completely empty body.
	request, _ := http.NewRequest(http.MethodPost, server.URL, bytes.NewReader(nil))
	request.Header.Set("Content-Type", "multipart/form-data; boundary=deadbeef")
	if _, err := HTTPClient().Do(request); err != nil {
		t.Fatal(err)
	}
	if gotContentType != "application/json" || gotBody != "{}" {
		t.Fatalf("bodyless multipart was not rewritten: content-type=%q body=%q", gotContentType, gotBody)
	}
}

func TestNonEmptyMultipartPassesThrough(t *testing.T) {
	var gotContentType, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotContentType, gotBody = r.Header.Get("Content-Type"), string(body)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	filled := &bytes.Buffer{}
	writer := multipart.NewWriter(filled)
	_ = writer.WriteField("language_code", "en")
	_ = writer.Close()
	request, _ := http.NewRequest(http.MethodPost, server.URL, bytes.NewReader(filled.Bytes()))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if _, err := HTTPClient().Do(request); err != nil {
		t.Fatal(err)
	}
	if gotContentType == "application/json" || gotBody != filled.String() {
		t.Fatalf("multipart with fields must pass through unchanged: content-type=%q", gotContentType)
	}
}

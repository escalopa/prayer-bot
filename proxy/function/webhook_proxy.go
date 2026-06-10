package function

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/escalopa/prayer-bot/log"
)

const upstreamTimeout = 15 * time.Second

func WebhookProxy(w http.ResponseWriter, r *http.Request) {
	startedAt := time.Now()

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		log.Warn("reject webhook request",
			log.String("method", r.Method),
			log.String("latency", time.Since(startedAt).String()),
		)
		return
	}

	upstreamURL := os.Getenv("YC_DISPATCHER_URL")
	if upstreamURL == "" {
		http.Error(w, "YC_DISPATCHER_URL is not set", http.StatusInternalServerError)
		log.Error("missing upstream dispatcher url")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read request body", http.StatusInternalServerError)
		log.Error("read webhook body", log.Err(err))
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, upstreamURL, bytes.NewReader(body))
	if err != nil {
		http.Error(w, "create upstream request", http.StatusInternalServerError)
		log.Error("create upstream request", log.Err(err))
		return
	}

	req.Header = r.Header.Clone()
	req.ContentLength = int64(len(body))

	client := &http.Client{Timeout: upstreamTimeout}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "forward webhook request", http.StatusBadGateway)
		log.Error("forward webhook request", log.Err(err))
		return
	}
	defer resp.Body.Close()

	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Error("copy upstream response", log.Err(err))
	}

	log.Info("proxied webhook",
		log.String("method", r.Method),
		log.String("upstream_status", strconv.Itoa(resp.StatusCode)),
		log.String("latency", time.Since(startedAt).String()),
		log.String("content_length", strconv.Itoa(len(body))),
	)
}

func copyHeaders(dst http.Header, src http.Header) {
	for key, values := range src {
		dst.Del(key)
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

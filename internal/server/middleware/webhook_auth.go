package middleware

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
)

type contextKey string

const rawBodyKey contextKey = "rawBody"

// WebhookAuth returns a middleware that verifies HMAC-SHA256 signatures.
// secret is the shared secret, headerName is the header carrying the signature,
// and sigPrefix is an optional prefix to strip (e.g. "sha256=").
func WebhookAuth(secret, headerName, sigPrefix string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB

			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "request body too large or unreadable", http.StatusBadRequest)
				return
			}

			sigHeader := r.Header.Get(headerName)
			if sigHeader == "" {
				http.Error(w, "missing signature header", http.StatusUnauthorized)
				return
			}

			sig := strings.TrimPrefix(sigHeader, sigPrefix)
			sigBytes, err := hex.DecodeString(sig)
			if err != nil {
				http.Error(w, "invalid signature encoding", http.StatusUnauthorized)
				return
			}

			mac := hmac.New(sha256.New, []byte(secret))
			mac.Write(body)
			expected := mac.Sum(nil)

			if !hmac.Equal(expected, sigBytes) {
				http.Error(w, "signature mismatch", http.StatusUnauthorized)
				return
			}

			// Re-inject body so handlers can read it again.
			r.Body = io.NopCloser(bytes.NewReader(body))

			ctx := context.WithValue(r.Context(), rawBodyKey, body)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RawBodyFromContext retrieves the pre-read request body stored by WebhookAuth.
func RawBodyFromContext(ctx context.Context) ([]byte, bool) {
	b, ok := ctx.Value(rawBodyKey).([]byte)
	return b, ok
}

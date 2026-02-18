package custom

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andrewserra/kitchen/internal/server/middleware"
)

func TestExtractEventType(t *testing.T) {
	tests := []struct {
		name     string
		body     []byte
		expected string
	}{
		{
			name:     "valid event_type field",
			body:     []byte(`{"event_type":"order.created","data":{}}`),
			expected: "order.created",
		},
		{
			name:     "missing event_type field",
			body:     []byte(`{"data":{}}`),
			expected: "unknown",
		},
		{
			name:     "invalid JSON",
			body:     []byte(`not json`),
			expected: "unknown",
		},
		{
			name:     "empty body",
			body:     []byte(``),
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractEventType(tt.body)
			if got != tt.expected {
				t.Errorf("extractEventType() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestWebhookAuthMiddleware(t *testing.T) {
	const secret = "test-secret"
	const header = "X-Webhook-Signature"
	const prefix = "sha256="

	sign := func(body []byte) string {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		return prefix + hex.EncodeToString(mac.Sum(nil))
	}

	mw := middleware.WebhookAuth(secret, header, prefix)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := middleware.RawBodyFromContext(r.Context())
		if !ok {
			http.Error(w, "no body in context", http.StatusInternalServerError)
			return
		}
		w.Write(body)
	}))

	t.Run("valid signature passes", func(t *testing.T) {
		body := []byte(`{"event_type":"test"}`)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		req = req.WithContext(context.Background())
		req.Header.Set(header, sign(body))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("missing signature returns 401", func(t *testing.T) {
		body := []byte(`{"event_type":"test"}`)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("wrong signature returns 401", func(t *testing.T) {
		body := []byte(`{"event_type":"test"}`)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		req.Header.Set(header, prefix+hex.EncodeToString([]byte("badsig")))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})
}

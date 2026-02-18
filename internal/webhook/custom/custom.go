package custom

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/andrewserra/kitchen/internal/webhook"
)

// Handler handles incoming webhook events from a generic custom source.
type Handler struct {
	DB *sql.DB
}

func New(db *sql.DB) *Handler {
	return &Handler{DB: db}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	body, ok := webhook.RawBodyFromContext(r.Context())
	if !ok {
		http.Error(w, "body unavailable", http.StatusInternalServerError)
		return
	}

	eventType := extractEventType(body)

	headers := make(map[string]string, len(r.Header))
	for k, vals := range r.Header {
		if len(vals) > 0 {
			headers[k] = vals[0]
		}
	}

	sig := r.Header.Get("X-Webhook-Signature")
	requestID := r.Header.Get("X-Request-Id")

	if err := webhook.SaveEvent(r.Context(), h.DB, "custom", eventType, body, headers, sig, requestID); err != nil {
		http.Error(w, "failed to save event", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// extractEventType attempts to pull an "event_type" field from a JSON payload.
// Falls back to "unknown" if the field is absent or the body is not valid JSON.
func extractEventType(body []byte) string {
	var payload struct {
		EventType string `json:"event_type"`
	}
	if err := json.Unmarshal(body, &payload); err == nil && payload.EventType != "" {
		return payload.EventType
	}
	return "unknown"
}

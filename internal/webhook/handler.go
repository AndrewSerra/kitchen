package webhook

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/andrewserra/kitchen/internal/db"
	"github.com/andrewserra/kitchen/internal/server/middleware"
)

// RawBodyFromContext retrieves the pre-read request body stored by WebhookAuth middleware.
func RawBodyFromContext(ctx context.Context) ([]byte, bool) {
	return middleware.RawBodyFromContext(ctx)
}

// SaveEvent persists a webhook event to the database.
func SaveEvent(ctx context.Context, sqlDB *sql.DB, source, eventType string, body []byte, headers map[string]string, sig, requestID string) error {
	_, err := db.InsertWebhookEvent(ctx, sqlDB, db.WebhookEvent{
		Source:    source,
		EventType: eventType,
		Payload:   body,
		Headers:   headers,
		Signature: sig,
		RequestID: requestID,
	})
	if err != nil {
		return fmt.Errorf("save event: %w", err)
	}
	return nil
}

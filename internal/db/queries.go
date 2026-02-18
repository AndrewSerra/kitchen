package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

type WebhookEvent struct {
	Source    string
	EventType string
	Payload   []byte
	Headers   map[string]string
	Signature string
	RequestID string
}

func InsertWebhookEvent(ctx context.Context, db *sql.DB, e WebhookEvent) (int64, error) {
	headersJSON, err := json.Marshal(e.Headers)
	if err != nil {
		return 0, fmt.Errorf("marshal headers: %w", err)
	}

	var id int64
	err = db.QueryRowContext(ctx, `
		INSERT INTO webhook_events (source, event_type, payload, headers, signature, request_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		e.Source,
		e.EventType,
		e.Payload,
		headersJSON,
		e.Signature,
		e.RequestID,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert webhook event: %w", err)
	}

	return id, nil
}

func ListWebhookEvents(ctx context.Context, db *sql.DB, source string, limit int) ([]WebhookEvent, error) {
	query := `
		SELECT source, event_type, payload, headers, signature, request_id
		FROM webhook_events
		WHERE ($1 = '' OR source = $1)
		ORDER BY received_at DESC
		LIMIT $2`

	rows, err := db.QueryContext(ctx, query, source, limit)
	if err != nil {
		return nil, fmt.Errorf("list webhook events: %w", err)
	}
	defer rows.Close()

	var events []WebhookEvent
	for rows.Next() {
		var e WebhookEvent
		var headersJSON []byte
		if err := rows.Scan(&e.Source, &e.EventType, &e.Payload, &headersJSON, &e.Signature, &e.RequestID); err != nil {
			return nil, fmt.Errorf("scan webhook event: %w", err)
		}
		if err := json.Unmarshal(headersJSON, &e.Headers); err != nil {
			return nil, fmt.Errorf("unmarshal headers: %w", err)
		}
		events = append(events, e)
	}

	return events, rows.Err()
}

CREATE TYPE webhook_status AS ENUM ('received', 'processing', 'processed', 'failed');

CREATE TABLE webhook_events (
    id           BIGSERIAL PRIMARY KEY,
    source       TEXT           NOT NULL,
    event_type   TEXT           NOT NULL,
    payload      JSONB          NOT NULL,
    headers      JSONB          NOT NULL DEFAULT '{}',
    signature    TEXT           NOT NULL DEFAULT '',
    request_id   TEXT           NOT NULL DEFAULT '',
    received_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    status       webhook_status NOT NULL DEFAULT 'received'
);

CREATE INDEX idx_webhook_events_source_event ON webhook_events (source, event_type);
CREATE INDEX idx_webhook_events_status ON webhook_events (status) WHERE status != 'processed';
CREATE INDEX idx_webhook_events_received_at ON webhook_events (received_at DESC);

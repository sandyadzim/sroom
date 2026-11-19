BEGIN;
ALTER TABLE booking.outbox_events
    ADD COLUMN published_at TIMESTAMPTZ,
    ADD COLUMN publish_attempts INTEGER NOT NULL DEFAULT 0 CHECK (publish_attempts >= 0),
    ADD COLUMN last_error_code TEXT;

CREATE INDEX outbox_confirmed_pending_idx ON booking.outbox_events (occurred_at, id)
    WHERE published_at IS NULL AND event_type = 'booking.confirmed.v1';
COMMIT;

BEGIN;

ALTER TABLE booking.bookings
    ADD COLUMN confirmed_payment_id UUID UNIQUE,
    ADD COLUMN confirmed_provider_order_id VARCHAR(50) UNIQUE;

CREATE TABLE booking.inbox_events (
    consumer_name TEXT NOT NULL,
    event_id UUID NOT NULL,
    event_type TEXT NOT NULL,
    aggregate_id UUID NOT NULL,
    booking_id UUID NOT NULL,
    payload_hash TEXT NOT NULL CHECK (length(payload_hash) = 64),
    processed_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (consumer_name, event_id)
);

CREATE INDEX inbox_events_booking_idx ON booking.inbox_events (booking_id);

COMMIT;

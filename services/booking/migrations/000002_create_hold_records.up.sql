BEGIN;

CREATE TABLE booking.idempotency_records (
    customer_id UUID NOT NULL,
    idempotency_key UUID NOT NULL,
    request_hash TEXT NOT NULL,
    booking_id UUID UNIQUE REFERENCES booking.bookings(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (customer_id, idempotency_key)
);

CREATE TABLE booking.outbox_events (
    id UUID PRIMARY KEY,
    aggregate_id UUID NOT NULL REFERENCES booking.bookings(id),
    event_type TEXT NOT NULL,
    envelope JSONB NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    UNIQUE (aggregate_id, event_type)
);

COMMIT;

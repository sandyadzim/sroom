BEGIN;

-- Existing attempts stay nullable; only new attempts capture the verified profile.
ALTER TABLE payment.payment_attempts
    ADD COLUMN customer_full_name TEXT,
    ADD COLUMN customer_email TEXT;

CREATE TABLE payment.invoices (
    id UUID PRIMARY KEY,
    invoice_number TEXT NOT NULL UNIQUE,
    booking_id UUID NOT NULL UNIQUE,
    payment_id UUID NOT NULL UNIQUE REFERENCES payment.payment_attempts(id),
    customer_id UUID NOT NULL,
    provider_order_id TEXT NOT NULL,
    billed_to_name TEXT NOT NULL,
    billed_to_email TEXT NOT NULL CHECK (btrim(billed_to_email) <> ''),
    variant_id UUID NOT NULL,
    vehicle_label TEXT NOT NULL,
    pickup_at TIMESTAMPTZ NOT NULL,
    return_at TIMESTAMPTZ NOT NULL CHECK (return_at > pickup_at),
    rental_days INTEGER NOT NULL CHECK (rental_days > 0),
    daily_rate_idr BIGINT NOT NULL CHECK (daily_rate_idr > 0),
    total_idr BIGINT NOT NULL CHECK (total_idr = daily_rate_idr * rental_days),
    currency CHAR(3) NOT NULL CHECK (currency = 'IDR'),
    bank TEXT NOT NULL CHECK (bank = 'bni'),
    confirmed_at TIMESTAMPTZ NOT NULL,
    issued_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE payment.inbox_events (
    consumer_name TEXT NOT NULL,
    event_id UUID NOT NULL,
    event_type TEXT NOT NULL,
    aggregate_id UUID NOT NULL,
    payload_hash TEXT NOT NULL CHECK (length(payload_hash) = 64),
    processed_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (consumer_name, event_id)
);

COMMIT;

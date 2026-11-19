BEGIN;

CREATE SCHEMA IF NOT EXISTS payment;

CREATE TABLE payment.payment_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id UUID NOT NULL UNIQUE,
    customer_id UUID NOT NULL,
    provider TEXT NOT NULL DEFAULT 'midtrans' CHECK (provider = 'midtrans'),
    provider_order_id VARCHAR(50) NOT NULL UNIQUE CHECK (btrim(provider_order_id) <> ''),
    provider_transaction_id TEXT UNIQUE,
    amount_idr BIGINT NOT NULL CHECK (amount_idr > 0),
    currency CHAR(3) NOT NULL DEFAULT 'IDR' CHECK (currency = 'IDR'),
    bank TEXT NOT NULL DEFAULT 'bni' CHECK (bank = 'bni'),
    va_number TEXT,
    status TEXT NOT NULL DEFAULT 'initiating'
        CHECK (status IN ('initiating', 'pending', 'paid', 'expired', 'failed')),
    provider_status TEXT,
    hold_expires_at TIMESTAMPTZ NOT NULL,
    provider_expires_at TIMESTAMPTZ,
    provider_paid_at TIMESTAMPTZ,
    verified_at TIMESTAMPTZ,
    review_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (status <> 'paid' OR verified_at IS NOT NULL)
);

CREATE INDEX payment_attempts_created_idx
    ON payment.payment_attempts (created_at DESC, id DESC);

CREATE INDEX payment_attempts_customer_created_idx
    ON payment.payment_attempts (customer_id, created_at DESC, id DESC);

CREATE INDEX payment_attempts_unresolved_idx
    ON payment.payment_attempts (updated_at, id)
    WHERE status IN ('initiating', 'pending') OR review_reason IS NOT NULL;

CREATE TABLE payment.idempotency_records (
    customer_id UUID NOT NULL,
    idempotency_key UUID NOT NULL,
    request_hash TEXT NOT NULL CHECK (length(request_hash) = 64),
    payment_id UUID NOT NULL REFERENCES payment.payment_attempts(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (customer_id, idempotency_key)
);

CREATE INDEX idempotency_records_payment_idx
    ON payment.idempotency_records (payment_id);

CREATE TABLE payment.provider_observations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id UUID NOT NULL REFERENCES payment.payment_attempts(id),
    source TEXT NOT NULL CHECK (source IN ('charge', 'webhook', 'status_lookup')),
    provider_transaction_id TEXT,
    provider_status TEXT,
    amount_idr BIGINT,
    currency TEXT,
    provider_paid_at TIMESTAMPTZ,
    verification_result TEXT NOT NULL
        CHECK (verification_result IN ('verified', 'rejected', 'unavailable')),
    payload_digest TEXT CHECK (length(payload_digest) = 64),
    error_code TEXT,
    request_id TEXT,
    observed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX provider_observations_payment_idx
    ON payment.provider_observations (payment_id, observed_at, id);

CREATE TABLE payment.webhook_receipts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id UUID REFERENCES payment.payment_attempts(id),
    provider_order_id TEXT,
    payload_hash TEXT NOT NULL UNIQUE CHECK (length(payload_hash) = 64),
    status TEXT NOT NULL DEFAULT 'received'
        CHECK (status IN ('received', 'processed', 'rejected', 'unmatched')),
    verified_at TIMESTAMPTZ,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    error_code TEXT
);

CREATE INDEX webhook_receipts_payment_idx
    ON payment.webhook_receipts (payment_id, received_at, id);

CREATE TABLE payment.outbox_events (
    id UUID PRIMARY KEY,
    aggregate_id UUID NOT NULL REFERENCES payment.payment_attempts(id),
    event_type TEXT NOT NULL,
    envelope JSONB NOT NULL CHECK (jsonb_typeof(envelope) = 'object'),
    occurred_at TIMESTAMPTZ NOT NULL,
    published_at TIMESTAMPTZ,
    publish_attempts INTEGER NOT NULL DEFAULT 0 CHECK (publish_attempts >= 0),
    last_error_code TEXT,
    UNIQUE (aggregate_id, event_type)
);

CREATE INDEX outbox_events_unpublished_idx
    ON payment.outbox_events (occurred_at, id)
    WHERE published_at IS NULL;

COMMIT;

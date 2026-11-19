BEGIN;

CREATE TABLE payment.email_deliveries (
    id UUID PRIMARY KEY,
    invoice_id UUID NOT NULL REFERENCES payment.invoices(id),
    booking_id UUID NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('booking_confirmation', 'invoice')),
    recipient TEXT NOT NULL CHECK (btrim(recipient) <> ''),
    status TEXT NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'sending', 'sent', 'failed')),
    attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    next_attempt_at TIMESTAMPTZ DEFAULT NOW(),
    locked_until TIMESTAMPTZ,
    message_id TEXT NOT NULL UNIQUE,
    sent_at TIMESTAMPTZ,
    last_error_code TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (booking_id, kind)
);

CREATE INDEX email_deliveries_queued_idx ON payment.email_deliveries (next_attempt_at, id)
    WHERE status = 'queued';

-- Existing invoices already contain the approved recipient snapshot.
WITH jobs AS (
    SELECT gen_random_uuid() AS id, i.id AS invoice_id, i.booking_id,
           kinds.kind, i.billed_to_email AS recipient
    FROM payment.invoices i
    CROSS JOIN (VALUES ('booking_confirmation'), ('invoice')) AS kinds(kind)
)
INSERT INTO payment.email_deliveries (id, invoice_id, booking_id, kind, recipient, message_id)
SELECT id, invoice_id, booking_id, kind, recipient, '<' || id::text || '@sroom.test>' FROM jobs
ON CONFLICT (booking_id, kind) DO NOTHING;

COMMIT;

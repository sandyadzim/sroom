BEGIN;

-- Granted only after an authenticated provider lookup verifies expiry after the hold deadline.
ALTER TABLE payment.payment_attempts ADD COLUMN release_authorized_at TIMESTAMPTZ;
ALTER TABLE payment.payment_attempts ADD CONSTRAINT release_authorized_after_deadline
    CHECK (release_authorized_at IS NULL OR release_authorized_at >= hold_expires_at);

COMMIT;

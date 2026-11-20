BEGIN;

ALTER TABLE payment.payment_attempts DROP COLUMN release_authorized_at;

COMMIT;

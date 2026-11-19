BEGIN;

DROP TABLE payment.outbox_events;
DROP TABLE payment.webhook_receipts;
DROP TABLE payment.provider_observations;
DROP TABLE payment.idempotency_records;
DROP TABLE payment.payment_attempts;

COMMIT;

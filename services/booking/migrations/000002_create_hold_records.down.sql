BEGIN;

DROP TABLE booking.outbox_events;
DROP TABLE booking.idempotency_records;

COMMIT;

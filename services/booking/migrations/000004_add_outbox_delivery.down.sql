BEGIN;
DROP INDEX booking.outbox_confirmed_pending_idx;
ALTER TABLE booking.outbox_events
    DROP COLUMN last_error_code, DROP COLUMN publish_attempts, DROP COLUMN published_at;
COMMIT;

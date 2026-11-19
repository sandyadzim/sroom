BEGIN;
DROP TABLE booking.inbox_events;
ALTER TABLE booking.bookings
    DROP COLUMN confirmed_provider_order_id,
    DROP COLUMN confirmed_payment_id;
COMMIT;

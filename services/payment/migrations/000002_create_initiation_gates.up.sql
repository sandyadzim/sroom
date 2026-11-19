BEGIN;

-- Every attempt reservation and expiry closure locks the same Booking-owned ID.
-- There is intentionally no foreign key to another service's database.
CREATE TABLE payment.initiation_gates (
    booking_id UUID PRIMARY KEY,
    closed_at TIMESTAMPTZ
);

COMMIT;

BEGIN;

CREATE SCHEMA IF NOT EXISTS booking;
CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE booking.bookings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL,
    variant_id UUID NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending_payment', 'confirmed', 'expired', 'payment_review')),
    pickup_at TIMESTAMPTZ NOT NULL,
    return_at TIMESTAMPTZ NOT NULL,
    pickup_local_date DATE NOT NULL,
    return_local_date DATE NOT NULL,
    vehicle_label_snapshot TEXT NOT NULL,
    daily_rate_idr BIGINT NOT NULL CHECK (daily_rate_idr > 0),
    rental_days INTEGER NOT NULL CHECK (rental_days >= 1),
    total_idr BIGINT NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'IDR' CHECK (currency = 'IDR'),
    terms_version TEXT NOT NULL,
    terms_accepted_at TIMESTAMPTZ NOT NULL,
    hold_expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (isfinite(pickup_at) AND isfinite(return_at) AND return_at > pickup_at),
    CHECK (pickup_local_date = (pickup_at AT TIME ZONE 'Asia/Makassar')::DATE),
    CHECK (return_local_date = (return_at AT TIME ZONE 'Asia/Makassar')::DATE),
    CHECK (rental_days = return_local_date - pickup_local_date + 1),
    CHECK (total_idr = daily_rate_idr * rental_days)
);

CREATE INDEX bookings_customer_created_idx
    ON booking.bookings (customer_id, created_at DESC, id DESC);

CREATE INDEX bookings_due_holds_idx
    ON booking.bookings (hold_expires_at, id)
    WHERE status = 'pending_payment';

CREATE TABLE booking.allocations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id UUID NOT NULL UNIQUE REFERENCES booking.bookings(id),
    unit_id UUID NOT NULL,
    occupied_period TSTZRANGE NOT NULL,
    blocks_inventory BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    released_at TIMESTAMPTZ,
    CHECK (NOT isempty(occupied_period)),
    CHECK (lower_inc(occupied_period) AND NOT upper_inc(occupied_period)),
    CHECK (NOT lower_inf(occupied_period) AND NOT upper_inf(occupied_period)),
    CHECK (isfinite(lower(occupied_period)) AND isfinite(upper(occupied_period))),
    CHECK (blocks_inventory = (released_at IS NULL)),
    CONSTRAINT allocations_no_overlap EXCLUDE USING gist (
        unit_id WITH =,
        occupied_period WITH &&
    ) WHERE (blocks_inventory)
);

CREATE TABLE booking.booking_status_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id UUID NOT NULL REFERENCES booking.bookings(id),
    from_status TEXT CHECK (from_status IN ('pending_payment', 'confirmed', 'expired', 'payment_review')),
    to_status TEXT NOT NULL CHECK (to_status IN ('pending_payment', 'confirmed', 'expired', 'payment_review')),
    actor_type TEXT NOT NULL CHECK (actor_type IN ('system', 'customer', 'admin', 'provider_event')),
    reason_code TEXT,
    request_id TEXT,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX booking_status_history_booking_idx
    ON booking.booking_status_history (booking_id, occurred_at, id);

COMMIT;

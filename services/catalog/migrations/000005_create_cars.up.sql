BEGIN;

CREATE TABLE cars (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    brand_id UUID NOT NULL REFERENCES brands(id),
    model VARCHAR(100) NOT NULL CHECK (BTRIM(model) <> ''),
    trim VARCHAR(100) NOT NULL DEFAULT '',
    slug TEXT NOT NULL CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    category TEXT NOT NULL CHECK (category IN ('economy', 'city', 'mpv', 'suv', 'premium')),
    body_type TEXT NOT NULL CHECK (body_type IN ('hatchback', 'sedan', 'mpv', 'suv')),
    seats INTEGER NOT NULL CHECK (seats > 0),
    transmission TEXT NOT NULL CHECK (transmission IN ('automatic', 'manual')),
    fuel_type TEXT NOT NULL CHECK (fuel_type IN ('petrol', 'diesel', 'hybrid', 'electric')),
    description VARCHAR(2000) NOT NULL CHECK (BTRIM(description) <> ''),
    daily_rate_idr BIGINT NOT NULL CHECK (daily_rate_idr > 0),
    status TEXT NOT NULL DEFAULT 'inactive' CHECK (status IN ('active', 'inactive')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT cars_slug_key UNIQUE (slug)
);

CREATE UNIQUE INDEX cars_variant_unique_idx
    ON cars (brand_id, LOWER(BTRIM(model)), LOWER(BTRIM(trim)));

COMMIT;

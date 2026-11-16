BEGIN;

CREATE TABLE car_units (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    car_id UUID NOT NULL REFERENCES cars(id),
    internal_code VARCHAR(100) NOT NULL CHECK (BTRIM(internal_code) <> ''),
    status TEXT NOT NULL DEFAULT 'inactive' CHECK (status IN ('active', 'inactive')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX car_units_internal_code_unique_idx
    ON car_units (LOWER(BTRIM(internal_code)));

CREATE INDEX car_units_car_id_idx ON car_units (car_id);

COMMIT;

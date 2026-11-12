BEGIN;

CREATE TABLE car_images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    car_id UUID NOT NULL REFERENCES cars(id),
    object_key TEXT NOT NULL CHECK (BTRIM(object_key) <> ''),
    kind TEXT NOT NULL CHECK (kind IN ('cover', 'gallery')),
    alt_text VARCHAR(200) NOT NULL CHECK (BTRIM(alt_text) <> ''),
    label VARCHAR(100) NOT NULL CHECK (BTRIM(label) <> ''),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX car_images_car_id_idx ON car_images (car_id);

CREATE UNIQUE INDEX car_images_cover_unique_idx
    ON car_images (car_id) WHERE kind = 'cover';

COMMIT;

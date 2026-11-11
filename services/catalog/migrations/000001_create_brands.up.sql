BEGIN;

CREATE TABLE brands (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL CHECK (BTRIM(name) <> ''),
    image_object_key TEXT CHECK (BTRIM(image_object_key) <> ''),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX brands_normalized_name_unique_idx
    ON brands (LOWER(BTRIM(name)));

COMMIT;

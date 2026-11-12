BEGIN;

CREATE TABLE banners (
    position INTEGER PRIMARY KEY CHECK (position BETWEEN 1 AND 3),
    brand_id UUID REFERENCES brands(id),
    car_model TEXT,
    car_slug TEXT,
    headline_line_1 TEXT,
    headline_line_2 TEXT,
    description TEXT,
    image_object_key TEXT,
    CHECK (num_nonnulls(brand_id, car_model, car_slug, headline_line_1,
                       headline_line_2, description, image_object_key) IN (0, 7))
);

INSERT INTO banners (position) VALUES (1), (2), (3);

COMMIT;

BEGIN;

ALTER TABLE banners
    ADD COLUMN car_id UUID REFERENCES cars(id),
    ADD CONSTRAINT banners_car_requires_content CHECK (car_id IS NULL OR brand_id IS NOT NULL);

UPDATE banners
SET car_id = cars.id
FROM cars
WHERE banners.car_slug = cars.slug
  AND banners.brand_id = cars.brand_id;

COMMIT;

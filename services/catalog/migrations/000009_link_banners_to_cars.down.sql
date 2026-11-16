BEGIN;

ALTER TABLE banners
    DROP CONSTRAINT banners_car_requires_content,
    DROP COLUMN car_id;

COMMIT;

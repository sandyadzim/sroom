BEGIN;

ALTER TABLE car_images ADD COLUMN position INTEGER;

WITH ordered_images AS (
    SELECT id, ROW_NUMBER() OVER (
        PARTITION BY car_id, kind ORDER BY created_at, id
    ) - 1 AS position
    FROM car_images
)
UPDATE car_images
SET position = ordered_images.position
FROM ordered_images
WHERE car_images.id = ordered_images.id;

ALTER TABLE car_images
    ALTER COLUMN position SET NOT NULL,
    ADD CONSTRAINT car_images_position_nonnegative CHECK (position >= 0),
    ADD CONSTRAINT car_images_position_unique UNIQUE (car_id, kind, position);

COMMIT;

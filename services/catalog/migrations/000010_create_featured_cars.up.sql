BEGIN;

CREATE TABLE featured_cars (
    position INTEGER PRIMARY KEY CHECK (position BETWEEN 1 AND 3),
    car_id UUID NOT NULL UNIQUE REFERENCES cars(id)
);

COMMIT;

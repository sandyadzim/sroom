ALTER TABLE users
ADD COLUMN status VARCHAR(8) NOT NULL DEFAULT 'active';

ALTER TABLE users
ADD CONSTRAINT users_status_check
CHECK (status IN ('active', 'inactive'));

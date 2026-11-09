
ALTER TABLE users
ADD COLUMN role VARCHAR(20) NOT NULL DEFAULT 'CUSTOMER';

ALTER TABLE users
ADD CONSTRAINT users_role_check
CHECK (role IN ('CUSTOMER', 'ADMIN', 'OWNER'));

CREATE UNIQUE INDEX users_single_owner_idx
ON users (role)
WHERE role = 'OWNER';

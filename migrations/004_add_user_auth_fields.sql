ALTER TABLE users
ADD COLUMN password_hash TEXT;

ALTER TABLE users
ADD COLUMN role VARCHAR(20);

UPDATE users
SET role = 'user'
WHERE role IS NULL;

ALTER TABLE users
ALTER COLUMN role SET NOT NULL;

ALTER TABLE users
ALTER COLUMN role SET DEFAULT 'user';

ALTER TABLE users
ADD CONSTRAINT users_role_check
CHECK (role IN ('user', 'admin'));

ALTER TABLE orders
ADD CONSTRAINT fk_orders_user
FOREIGN KEY (user_id)
REFERENCES users(id);
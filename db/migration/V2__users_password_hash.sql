-- Bcrypt password hashes (never store plaintext). Register/login verify server-side.
-- If this fails because legacy rows exist without passwords, truncate `users` (or migrate data) then re-run Flyway repair/migrate.

ALTER TABLE users
    ADD COLUMN password_hash TEXT NOT NULL;

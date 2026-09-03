-- +goose Up
ALTER TABLE users ADD COLUMN hashed_passwords TEXT NOT NULL DEFAULT 'unset';
UPDATE users 
SET hashed_passwords = 'unset' 
WHERE hashed_passwords IS NULL;

-- +goose Down

ALTER TABLE users DROP COLUMN hashed_passwords;
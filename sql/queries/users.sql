-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_passwords)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2
)
RETURNING *;

-- name: DeleteAllUsers :exec

DELETE FROM users;

-- name: GetUserByID :one

SELECT * FROM users WHERE email = $1;
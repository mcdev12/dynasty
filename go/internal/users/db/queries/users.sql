-- name: CreateUser :one
INSERT INTO users (
    id,
    username,
    email
) VALUES (
    gen_random_uuid(),
    $1,
    $2
) RETURNING *;

-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: UpdateUser :one
UPDATE users SET
    username = $2,
    email = $3
WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;
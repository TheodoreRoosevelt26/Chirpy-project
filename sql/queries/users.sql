-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2
)
RETURNING *;

-- name: GetUserFromEmail :one
SELECT id, created_at, updated_at, email FROM users WHERE email = $1;

-- name: DeleteAllUsers :exec
DELETE FROM users;

-- name: UpdateUserEmailPassword :exec
UPDATE users 
SET email = COALESCE(NULLIF($1, email), email),
    hashed_password = COALESCE(NULLIF($2, hashed_password), hashed_password)
WHERE id = $3
RETURNING id, email;
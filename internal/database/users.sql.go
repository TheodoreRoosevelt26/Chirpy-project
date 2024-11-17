// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: users.sql

package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

const createUser = `-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2
)
RETURNING id, created_at, updated_at, email, hashed_password, is_chirpy_red
`

type CreateUserParams struct {
	Email          string
	HashedPassword string
}

func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (User, error) {
	row := q.db.QueryRowContext(ctx, createUser, arg.Email, arg.HashedPassword)
	var i User
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.Email,
		&i.HashedPassword,
		&i.IsChirpyRed,
	)
	return i, err
}

const deleteAllUsers = `-- name: DeleteAllUsers :exec
DELETE FROM users
`

func (q *Queries) DeleteAllUsers(ctx context.Context) error {
	_, err := q.db.ExecContext(ctx, deleteAllUsers)
	return err
}

const getUserFromEmail = `-- name: GetUserFromEmail :one
SELECT id, created_at, updated_at, email, is_chirpy_red FROM users WHERE email = $1
`

type GetUserFromEmailRow struct {
	ID          uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Email       string
	IsChirpyRed sql.NullBool
}

func (q *Queries) GetUserFromEmail(ctx context.Context, email string) (GetUserFromEmailRow, error) {
	row := q.db.QueryRowContext(ctx, getUserFromEmail, email)
	var i GetUserFromEmailRow
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.Email,
		&i.IsChirpyRed,
	)
	return i, err
}

const updateUserEmailPassword = `-- name: UpdateUserEmailPassword :exec
UPDATE users 
SET email = COALESCE(NULLIF($1, email), email),
    hashed_password = COALESCE(NULLIF($2, hashed_password), hashed_password)
WHERE id = $3
RETURNING id, email
`

type UpdateUserEmailPasswordParams struct {
	Email          string
	HashedPassword string
	ID             uuid.UUID
}

func (q *Queries) UpdateUserEmailPassword(ctx context.Context, arg UpdateUserEmailPasswordParams) error {
	_, err := q.db.ExecContext(ctx, updateUserEmailPassword, arg.Email, arg.HashedPassword, arg.ID)
	return err
}

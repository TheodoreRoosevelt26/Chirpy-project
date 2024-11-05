// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: refresh_token.sql

package database

import (
	"context"

	"github.com/google/uuid"
)

const registerRefreshToken = `-- name: RegisterRefreshToken :one
INSERT INTO refresh_tokens(token, created_at, updated_at, user_id, expires_at)
VALUES (
    $1,
    NOW(),
    NOW(),
    $2,
    NOW() + INTERVAL '60 days'
)
RETURNING token, created_at, updated_at, user_id, expires_at, revoked_at
`

type RegisterRefreshTokenParams struct {
	Token  string
	UserID uuid.UUID
}

func (q *Queries) RegisterRefreshToken(ctx context.Context, arg RegisterRefreshTokenParams) (RefreshToken, error) {
	row := q.db.QueryRowContext(ctx, registerRefreshToken, arg.Token, arg.UserID)
	var i RefreshToken
	err := row.Scan(
		&i.Token,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.UserID,
		&i.ExpiresAt,
		&i.RevokedAt,
	)
	return i, err
}

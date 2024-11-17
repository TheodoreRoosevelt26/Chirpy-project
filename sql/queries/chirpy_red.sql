-- name: UpgradeToRed :one
UPDATE users SET is_chirpy_red = true, updated_at = NOW() WHERE id = $1 RETURNING id, is_chirpy_red;
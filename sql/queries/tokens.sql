-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens(token, created_at, updated_at, user_id, expires_at, revoked_at)
VALUES (
    $1,
    NOW(),
    NOW(),
    $2,
    NOW() + INTERVAL '60 days',
    NULL
)
RETURNING *;

-- name: GetTokenByTokenValue :one
SELECT *
FROM refresh_tokens
WHERE token = $1;

-- name: GetUserFromRefreshToken :one
SELECT users.* 
FROM refresh_tokens
JOIN users
ON users.id = refresh_tokens.user_id
WHERE token = $1;

-- name: SetTokenTimestamps :execrows
UPDATE refresh_tokens
SET revoked_at = $1, updated_at = $2
WHERE token = $3 AND revoked_at IS NULL;
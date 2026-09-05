-- name: GetUserFromRefreshToken :one
SELECT * from refresh_tokens
WHERE token = $1;

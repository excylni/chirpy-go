-- name: GetChirpByAuthor :many
    SELECT * from chirps
    WHERE user_id = $1
    ORDER BY created_at ASC;
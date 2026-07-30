-- name: GetChirpByID :one
    SELECT * from chirps
    WHERE id = $1;
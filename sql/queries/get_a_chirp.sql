-- name: GetAChirp :one
SELECT * FROM chirps
WHERE id = $1;

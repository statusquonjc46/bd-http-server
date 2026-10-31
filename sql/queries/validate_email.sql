-- name: ValidateEmail :one
SELECT * FROM users
WHERE email = $1;

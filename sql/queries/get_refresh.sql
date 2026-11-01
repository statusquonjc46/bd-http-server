-- name: ValidateRefreshToken :one
SELECT * FROM refreshtokens
WHERE token = $1;

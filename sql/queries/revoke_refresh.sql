-- name: RevokeRefreshToken :exec
UPDATE refreshtokens
SET revoked_at = NOW(), updated_at = NOW()
WHERE token = $1;

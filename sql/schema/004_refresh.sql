-- +goose Up
CREATE TABLE refreshtokens (
    token TEXT NOT NULL PRIMARY KEY,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMP,
    revoked_at TIMESTAMP
);

-- +goose Down
DROP TABLE refreshtokens;

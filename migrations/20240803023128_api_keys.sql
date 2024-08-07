-- +goose Up
-- +goose StatementBegin
CrEaTe TABLE IF NOT EXISTS api_keys (
    id SERIAL PRIMARY KEY,
    user_id TEXT NOT NULL,
    `key` TEXT NOT NULL,
    valid BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS api_keys;
-- +goose StatementEnd

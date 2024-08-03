-- +goose Up
-- +goose StatementBegin
CrEaTe TABLE IF NOT EXISTS endpoints (
    id integer PRIMARY KEY AUTOINCREMENT,
    api_key_id integer NOT NULL,
    original_url TEXT NOT NULL,
    throttlr_url TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS endpoints;
-- +goose StatementEnd

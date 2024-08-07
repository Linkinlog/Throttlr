-- +goose Up
-- +goose StatementBegin
CrEaTe TABLE IF NOT EXISTS endpoints (
    id SERIAL PRIMARY KEY,
    api_key_id integer NOT NULL,
    original_url TEXT NOT NULL,
    throttlr_url TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX throttlr_url_idx ON endpoints(throttlr_url);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS endpoints;
DROP INDEX throttlr_url_idx;
-- +goose StatementEnd

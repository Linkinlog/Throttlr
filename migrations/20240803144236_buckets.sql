-- +goose Up
-- +goose StatementBegin
CrEaTe TABLE IF NOT EXISTS buckets (
    id SERIAL PRIMARY KEY,
    endpoint_id integer NOT NULL,
    max integer NOT NULL,
    interval integer NOT NULL,
    window_opened_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS buckets;
-- +goose StatementEnd

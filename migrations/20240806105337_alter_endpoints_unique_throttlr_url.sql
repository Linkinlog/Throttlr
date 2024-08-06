-- +goose Up
-- +goose StatementBegin
CREATE UNIQUE INDEX throttlr_url_idx ON endpoints(throttlr_url);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX throttlr_url_idx;
-- +goose StatementEnd

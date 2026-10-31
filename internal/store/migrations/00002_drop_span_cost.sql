-- +goose Up
ALTER TABLE spans DROP COLUMN cost_usd;

-- +goose Down
ALTER TABLE spans ADD COLUMN cost_usd REAL;

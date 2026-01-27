-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    login         TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS accounts (
    user_id   BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    balance   NUMERIC(12,2) NOT NULL DEFAULT 0,
    withdrawn NUMERIC(12,2) NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS orders (
    number      TEXT PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status      TEXT NOT NULL,
    accrual     NUMERIC(12,2),
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS withdrawals (
    order_number TEXT PRIMARY KEY,
    user_id      BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    sum          NUMERIC(12,2) NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS withdrawals;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS accounts;
DROP TABLE IF EXISTS users;
-- +goose StatementEnd

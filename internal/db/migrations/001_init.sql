CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS plaid_items (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    item_id TEXT NOT NULL UNIQUE,
    access_token_ciphertext TEXT NOT NULL,
    access_token_nonce TEXT NOT NULL,
    access_token_key_id TEXT NOT NULL,
    access_token_algorithm TEXT NOT NULL,
    transactions_cursor TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS bank_accounts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plaid_item_id BIGINT NOT NULL REFERENCES plaid_items(id) ON DELETE CASCADE,
    plaid_account_id TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    official_name TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL DEFAULT '',
    subtype TEXT NOT NULL DEFAULT '',
    mask TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS transactions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    bank_account_id BIGINT NOT NULL REFERENCES bank_accounts(id) ON DELETE CASCADE,
    plaid_transaction_id TEXT NOT NULL UNIQUE,
    amount NUMERIC(14, 2) NOT NULL,
    iso_currency_code TEXT NOT NULL DEFAULT '',
    transaction_date DATE NOT NULL,
    pending BOOLEAN NOT NULL DEFAULT false,
    description_ciphertext TEXT NOT NULL,
    description_nonce TEXT NOT NULL,
    description_key_id TEXT NOT NULL,
    description_algorithm TEXT NOT NULL,
    merchant_name TEXT NOT NULL DEFAULT '',
    source_environment TEXT NOT NULL DEFAULT 'sandbox',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS webhook_events (
    id BIGSERIAL PRIMARY KEY,
    event_key TEXT NOT NULL UNIQUE,
    item_id TEXT NOT NULL DEFAULT '',
    webhook_type TEXT NOT NULL DEFAULT '',
    webhook_code TEXT NOT NULL DEFAULT '',
    payload JSONB NOT NULL,
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);


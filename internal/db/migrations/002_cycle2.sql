ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS category        TEXT,
    ADD COLUMN IF NOT EXISTS ml_confidence   REAL,
    ADD COLUMN IF NOT EXISTS categorized_at  TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS transactions_category_idx ON transactions (category);

CREATE TABLE IF NOT EXISTS arbitrage_opportunities (
    id               BIGSERIAL PRIMARY KEY,
    user_id          BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    transaction_id   BIGINT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    merchant_name    TEXT NOT NULL,
    category         TEXT NOT NULL,
    current_amount   NUMERIC(14, 2) NOT NULL,
    market_rate      NUMERIC(14, 2),
    savings_estimate NUMERIC(14, 2),
    provider_url     TEXT NOT NULL DEFAULT '',
    found_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS arbitrage_opportunities_user_id_idx ON arbitrage_opportunities (user_id);
CREATE INDEX IF NOT EXISTS arbitrage_opportunities_transaction_id_idx ON arbitrage_opportunities (transaction_id);

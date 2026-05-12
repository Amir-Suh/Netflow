CREATE TABLE IF NOT EXISTS quant_profiles (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    monthly_surplus NUMERIC(14, 2) NOT NULL DEFAULT 0,
    cash_balance NUMERIC(14, 2) NOT NULL DEFAULT 0,
    emergency_savings_balance NUMERIC(14, 2) NOT NULL DEFAULT 0,
    current_investment_balance NUMERIC(14, 2) NOT NULL DEFAULT 0,
    monthly_contribution NUMERIC(14, 2) NOT NULL DEFAULT 0,
    target_wealth NUMERIC(14, 2) NOT NULL DEFAULT 0,
    risk_tolerance TEXT NOT NULL DEFAULT 'moderate',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS debts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    balance NUMERIC(14, 2) NOT NULL,
    apr DOUBLE PRECISION NOT NULL,
    minimum_payment NUMERIC(14, 2) NOT NULL,
    source TEXT NOT NULL DEFAULT 'manual',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT debts_nonnegative_values CHECK (balance >= 0 AND apr >= 0 AND minimum_payment >= 0),
    CONSTRAINT debts_user_name_unique UNIQUE (user_id, name)
);

CREATE INDEX IF NOT EXISTS debts_user_id_idx ON debts (user_id);

CREATE TABLE IF NOT EXISTS debt_optimization_runs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'queued',
    preference TEXT NOT NULL DEFAULT 'lowest_interest',
    monthly_surplus NUMERIC(14, 2) NOT NULL DEFAULT 0,
    recommended_strategy TEXT,
    payoff_months INTEGER,
    payoff_date DATE,
    total_interest NUMERIC(14, 2),
    request_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    assumptions_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS debt_optimization_runs_user_id_idx ON debt_optimization_runs (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS debt_optimization_strategy_summaries (
    id BIGSERIAL PRIMARY KEY,
    run_id BIGINT NOT NULL REFERENCES debt_optimization_runs(id) ON DELETE CASCADE,
    strategy TEXT NOT NULL,
    feasible BOOLEAN NOT NULL DEFAULT false,
    payoff_months INTEGER,
    payoff_date DATE,
    total_interest NUMERIC(14, 2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT debt_strategy_run_unique UNIQUE (run_id, strategy)
);

CREATE TABLE IF NOT EXISTS debt_optimization_steps (
    id BIGSERIAL PRIMARY KEY,
    run_id BIGINT NOT NULL REFERENCES debt_optimization_runs(id) ON DELETE CASCADE,
    strategy TEXT NOT NULL,
    month_index INTEGER NOT NULL,
    debt_id BIGINT,
    debt_name TEXT NOT NULL,
    starting_balance NUMERIC(14, 2) NOT NULL,
    payment NUMERIC(14, 2) NOT NULL,
    interest NUMERIC(14, 2) NOT NULL,
    principal NUMERIC(14, 2) NOT NULL,
    ending_balance NUMERIC(14, 2) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS debt_optimization_steps_run_id_idx ON debt_optimization_steps (run_id, strategy, month_index);

CREATE TABLE IF NOT EXISTS investment_holdings (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    symbol TEXT NOT NULL,
    name TEXT NOT NULL,
    asset_class TEXT NOT NULL,
    quantity DOUBLE PRECISION NOT NULL DEFAULT 0,
    current_value NUMERIC(14, 2) NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'USD',
    source TEXT NOT NULL DEFAULT 'manual',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT investment_holdings_nonnegative_values CHECK (quantity >= 0 AND current_value >= 0),
    CONSTRAINT investment_holdings_user_symbol_unique UNIQUE (user_id, symbol)
);

CREATE INDEX IF NOT EXISTS investment_holdings_user_id_idx ON investment_holdings (user_id);

CREATE TABLE IF NOT EXISTS asset_class_assumptions (
    id BIGSERIAL PRIMARY KEY,
    method TEXT NOT NULL,
    asset_class TEXT NOT NULL,
    expected_return DOUBLE PRECISION NOT NULL,
    volatility DOUBLE PRECISION NOT NULL,
    parameters JSONB NOT NULL DEFAULT '{}'::jsonb,
    source TEXT NOT NULL DEFAULT 'demo',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT asset_class_assumptions_unique UNIQUE (method, asset_class, source)
);

CREATE INDEX IF NOT EXISTS asset_class_assumptions_method_idx ON asset_class_assumptions (method, is_active);

CREATE TABLE IF NOT EXISTS benchmark_return_series (
    id BIGSERIAL PRIMARY KEY,
    asset_class TEXT NOT NULL,
    period_start DATE NOT NULL,
    period TEXT NOT NULL DEFAULT 'monthly',
    return_value DOUBLE PRECISION NOT NULL,
    source TEXT NOT NULL DEFAULT 'demo_fixture',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT benchmark_return_series_unique UNIQUE (asset_class, period_start, period, source)
);

CREATE INDEX IF NOT EXISTS benchmark_return_series_asset_class_idx ON benchmark_return_series (asset_class, period_start);

CREATE TABLE IF NOT EXISTS wealth_simulation_runs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'queued',
    seed BIGINT NOT NULL DEFAULT 424242,
    scenario_count INTEGER NOT NULL DEFAULT 10000,
    horizons JSONB NOT NULL DEFAULT '[10,20,30]'::jsonb,
    current_assets NUMERIC(14, 2) NOT NULL DEFAULT 0,
    monthly_contribution NUMERIC(14, 2) NOT NULL DEFAULT 0,
    target_wealth NUMERIC(14, 2) NOT NULL DEFAULT 0,
    assumption_method TEXT NOT NULL DEFAULT 'demo_static',
    assumptions_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    covariance_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS wealth_simulation_runs_user_id_idx ON wealth_simulation_runs (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS wealth_simulation_percentiles (
    id BIGSERIAL PRIMARY KEY,
    run_id BIGINT NOT NULL REFERENCES wealth_simulation_runs(id) ON DELETE CASCADE,
    horizon_years INTEGER NOT NULL,
    percentile INTEGER NOT NULL,
    nominal_value NUMERIC(14, 2) NOT NULL,
    real_value NUMERIC(14, 2) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT wealth_percentiles_unique UNIQUE (run_id, horizon_years, percentile)
);

CREATE TABLE IF NOT EXISTS wealth_simulation_goal_probabilities (
    id BIGSERIAL PRIMARY KEY,
    run_id BIGINT NOT NULL REFERENCES wealth_simulation_runs(id) ON DELETE CASCADE,
    horizon_years INTEGER NOT NULL,
    target_wealth NUMERIC(14, 2) NOT NULL,
    probability DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT wealth_goal_probabilities_unique UNIQUE (run_id, horizon_years)
);

CREATE TABLE IF NOT EXISTS portfolio_optimization_runs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'queued',
    assumption_method TEXT NOT NULL DEFAULT 'demo_static',
    risk_tolerance TEXT NOT NULL DEFAULT 'moderate',
    portfolio_value NUMERIC(14, 2) NOT NULL DEFAULT 0,
    current_return DOUBLE PRECISION,
    current_volatility DOUBLE PRECISION,
    current_sharpe DOUBLE PRECISION,
    min_variance_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    max_sharpe_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    recommended_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    assumptions_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    covariance_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS portfolio_optimization_runs_user_id_idx ON portfolio_optimization_runs (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS portfolio_frontier_points (
    id BIGSERIAL PRIMARY KEY,
    run_id BIGINT NOT NULL REFERENCES portfolio_optimization_runs(id) ON DELETE CASCADE,
    point_index INTEGER NOT NULL,
    expected_return DOUBLE PRECISION NOT NULL,
    volatility DOUBLE PRECISION NOT NULL,
    sharpe DOUBLE PRECISION NOT NULL,
    weights JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT portfolio_frontier_points_unique UNIQUE (run_id, point_index)
);

CREATE TABLE IF NOT EXISTS portfolio_recommendations (
    id BIGSERIAL PRIMARY KEY,
    run_id BIGINT NOT NULL REFERENCES portfolio_optimization_runs(id) ON DELETE CASCADE,
    asset_class TEXT NOT NULL,
    current_weight DOUBLE PRECISION NOT NULL,
    target_weight DOUBLE PRECISION NOT NULL,
    current_value NUMERIC(14, 2) NOT NULL,
    target_value NUMERIC(14, 2) NOT NULL,
    dollar_delta NUMERIC(14, 2) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT portfolio_recommendations_unique UNIQUE (run_id, asset_class)
);

INSERT INTO benchmark_return_series (asset_class, period_start, period, return_value, source)
SELECT
    asset_class,
    (DATE '2021-01-01' + (month_index::text || ' months')::interval)::date AS period_start,
    'monthly',
    CASE asset_class
        WHEN 'cash' THEN 0.00120 + (((month_index % 3) - 1) * 0.00005)
        WHEN 'bonds' THEN 0.00280 + (((month_index % 7) - 3) * 0.00110)
        WHEN 'us_equity' THEN 0.00600 + (((month_index % 11) - 5) * 0.00420)
        WHEN 'international_equity' THEN 0.00550 + (((month_index % 13) - 6) * 0.00480)
        ELSE 0
    END AS return_value,
    'demo_fixture'
FROM generate_series(0, 59) AS month_index
CROSS JOIN (VALUES
    ('cash'),
    ('bonds'),
    ('us_equity'),
    ('international_equity')
) AS classes(asset_class)
ON CONFLICT (asset_class, period_start, period, source) DO NOTHING;

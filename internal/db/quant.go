package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type QuantProfile struct {
	UserID                   int64   `json:"user_id"`
	MonthlySurplus           float64 `json:"monthly_surplus"`
	CashBalance              float64 `json:"cash_balance"`
	EmergencySavingsBalance  float64 `json:"emergency_savings_balance"`
	CurrentInvestmentBalance float64 `json:"current_investment_balance"`
	MonthlyContribution      float64 `json:"monthly_contribution"`
	TargetWealth             float64 `json:"target_wealth"`
	RiskTolerance            string  `json:"risk_tolerance"`
}

type Debt struct {
	ID             int64   `json:"id"`
	UserID         int64   `json:"user_id"`
	Name           string  `json:"name"`
	Balance        float64 `json:"balance"`
	APR            float64 `json:"apr"`
	MinimumPayment float64 `json:"minimum_payment"`
	Source         string  `json:"source"`
	IsActive       bool    `json:"is_active"`
}

type DebtInput struct {
	UserID         int64
	Name           string
	Balance        float64
	APR            float64
	MinimumPayment float64
	Source         string
	IsActive       bool
}

type InvestmentHolding struct {
	ID           int64   `json:"id"`
	UserID       int64   `json:"user_id"`
	Symbol       string  `json:"symbol"`
	Name         string  `json:"name"`
	AssetClass   string  `json:"asset_class"`
	Quantity     float64 `json:"quantity"`
	CurrentValue float64 `json:"current_value"`
	Currency     string  `json:"currency"`
	Source       string  `json:"source"`
}

type HoldingInput struct {
	UserID       int64
	Symbol       string
	Name         string
	AssetClass   string
	Quantity     float64
	CurrentValue float64
	Currency     string
	Source       string
}

type AssetClassAssumption struct {
	Method         string          `json:"method"`
	AssetClass     string          `json:"asset_class"`
	ExpectedReturn float64         `json:"expected_return"`
	Volatility     float64         `json:"volatility"`
	Parameters     json.RawMessage `json:"parameters"`
	Source         string          `json:"source"`
	IsActive       bool            `json:"is_active"`
}

type Cycle3SeedSummary struct {
	Debts       int `json:"debts"`
	Holdings    int `json:"holdings"`
	Assumptions int `json:"assumptions"`
	Benchmarks  int `json:"benchmark_return_rows"`
}

type DebtRun struct {
	ID                  int64           `json:"id"`
	UserID              int64           `json:"user_id"`
	Status              string          `json:"status"`
	Preference          string          `json:"preference"`
	MonthlySurplus      float64         `json:"monthly_surplus"`
	RecommendedStrategy string          `json:"recommended_strategy"`
	PayoffMonths        int             `json:"payoff_months"`
	PayoffDate          string          `json:"payoff_date"`
	TotalInterest       float64         `json:"total_interest"`
	RequestSnapshot     json.RawMessage `json:"request_snapshot"`
	AssumptionsSnapshot json.RawMessage `json:"assumptions_snapshot"`
	ErrorMessage        string          `json:"error_message"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

type DebtStrategySummary struct {
	Strategy      string  `json:"strategy"`
	Feasible      bool    `json:"feasible"`
	PayoffMonths  int     `json:"payoff_months"`
	PayoffDate    string  `json:"payoff_date"`
	TotalInterest float64 `json:"total_interest"`
}

type DebtOptimizationStep struct {
	Strategy        string  `json:"strategy"`
	MonthIndex      int     `json:"month_index"`
	DebtID          int64   `json:"debt_id"`
	DebtName        string  `json:"debt_name"`
	StartingBalance float64 `json:"starting_balance"`
	Payment         float64 `json:"payment"`
	Interest        float64 `json:"interest"`
	Principal       float64 `json:"principal"`
	EndingBalance   float64 `json:"ending_balance"`
}

type DebtRunDetails struct {
	Run       DebtRun                `json:"run"`
	Summaries []DebtStrategySummary  `json:"strategy_summaries"`
	Steps     []DebtOptimizationStep `json:"steps"`
}

type WealthRun struct {
	ID                  int64           `json:"id"`
	UserID              int64           `json:"user_id"`
	Status              string          `json:"status"`
	Seed                int64           `json:"seed"`
	ScenarioCount       int             `json:"scenario_count"`
	Horizons            json.RawMessage `json:"horizons"`
	CurrentAssets       float64         `json:"current_assets"`
	MonthlyContribution float64         `json:"monthly_contribution"`
	TargetWealth        float64         `json:"target_wealth"`
	AssumptionMethod    string          `json:"assumption_method"`
	AssumptionsSnapshot json.RawMessage `json:"assumptions_snapshot"`
	CovarianceSnapshot  json.RawMessage `json:"covariance_snapshot"`
	ErrorMessage        string          `json:"error_message"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

type WealthPercentile struct {
	HorizonYears int     `json:"horizon_years"`
	Percentile   int     `json:"percentile"`
	NominalValue float64 `json:"nominal_value"`
	RealValue    float64 `json:"real_value"`
}

type WealthGoalProbability struct {
	HorizonYears int     `json:"horizon_years"`
	TargetWealth float64 `json:"target_wealth"`
	Probability  float64 `json:"probability"`
}

type WealthRunDetails struct {
	Run               WealthRun               `json:"run"`
	Percentiles       []WealthPercentile      `json:"percentiles"`
	GoalProbabilities []WealthGoalProbability `json:"goal_probabilities"`
}

type PortfolioRun struct {
	ID                  int64           `json:"id"`
	UserID              int64           `json:"user_id"`
	Status              string          `json:"status"`
	AssumptionMethod    string          `json:"assumption_method"`
	RiskTolerance       string          `json:"risk_tolerance"`
	PortfolioValue      float64         `json:"portfolio_value"`
	CurrentReturn       float64         `json:"current_return"`
	CurrentVolatility   float64         `json:"current_volatility"`
	CurrentSharpe       float64         `json:"current_sharpe"`
	MinVarianceSnapshot json.RawMessage `json:"min_variance_snapshot"`
	MaxSharpeSnapshot   json.RawMessage `json:"max_sharpe_snapshot"`
	RecommendedSnapshot json.RawMessage `json:"recommended_snapshot"`
	AssumptionsSnapshot json.RawMessage `json:"assumptions_snapshot"`
	CovarianceSnapshot  json.RawMessage `json:"covariance_snapshot"`
	ErrorMessage        string          `json:"error_message"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

type PortfolioFrontierPoint struct {
	PointIndex     int             `json:"point_index"`
	ExpectedReturn float64         `json:"expected_return"`
	Volatility     float64         `json:"volatility"`
	Sharpe         float64         `json:"sharpe"`
	Weights        json.RawMessage `json:"weights"`
}

type PortfolioRecommendation struct {
	AssetClass    string  `json:"asset_class"`
	CurrentWeight float64 `json:"current_weight"`
	TargetWeight  float64 `json:"target_weight"`
	CurrentValue  float64 `json:"current_value"`
	TargetValue   float64 `json:"target_value"`
	DollarDelta   float64 `json:"dollar_delta"`
}

type PortfolioRunDetails struct {
	Run             PortfolioRun              `json:"run"`
	FrontierPoints  []PortfolioFrontierPoint  `json:"frontier_points"`
	Recommendations []PortfolioRecommendation `json:"recommendations"`
}

func defaultRawMessage(raw []byte) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	return json.RawMessage(raw)
}

func marshalJSON(value any) (string, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (s *Store) SeedCycle3Demo(ctx context.Context, userID int64) (Cycle3SeedSummary, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Cycle3SeedSummary{}, fmt.Errorf("begin cycle3 seed: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`INSERT INTO quant_profiles (
			user_id, monthly_surplus, cash_balance, emergency_savings_balance,
			current_investment_balance, monthly_contribution, target_wealth, risk_tolerance
		) VALUES ($1, 600, 3200, 15000, 125000, 950, 1000000, 'moderate')
		ON CONFLICT (user_id) DO UPDATE SET
			monthly_surplus = EXCLUDED.monthly_surplus,
			cash_balance = EXCLUDED.cash_balance,
			emergency_savings_balance = EXCLUDED.emergency_savings_balance,
			current_investment_balance = EXCLUDED.current_investment_balance,
			monthly_contribution = EXCLUDED.monthly_contribution,
			target_wealth = EXCLUDED.target_wealth,
			risk_tolerance = EXCLUDED.risk_tolerance,
			updated_at = now()`,
		userID,
	); err != nil {
		return Cycle3SeedSummary{}, fmt.Errorf("seed quant profile: %w", err)
	}

	debts := []DebtInput{
		{UserID: userID, Name: "Credit Card A", Balance: 4200, APR: 0.2499, MinimumPayment: 125, Source: "demo_seed", IsActive: true},
		{UserID: userID, Name: "Credit Card B", Balance: 1700, APR: 0.1999, MinimumPayment: 60, Source: "demo_seed", IsActive: true},
		{UserID: userID, Name: "Student Loan", Balance: 18000, APR: 0.0575, MinimumPayment: 190, Source: "demo_seed", IsActive: true},
		{UserID: userID, Name: "Auto Loan", Balance: 9500, APR: 0.0720, MinimumPayment: 275, Source: "demo_seed", IsActive: true},
	}
	for _, debt := range debts {
		if _, err := tx.Exec(ctx,
			`INSERT INTO debts (user_id, name, balance, apr, minimum_payment, source, is_active)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)
			 ON CONFLICT (user_id, name) DO UPDATE SET
				balance = EXCLUDED.balance,
				apr = EXCLUDED.apr,
				minimum_payment = EXCLUDED.minimum_payment,
				source = EXCLUDED.source,
				is_active = EXCLUDED.is_active,
				updated_at = now()`,
			debt.UserID, debt.Name, debt.Balance, debt.APR, debt.MinimumPayment, debt.Source, debt.IsActive,
		); err != nil {
			return Cycle3SeedSummary{}, fmt.Errorf("seed debt %s: %w", debt.Name, err)
		}
	}

	holdings := []HoldingInput{
		{UserID: userID, Symbol: "VTI", Name: "US Total Market ETF", AssetClass: "us_equity", Quantity: 620, CurrentValue: 70000, Currency: "USD", Source: "demo_seed"},
		{UserID: userID, Symbol: "VXUS", Name: "International Equity ETF", AssetClass: "international_equity", Quantity: 520, CurrentValue: 25000, Currency: "USD", Source: "demo_seed"},
		{UserID: userID, Symbol: "BND", Name: "Bond ETF", AssetClass: "bonds", Quantity: 310, CurrentValue: 25000, Currency: "USD", Source: "demo_seed"},
		{UserID: userID, Symbol: "CASH", Name: "Cash", AssetClass: "cash", Quantity: 1, CurrentValue: 5000, Currency: "USD", Source: "demo_seed"},
	}
	for _, holding := range holdings {
		if _, err := tx.Exec(ctx,
			`INSERT INTO investment_holdings (
				user_id, symbol, name, asset_class, quantity, current_value, currency, source
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (user_id, symbol) DO UPDATE SET
				name = EXCLUDED.name,
				asset_class = EXCLUDED.asset_class,
				quantity = EXCLUDED.quantity,
				current_value = EXCLUDED.current_value,
				currency = EXCLUDED.currency,
				source = EXCLUDED.source,
				updated_at = now()`,
			holding.UserID, holding.Symbol, holding.Name, holding.AssetClass, holding.Quantity,
			holding.CurrentValue, holding.Currency, holding.Source,
		); err != nil {
			return Cycle3SeedSummary{}, fmt.Errorf("seed holding %s: %w", holding.Symbol, err)
		}
	}

	type assumptionSeed struct {
		assetClass     string
		expectedReturn float64
		volatility     float64
		parameters     map[string]any
	}
	assumptions := []assumptionSeed{
		{"cash", 0.015, 0.010, map[string]any{"inflation": 0.025}},
		{"bonds", 0.035, 0.050, map[string]any{"inflation": 0.025}},
		{"us_equity", 0.070, 0.150, map[string]any{"inflation": 0.025, "risk_free_rate": 0.04, "market_risk_premium": 0.05, "beta": 1.0}},
		{"international_equity", 0.065, 0.170, map[string]any{"inflation": 0.025, "risk_free_rate": 0.04, "market_risk_premium": 0.05, "beta": 1.05}},
	}
	for _, assumption := range assumptions {
		parameters, err := marshalJSON(assumption.parameters)
		if err != nil {
			return Cycle3SeedSummary{}, fmt.Errorf("marshal assumption: %w", err)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO asset_class_assumptions (
				method, asset_class, expected_return, volatility, parameters, source, is_active
			) VALUES ('demo_static', $1, $2, $3, $4::jsonb, 'demo_seed', true)
			ON CONFLICT (method, asset_class, source) DO UPDATE SET
				expected_return = EXCLUDED.expected_return,
				volatility = EXCLUDED.volatility,
				parameters = EXCLUDED.parameters,
				is_active = EXCLUDED.is_active,
				updated_at = now()`,
			assumption.assetClass, assumption.expectedReturn, assumption.volatility, parameters,
		); err != nil {
			return Cycle3SeedSummary{}, fmt.Errorf("seed assumption %s: %w", assumption.assetClass, err)
		}
	}

	var benchmarkRows int
	if err := tx.QueryRow(ctx,
		`SELECT COUNT(*) FROM benchmark_return_series WHERE source = 'demo_fixture'`,
	).Scan(&benchmarkRows); err != nil {
		return Cycle3SeedSummary{}, fmt.Errorf("count benchmark fixtures: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Cycle3SeedSummary{}, fmt.Errorf("commit cycle3 seed: %w", err)
	}
	return Cycle3SeedSummary{
		Debts:       len(debts),
		Holdings:    len(holdings),
		Assumptions: len(assumptions),
		Benchmarks:  benchmarkRows,
	}, nil
}

func (s *Store) GetQuantProfile(ctx context.Context, userID int64) (QuantProfile, error) {
	var profile QuantProfile
	err := s.pool.QueryRow(ctx,
		`SELECT user_id, monthly_surplus, cash_balance, emergency_savings_balance,
			current_investment_balance, monthly_contribution, target_wealth, risk_tolerance
		 FROM quant_profiles WHERE user_id = $1`,
		userID,
	).Scan(
		&profile.UserID, &profile.MonthlySurplus, &profile.CashBalance,
		&profile.EmergencySavingsBalance, &profile.CurrentInvestmentBalance,
		&profile.MonthlyContribution, &profile.TargetWealth, &profile.RiskTolerance,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return QuantProfile{UserID: userID, RiskTolerance: "moderate"}, nil
	}
	if err != nil {
		return QuantProfile{}, fmt.Errorf("get quant profile: %w", err)
	}
	return profile, nil
}

func (s *Store) ListDebts(ctx context.Context, userID int64) ([]Debt, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, user_id, name, balance, apr, minimum_payment, source, is_active
		 FROM debts WHERE user_id = $1 ORDER BY is_active DESC, name`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list debts: %w", err)
	}
	defer rows.Close()

	var debts []Debt
	for rows.Next() {
		var debt Debt
		if err := rows.Scan(&debt.ID, &debt.UserID, &debt.Name, &debt.Balance, &debt.APR, &debt.MinimumPayment, &debt.Source, &debt.IsActive); err != nil {
			return nil, fmt.Errorf("scan debt: %w", err)
		}
		debts = append(debts, debt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate debts: %w", err)
	}
	return debts, nil
}

func (s *Store) UpsertDebt(ctx context.Context, input DebtInput) (Debt, error) {
	if input.Source == "" {
		input.Source = "manual"
	}
	var debt Debt
	err := s.pool.QueryRow(ctx,
		`INSERT INTO debts (user_id, name, balance, apr, minimum_payment, source, is_active)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (user_id, name) DO UPDATE SET
			balance = EXCLUDED.balance,
			apr = EXCLUDED.apr,
			minimum_payment = EXCLUDED.minimum_payment,
			source = EXCLUDED.source,
			is_active = EXCLUDED.is_active,
			updated_at = now()
		 RETURNING id, user_id, name, balance, apr, minimum_payment, source, is_active`,
		input.UserID, input.Name, input.Balance, input.APR, input.MinimumPayment, input.Source, input.IsActive,
	).Scan(&debt.ID, &debt.UserID, &debt.Name, &debt.Balance, &debt.APR, &debt.MinimumPayment, &debt.Source, &debt.IsActive)
	if err != nil {
		return Debt{}, fmt.Errorf("upsert debt: %w", err)
	}
	return debt, nil
}

func (s *Store) ListHoldings(ctx context.Context, userID int64) ([]InvestmentHolding, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, user_id, symbol, name, asset_class, quantity, current_value, currency, source
		 FROM investment_holdings WHERE user_id = $1 ORDER BY asset_class, symbol`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list holdings: %w", err)
	}
	defer rows.Close()

	var holdings []InvestmentHolding
	for rows.Next() {
		var holding InvestmentHolding
		if err := rows.Scan(
			&holding.ID, &holding.UserID, &holding.Symbol, &holding.Name, &holding.AssetClass,
			&holding.Quantity, &holding.CurrentValue, &holding.Currency, &holding.Source,
		); err != nil {
			return nil, fmt.Errorf("scan holding: %w", err)
		}
		holdings = append(holdings, holding)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate holdings: %w", err)
	}
	return holdings, nil
}

func (s *Store) UpsertHolding(ctx context.Context, input HoldingInput) (InvestmentHolding, error) {
	if input.Currency == "" {
		input.Currency = "USD"
	}
	if input.Source == "" {
		input.Source = "manual"
	}
	var holding InvestmentHolding
	err := s.pool.QueryRow(ctx,
		`INSERT INTO investment_holdings (
			user_id, symbol, name, asset_class, quantity, current_value, currency, source
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id, symbol) DO UPDATE SET
			name = EXCLUDED.name,
			asset_class = EXCLUDED.asset_class,
			quantity = EXCLUDED.quantity,
			current_value = EXCLUDED.current_value,
			currency = EXCLUDED.currency,
			source = EXCLUDED.source,
			updated_at = now()
		RETURNING id, user_id, symbol, name, asset_class, quantity, current_value, currency, source`,
		input.UserID, input.Symbol, input.Name, input.AssetClass, input.Quantity, input.CurrentValue,
		input.Currency, input.Source,
	).Scan(
		&holding.ID, &holding.UserID, &holding.Symbol, &holding.Name, &holding.AssetClass,
		&holding.Quantity, &holding.CurrentValue, &holding.Currency, &holding.Source,
	)
	if err != nil {
		return InvestmentHolding{}, fmt.Errorf("upsert holding: %w", err)
	}
	return holding, nil
}

func (s *Store) ListAssumptions(ctx context.Context) ([]AssetClassAssumption, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT method, asset_class, expected_return, volatility, parameters, source, is_active
		 FROM asset_class_assumptions
		 ORDER BY method, asset_class, source`,
	)
	if err != nil {
		return nil, fmt.Errorf("list assumptions: %w", err)
	}
	defer rows.Close()

	var assumptions []AssetClassAssumption
	for rows.Next() {
		var assumption AssetClassAssumption
		var parameters []byte
		if err := rows.Scan(
			&assumption.Method, &assumption.AssetClass, &assumption.ExpectedReturn,
			&assumption.Volatility, &parameters, &assumption.Source, &assumption.IsActive,
		); err != nil {
			return nil, fmt.Errorf("scan assumption: %w", err)
		}
		assumption.Parameters = defaultRawMessage(parameters)
		assumptions = append(assumptions, assumption)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate assumptions: %w", err)
	}
	return assumptions, nil
}

func (s *Store) CreateDebtOptimizationRun(ctx context.Context, userID int64, preference string, monthlySurplus float64, requestSnapshot any) (int64, error) {
	requestJSON, err := marshalJSON(requestSnapshot)
	if err != nil {
		return 0, fmt.Errorf("marshal debt run request: %w", err)
	}
	var id int64
	err = s.pool.QueryRow(ctx,
		`INSERT INTO debt_optimization_runs (user_id, status, preference, monthly_surplus, request_snapshot)
		 VALUES ($1, 'queued', $2, $3, $4::jsonb)
		 RETURNING id`,
		userID, preference, monthlySurplus, requestJSON,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create debt optimization run: %w", err)
	}
	return id, nil
}

func (s *Store) CreateWealthSimulationRun(ctx context.Context, userID int64, seed int64, scenarioCount int, horizons []int, currentAssets, monthlyContribution, targetWealth float64, method string) (int64, error) {
	horizonsJSON, err := marshalJSON(horizons)
	if err != nil {
		return 0, fmt.Errorf("marshal horizons: %w", err)
	}
	var id int64
	err = s.pool.QueryRow(ctx,
		`INSERT INTO wealth_simulation_runs (
			user_id, status, seed, scenario_count, horizons, current_assets,
			monthly_contribution, target_wealth, assumption_method
		) VALUES ($1, 'queued', $2, $3, $4::jsonb, $5, $6, $7, $8)
		RETURNING id`,
		userID, seed, scenarioCount, horizonsJSON, currentAssets, monthlyContribution, targetWealth, method,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create wealth simulation run: %w", err)
	}
	return id, nil
}

func (s *Store) CreatePortfolioOptimizationRun(ctx context.Context, userID int64, method, riskTolerance string, portfolioValue float64) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx,
		`INSERT INTO portfolio_optimization_runs (
			user_id, status, assumption_method, risk_tolerance, portfolio_value
		) VALUES ($1, 'queued', $2, $3, $4)
		RETURNING id`,
		userID, method, riskTolerance, portfolioValue,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create portfolio optimization run: %w", err)
	}
	return id, nil
}

func (s *Store) GetDebtRunDetails(ctx context.Context, userID, runID int64) (DebtRunDetails, error) {
	run, err := s.getDebtRun(ctx, userID, runID)
	if err != nil {
		return DebtRunDetails{}, err
	}
	summaries, err := s.listDebtStrategySummaries(ctx, run.ID)
	if err != nil {
		return DebtRunDetails{}, err
	}
	steps, err := s.listDebtOptimizationSteps(ctx, run.ID)
	if err != nil {
		return DebtRunDetails{}, err
	}
	return DebtRunDetails{Run: run, Summaries: summaries, Steps: steps}, nil
}

func (s *Store) GetLatestDebtRunDetails(ctx context.Context, userID int64) (DebtRunDetails, error) {
	runID, err := s.latestRunID(ctx, `debt_optimization_runs`, userID)
	if err != nil {
		return DebtRunDetails{}, err
	}
	return s.GetDebtRunDetails(ctx, userID, runID)
}

func (s *Store) getDebtRun(ctx context.Context, userID, runID int64) (DebtRun, error) {
	var run DebtRun
	var requestSnapshot, assumptionsSnapshot []byte
	err := s.pool.QueryRow(ctx,
		`SELECT id, user_id, status, preference, monthly_surplus,
			COALESCE(recommended_strategy, ''), COALESCE(payoff_months, 0),
			COALESCE(payoff_date::text, ''), COALESCE(total_interest, 0),
			request_snapshot, assumptions_snapshot, error_message, created_at, updated_at
		 FROM debt_optimization_runs WHERE user_id = $1 AND id = $2`,
		userID, runID,
	).Scan(
		&run.ID, &run.UserID, &run.Status, &run.Preference, &run.MonthlySurplus,
		&run.RecommendedStrategy, &run.PayoffMonths, &run.PayoffDate, &run.TotalInterest,
		&requestSnapshot, &assumptionsSnapshot, &run.ErrorMessage, &run.CreatedAt, &run.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return DebtRun{}, pgx.ErrNoRows
	}
	if err != nil {
		return DebtRun{}, fmt.Errorf("get debt run: %w", err)
	}
	run.RequestSnapshot = defaultRawMessage(requestSnapshot)
	run.AssumptionsSnapshot = defaultRawMessage(assumptionsSnapshot)
	return run, nil
}

func (s *Store) listDebtStrategySummaries(ctx context.Context, runID int64) ([]DebtStrategySummary, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT strategy, feasible, COALESCE(payoff_months, 0), COALESCE(payoff_date::text, ''), COALESCE(total_interest, 0)
		 FROM debt_optimization_strategy_summaries
		 WHERE run_id = $1
		 ORDER BY strategy`,
		runID,
	)
	if err != nil {
		return nil, fmt.Errorf("list debt summaries: %w", err)
	}
	defer rows.Close()
	var summaries []DebtStrategySummary
	for rows.Next() {
		var summary DebtStrategySummary
		if err := rows.Scan(&summary.Strategy, &summary.Feasible, &summary.PayoffMonths, &summary.PayoffDate, &summary.TotalInterest); err != nil {
			return nil, fmt.Errorf("scan debt summary: %w", err)
		}
		summaries = append(summaries, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate debt summaries: %w", err)
	}
	return summaries, nil
}

func (s *Store) listDebtOptimizationSteps(ctx context.Context, runID int64) ([]DebtOptimizationStep, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT strategy, month_index, COALESCE(debt_id, 0), debt_name,
			starting_balance, payment, interest, principal, ending_balance
		 FROM debt_optimization_steps
		 WHERE run_id = $1
		 ORDER BY strategy, month_index, debt_name`,
		runID,
	)
	if err != nil {
		return nil, fmt.Errorf("list debt steps: %w", err)
	}
	defer rows.Close()
	var steps []DebtOptimizationStep
	for rows.Next() {
		var step DebtOptimizationStep
		if err := rows.Scan(
			&step.Strategy, &step.MonthIndex, &step.DebtID, &step.DebtName,
			&step.StartingBalance, &step.Payment, &step.Interest, &step.Principal, &step.EndingBalance,
		); err != nil {
			return nil, fmt.Errorf("scan debt step: %w", err)
		}
		steps = append(steps, step)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate debt steps: %w", err)
	}
	return steps, nil
}

func (s *Store) GetWealthRunDetails(ctx context.Context, userID, runID int64) (WealthRunDetails, error) {
	run, err := s.getWealthRun(ctx, userID, runID)
	if err != nil {
		return WealthRunDetails{}, err
	}
	percentiles, err := s.listWealthPercentiles(ctx, run.ID)
	if err != nil {
		return WealthRunDetails{}, err
	}
	goals, err := s.listWealthGoalProbabilities(ctx, run.ID)
	if err != nil {
		return WealthRunDetails{}, err
	}
	return WealthRunDetails{Run: run, Percentiles: percentiles, GoalProbabilities: goals}, nil
}

func (s *Store) GetLatestWealthRunDetails(ctx context.Context, userID int64) (WealthRunDetails, error) {
	runID, err := s.latestRunID(ctx, `wealth_simulation_runs`, userID)
	if err != nil {
		return WealthRunDetails{}, err
	}
	return s.GetWealthRunDetails(ctx, userID, runID)
}

func (s *Store) getWealthRun(ctx context.Context, userID, runID int64) (WealthRun, error) {
	var run WealthRun
	var horizons, assumptionsSnapshot, covarianceSnapshot []byte
	err := s.pool.QueryRow(ctx,
		`SELECT id, user_id, status, seed, scenario_count, horizons, current_assets,
			monthly_contribution, target_wealth, assumption_method,
			assumptions_snapshot, covariance_snapshot, error_message, created_at, updated_at
		 FROM wealth_simulation_runs WHERE user_id = $1 AND id = $2`,
		userID, runID,
	).Scan(
		&run.ID, &run.UserID, &run.Status, &run.Seed, &run.ScenarioCount, &horizons,
		&run.CurrentAssets, &run.MonthlyContribution, &run.TargetWealth, &run.AssumptionMethod,
		&assumptionsSnapshot, &covarianceSnapshot, &run.ErrorMessage, &run.CreatedAt, &run.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return WealthRun{}, pgx.ErrNoRows
	}
	if err != nil {
		return WealthRun{}, fmt.Errorf("get wealth run: %w", err)
	}
	run.Horizons = defaultRawMessage(horizons)
	run.AssumptionsSnapshot = defaultRawMessage(assumptionsSnapshot)
	run.CovarianceSnapshot = defaultRawMessage(covarianceSnapshot)
	return run, nil
}

func (s *Store) listWealthPercentiles(ctx context.Context, runID int64) ([]WealthPercentile, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT horizon_years, percentile, nominal_value, real_value
		 FROM wealth_simulation_percentiles
		 WHERE run_id = $1
		 ORDER BY horizon_years, percentile`,
		runID,
	)
	if err != nil {
		return nil, fmt.Errorf("list wealth percentiles: %w", err)
	}
	defer rows.Close()
	var percentiles []WealthPercentile
	for rows.Next() {
		var percentile WealthPercentile
		if err := rows.Scan(&percentile.HorizonYears, &percentile.Percentile, &percentile.NominalValue, &percentile.RealValue); err != nil {
			return nil, fmt.Errorf("scan wealth percentile: %w", err)
		}
		percentiles = append(percentiles, percentile)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate wealth percentiles: %w", err)
	}
	return percentiles, nil
}

func (s *Store) listWealthGoalProbabilities(ctx context.Context, runID int64) ([]WealthGoalProbability, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT horizon_years, target_wealth, probability
		 FROM wealth_simulation_goal_probabilities
		 WHERE run_id = $1
		 ORDER BY horizon_years`,
		runID,
	)
	if err != nil {
		return nil, fmt.Errorf("list wealth goals: %w", err)
	}
	defer rows.Close()
	var goals []WealthGoalProbability
	for rows.Next() {
		var goal WealthGoalProbability
		if err := rows.Scan(&goal.HorizonYears, &goal.TargetWealth, &goal.Probability); err != nil {
			return nil, fmt.Errorf("scan wealth goal: %w", err)
		}
		goals = append(goals, goal)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate wealth goals: %w", err)
	}
	return goals, nil
}

func (s *Store) GetPortfolioRunDetails(ctx context.Context, userID, runID int64) (PortfolioRunDetails, error) {
	run, err := s.getPortfolioRun(ctx, userID, runID)
	if err != nil {
		return PortfolioRunDetails{}, err
	}
	points, err := s.listPortfolioFrontierPoints(ctx, run.ID)
	if err != nil {
		return PortfolioRunDetails{}, err
	}
	recommendations, err := s.listPortfolioRecommendations(ctx, run.ID)
	if err != nil {
		return PortfolioRunDetails{}, err
	}
	return PortfolioRunDetails{Run: run, FrontierPoints: points, Recommendations: recommendations}, nil
}

func (s *Store) GetLatestPortfolioRunDetails(ctx context.Context, userID int64) (PortfolioRunDetails, error) {
	runID, err := s.latestRunID(ctx, `portfolio_optimization_runs`, userID)
	if err != nil {
		return PortfolioRunDetails{}, err
	}
	return s.GetPortfolioRunDetails(ctx, userID, runID)
}

func (s *Store) getPortfolioRun(ctx context.Context, userID, runID int64) (PortfolioRun, error) {
	var run PortfolioRun
	var minVariance, maxSharpe, recommended, assumptions, covariance []byte
	err := s.pool.QueryRow(ctx,
		`SELECT id, user_id, status, assumption_method, risk_tolerance, portfolio_value,
			COALESCE(current_return, 0), COALESCE(current_volatility, 0), COALESCE(current_sharpe, 0),
			min_variance_snapshot, max_sharpe_snapshot, recommended_snapshot,
			assumptions_snapshot, covariance_snapshot, error_message, created_at, updated_at
		 FROM portfolio_optimization_runs WHERE user_id = $1 AND id = $2`,
		userID, runID,
	).Scan(
		&run.ID, &run.UserID, &run.Status, &run.AssumptionMethod, &run.RiskTolerance,
		&run.PortfolioValue, &run.CurrentReturn, &run.CurrentVolatility, &run.CurrentSharpe,
		&minVariance, &maxSharpe, &recommended, &assumptions, &covariance,
		&run.ErrorMessage, &run.CreatedAt, &run.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return PortfolioRun{}, pgx.ErrNoRows
	}
	if err != nil {
		return PortfolioRun{}, fmt.Errorf("get portfolio run: %w", err)
	}
	run.MinVarianceSnapshot = defaultRawMessage(minVariance)
	run.MaxSharpeSnapshot = defaultRawMessage(maxSharpe)
	run.RecommendedSnapshot = defaultRawMessage(recommended)
	run.AssumptionsSnapshot = defaultRawMessage(assumptions)
	run.CovarianceSnapshot = defaultRawMessage(covariance)
	return run, nil
}

func (s *Store) listPortfolioFrontierPoints(ctx context.Context, runID int64) ([]PortfolioFrontierPoint, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT point_index, expected_return, volatility, sharpe, weights
		 FROM portfolio_frontier_points
		 WHERE run_id = $1
		 ORDER BY point_index`,
		runID,
	)
	if err != nil {
		return nil, fmt.Errorf("list portfolio frontier: %w", err)
	}
	defer rows.Close()
	var points []PortfolioFrontierPoint
	for rows.Next() {
		var point PortfolioFrontierPoint
		var weights []byte
		if err := rows.Scan(&point.PointIndex, &point.ExpectedReturn, &point.Volatility, &point.Sharpe, &weights); err != nil {
			return nil, fmt.Errorf("scan portfolio frontier: %w", err)
		}
		point.Weights = defaultRawMessage(weights)
		points = append(points, point)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate portfolio frontier: %w", err)
	}
	return points, nil
}

func (s *Store) listPortfolioRecommendations(ctx context.Context, runID int64) ([]PortfolioRecommendation, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT asset_class, current_weight, target_weight, current_value, target_value, dollar_delta
		 FROM portfolio_recommendations
		 WHERE run_id = $1
		 ORDER BY asset_class`,
		runID,
	)
	if err != nil {
		return nil, fmt.Errorf("list portfolio recommendations: %w", err)
	}
	defer rows.Close()
	var recommendations []PortfolioRecommendation
	for rows.Next() {
		var recommendation PortfolioRecommendation
		if err := rows.Scan(
			&recommendation.AssetClass, &recommendation.CurrentWeight, &recommendation.TargetWeight,
			&recommendation.CurrentValue, &recommendation.TargetValue, &recommendation.DollarDelta,
		); err != nil {
			return nil, fmt.Errorf("scan portfolio recommendation: %w", err)
		}
		recommendations = append(recommendations, recommendation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate portfolio recommendations: %w", err)
	}
	return recommendations, nil
}

func (s *Store) latestRunID(ctx context.Context, table string, userID int64) (int64, error) {
	switch table {
	case "debt_optimization_runs", "wealth_simulation_runs", "portfolio_optimization_runs":
	default:
		return 0, fmt.Errorf("unsupported run table %q", table)
	}
	var id int64
	query := fmt.Sprintf(`SELECT id FROM %s WHERE user_id = $1 ORDER BY created_at DESC, id DESC LIMIT 1`, table)
	if err := s.pool.QueryRow(ctx, query, userID).Scan(&id); errors.Is(err, pgx.ErrNoRows) {
		return 0, pgx.ErrNoRows
	} else if err != nil {
		return 0, fmt.Errorf("latest run id: %w", err)
	}
	return id, nil
}

package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"netflow/internal/db"
	"netflow/internal/queue"
)

const (
	defaultQuantSeed          int64 = 424242
	defaultQuantScenarioCount       = 10000
)

var defaultQuantHorizons = []int{10, 20, 30}

func (s *Server) registerQuantRoutes(mux *http.ServeMux) {
	mux.Handle("POST /demo/seed-cycle-3", s.requireAuth(http.HandlerFunc(s.seedCycle3Demo)))
	mux.Handle("GET /quant/debts", s.requireAuth(http.HandlerFunc(s.listDebts)))
	mux.Handle("POST /quant/debts", s.requireAuth(http.HandlerFunc(s.createDebt)))
	mux.Handle("POST /quant/debt/optimize", s.requireAuth(http.HandlerFunc(s.optimizeDebt)))
	mux.Handle("GET /quant/debt/runs/latest", s.requireAuth(http.HandlerFunc(s.latestDebtRun)))
	mux.Handle("GET /quant/debt/runs/{id}", s.requireAuth(http.HandlerFunc(s.getDebtRun)))
	mux.Handle("POST /quant/wealth/simulate", s.requireAuth(http.HandlerFunc(s.simulateWealth)))
	mux.Handle("GET /quant/wealth/runs/latest", s.requireAuth(http.HandlerFunc(s.latestWealthRun)))
	mux.Handle("GET /quant/wealth/runs/{id}", s.requireAuth(http.HandlerFunc(s.getWealthRun)))
	mux.Handle("GET /quant/holdings", s.requireAuth(http.HandlerFunc(s.listHoldings)))
	mux.Handle("POST /quant/holdings", s.requireAuth(http.HandlerFunc(s.createHolding)))
	mux.Handle("POST /quant/portfolio/optimize", s.requireAuth(http.HandlerFunc(s.optimizePortfolio)))
	mux.Handle("GET /quant/portfolio/runs/latest", s.requireAuth(http.HandlerFunc(s.latestPortfolioRun)))
	mux.Handle("GET /quant/portfolio/runs/{id}", s.requireAuth(http.HandlerFunc(s.getPortfolioRun)))
	mux.Handle("GET /quant/assumptions", s.requireAuth(http.HandlerFunc(s.listAssumptions)))
}

func (s *Server) seedCycle3Demo(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.Demo.SeedEnabled || s.cfg.Plaid.Env != "sandbox" {
		writeError(w, http.StatusForbidden, "cycle 3 demo seeding is disabled")
		return
	}
	summary, err := s.store.SeedCycle3Demo(r.Context(), userIDFromContext(r.Context()))
	if err != nil {
		s.logger.Error("cycle 3 seed failed", "error", err)
		writeError(w, http.StatusInternalServerError, "cycle 3 seed failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "seeded", "summary": summary})
}

func (s *Server) listDebts(w http.ResponseWriter, r *http.Request) {
	debts, err := s.store.ListDebts(r.Context(), userIDFromContext(r.Context()))
	if err != nil {
		s.logger.Error("list debts failed", "error", err)
		writeError(w, http.StatusInternalServerError, "list debts failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"debts": debts})
}

type debtRequest struct {
	Name           string  `json:"name"`
	Balance        float64 `json:"balance"`
	APR            float64 `json:"apr"`
	MinimumPayment float64 `json:"minimum_payment"`
	Source         string  `json:"source"`
	IsActive       *bool   `json:"is_active"`
}

func (s *Server) createDebt(w http.ResponseWriter, r *http.Request) {
	var req debtRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "debt name is required")
		return
	}
	if req.Balance < 0 || req.APR < 0 || req.MinimumPayment < 0 {
		writeError(w, http.StatusBadRequest, "debt values must be nonnegative")
		return
	}
	if req.APR > 1 {
		req.APR = req.APR / 100
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	debt, err := s.store.UpsertDebt(r.Context(), db.DebtInput{
		UserID:         userIDFromContext(r.Context()),
		Name:           req.Name,
		Balance:        req.Balance,
		APR:            req.APR,
		MinimumPayment: req.MinimumPayment,
		Source:         strings.TrimSpace(req.Source),
		IsActive:       isActive,
	})
	if err != nil {
		s.logger.Error("create debt failed", "error", err)
		writeError(w, http.StatusInternalServerError, "debt could not be saved")
		return
	}
	writeJSON(w, http.StatusCreated, debt)
}

type debtOptimizeRequest struct {
	Preference     string   `json:"preference"`
	MonthlySurplus *float64 `json:"monthly_surplus"`
}

func (s *Server) optimizeDebt(w http.ResponseWriter, r *http.Request) {
	var req debtOptimizeRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	userID := userIDFromContext(r.Context())
	preference := normalizeDebtPreference(req.Preference)
	profile, err := s.store.GetQuantProfile(r.Context(), userID)
	if err != nil {
		s.logger.Error("load quant profile failed", "error", err)
		writeError(w, http.StatusInternalServerError, "profile lookup failed")
		return
	}
	monthlySurplus := profile.MonthlySurplus
	if req.MonthlySurplus != nil {
		monthlySurplus = *req.MonthlySurplus
	}
	if monthlySurplus < 0 {
		writeError(w, http.StatusBadRequest, "monthly_surplus must be nonnegative")
		return
	}
	runID, err := s.store.CreateDebtOptimizationRun(r.Context(), userID, preference, monthlySurplus, map[string]any{
		"preference":      preference,
		"monthly_surplus": monthlySurplus,
	})
	if err != nil {
		s.logger.Error("create debt optimization run failed", "error", err)
		writeError(w, http.StatusInternalServerError, "debt optimization run could not be created")
		return
	}
	if err := s.publishQuantJob(r, userID, runID, "debt"); err != nil {
		s.logger.Error("publish debt quant job failed", "error", err)
		writeError(w, http.StatusBadGateway, "debt optimization job could not be queued")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"run_id": runID, "status": "queued"})
}

func normalizeDebtPreference(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "fastest_payoff", "snowball":
		return "fastest_payoff"
	case "lowest_interest", "avalanche", "":
		return "lowest_interest"
	default:
		return "lowest_interest"
	}
}

func (s *Server) latestDebtRun(w http.ResponseWriter, r *http.Request) {
	details, err := s.store.GetLatestDebtRunDetails(r.Context(), userIDFromContext(r.Context()))
	writeRunDetails(w, s, details, err, "debt run not found")
}

func (s *Server) getDebtRun(w http.ResponseWriter, r *http.Request) {
	runID, ok := parseRunID(w, r)
	if !ok {
		return
	}
	details, err := s.store.GetDebtRunDetails(r.Context(), userIDFromContext(r.Context()), runID)
	writeRunDetails(w, s, details, err, "debt run not found")
}

type wealthSimulateRequest struct {
	Seed             *int64 `json:"seed"`
	ScenarioCount    *int   `json:"scenario_count"`
	Horizons         []int  `json:"horizons"`
	AssumptionMethod string `json:"assumption_method"`
}

func (s *Server) simulateWealth(w http.ResponseWriter, r *http.Request) {
	var req wealthSimulateRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	userID := userIDFromContext(r.Context())
	profile, err := s.store.GetQuantProfile(r.Context(), userID)
	if err != nil {
		s.logger.Error("load quant profile failed", "error", err)
		writeError(w, http.StatusInternalServerError, "profile lookup failed")
		return
	}
	holdings, err := s.store.ListHoldings(r.Context(), userID)
	if err != nil {
		s.logger.Error("load holdings failed", "error", err)
		writeError(w, http.StatusInternalServerError, "holdings lookup failed")
		return
	}
	holdingValue := sumHoldingValue(holdings)
	currentAssets := profile.CashBalance + profile.EmergencySavingsBalance
	if holdingValue > 0 {
		currentAssets += holdingValue
	} else {
		currentAssets += profile.CurrentInvestmentBalance
	}
	seed := defaultQuantSeed
	if req.Seed != nil {
		seed = *req.Seed
	}
	scenarioCount := defaultQuantScenarioCount
	if req.ScenarioCount != nil {
		scenarioCount = *req.ScenarioCount
	}
	if scenarioCount <= 0 || scenarioCount > 100000 {
		writeError(w, http.StatusBadRequest, "scenario_count must be between 1 and 100000")
		return
	}
	horizons := defaultQuantHorizons
	if len(req.Horizons) > 0 {
		horizons = req.Horizons
	}
	for _, horizon := range horizons {
		if horizon <= 0 || horizon > 60 {
			writeError(w, http.StatusBadRequest, "horizons must be between 1 and 60 years")
			return
		}
	}
	method := normalizeAssumptionMethod(req.AssumptionMethod)
	runID, err := s.store.CreateWealthSimulationRun(
		r.Context(), userID, seed, scenarioCount, horizons,
		currentAssets, profile.MonthlyContribution, profile.TargetWealth, method,
	)
	if err != nil {
		s.logger.Error("create wealth simulation run failed", "error", err)
		writeError(w, http.StatusInternalServerError, "wealth simulation run could not be created")
		return
	}
	if err := s.publishQuantJob(r, userID, runID, "wealth"); err != nil {
		s.logger.Error("publish wealth quant job failed", "error", err)
		writeError(w, http.StatusBadGateway, "wealth simulation job could not be queued")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"run_id": runID, "status": "queued"})
}

func (s *Server) latestWealthRun(w http.ResponseWriter, r *http.Request) {
	details, err := s.store.GetLatestWealthRunDetails(r.Context(), userIDFromContext(r.Context()))
	writeRunDetails(w, s, details, err, "wealth run not found")
}

func (s *Server) getWealthRun(w http.ResponseWriter, r *http.Request) {
	runID, ok := parseRunID(w, r)
	if !ok {
		return
	}
	details, err := s.store.GetWealthRunDetails(r.Context(), userIDFromContext(r.Context()), runID)
	writeRunDetails(w, s, details, err, "wealth run not found")
}

func (s *Server) listHoldings(w http.ResponseWriter, r *http.Request) {
	holdings, err := s.store.ListHoldings(r.Context(), userIDFromContext(r.Context()))
	if err != nil {
		s.logger.Error("list holdings failed", "error", err)
		writeError(w, http.StatusInternalServerError, "list holdings failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"holdings": holdings})
}

type holdingRequest struct {
	Symbol       string  `json:"symbol"`
	Name         string  `json:"name"`
	AssetClass   string  `json:"asset_class"`
	Quantity     float64 `json:"quantity"`
	CurrentValue float64 `json:"current_value"`
	Currency     string  `json:"currency"`
	Source       string  `json:"source"`
}

func (s *Server) createHolding(w http.ResponseWriter, r *http.Request) {
	var req holdingRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Symbol = strings.ToUpper(strings.TrimSpace(req.Symbol))
	req.Name = strings.TrimSpace(req.Name)
	req.AssetClass = normalizeAssetClass(req.AssetClass, req.Symbol, req.Name)
	if req.Symbol == "" || req.Name == "" || req.AssetClass == "" {
		writeError(w, http.StatusBadRequest, "symbol, name, and asset_class are required")
		return
	}
	if !isSupportedAssetClass(req.AssetClass) {
		writeError(w, http.StatusBadRequest, "asset_class must be one of cash, bonds, us_equity, international_equity")
		return
	}
	if req.Quantity < 0 || req.CurrentValue < 0 {
		writeError(w, http.StatusBadRequest, "holding values must be nonnegative")
		return
	}
	holding, err := s.store.UpsertHolding(r.Context(), db.HoldingInput{
		UserID:       userIDFromContext(r.Context()),
		Symbol:       req.Symbol,
		Name:         req.Name,
		AssetClass:   req.AssetClass,
		Quantity:     req.Quantity,
		CurrentValue: req.CurrentValue,
		Currency:     strings.TrimSpace(req.Currency),
		Source:       strings.TrimSpace(req.Source),
	})
	if err != nil {
		s.logger.Error("create holding failed", "error", err)
		writeError(w, http.StatusInternalServerError, "holding could not be saved")
		return
	}
	writeJSON(w, http.StatusCreated, holding)
}

type portfolioOptimizeRequest struct {
	AssumptionMethod string `json:"assumption_method"`
	RiskTolerance    string `json:"risk_tolerance"`
}

func (s *Server) optimizePortfolio(w http.ResponseWriter, r *http.Request) {
	var req portfolioOptimizeRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	userID := userIDFromContext(r.Context())
	profile, err := s.store.GetQuantProfile(r.Context(), userID)
	if err != nil {
		s.logger.Error("load quant profile failed", "error", err)
		writeError(w, http.StatusInternalServerError, "profile lookup failed")
		return
	}
	holdings, err := s.store.ListHoldings(r.Context(), userID)
	if err != nil {
		s.logger.Error("load holdings failed", "error", err)
		writeError(w, http.StatusInternalServerError, "holdings lookup failed")
		return
	}
	portfolioValue := sumHoldingValue(holdings)
	if portfolioValue <= 0 {
		writeError(w, http.StatusBadRequest, "at least one positive holding is required")
		return
	}
	method := normalizeAssumptionMethod(req.AssumptionMethod)
	riskTolerance := normalizeRiskTolerance(req.RiskTolerance, profile.RiskTolerance)
	runID, err := s.store.CreatePortfolioOptimizationRun(r.Context(), userID, method, riskTolerance, portfolioValue)
	if err != nil {
		s.logger.Error("create portfolio optimization run failed", "error", err)
		writeError(w, http.StatusInternalServerError, "portfolio optimization run could not be created")
		return
	}
	if err := s.publishQuantJob(r, userID, runID, "portfolio"); err != nil {
		s.logger.Error("publish portfolio quant job failed", "error", err)
		writeError(w, http.StatusBadGateway, "portfolio optimization job could not be queued")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"run_id": runID, "status": "queued"})
}

func (s *Server) latestPortfolioRun(w http.ResponseWriter, r *http.Request) {
	details, err := s.store.GetLatestPortfolioRunDetails(r.Context(), userIDFromContext(r.Context()))
	writeRunDetails(w, s, details, err, "portfolio run not found")
}

func (s *Server) getPortfolioRun(w http.ResponseWriter, r *http.Request) {
	runID, ok := parseRunID(w, r)
	if !ok {
		return
	}
	details, err := s.store.GetPortfolioRunDetails(r.Context(), userIDFromContext(r.Context()), runID)
	writeRunDetails(w, s, details, err, "portfolio run not found")
}

func (s *Server) listAssumptions(w http.ResponseWriter, r *http.Request) {
	assumptions, err := s.store.ListAssumptions(r.Context())
	if err != nil {
		s.logger.Error("list assumptions failed", "error", err)
		writeError(w, http.StatusInternalServerError, "list assumptions failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"methods":     []string{"demo_static", "historical_sample", "capm", "black_litterman", "historical_with_shrinkage"},
		"assumptions": assumptions,
		"disclaimer":  "Educational projections only; not financial advice.",
	})
}

func (s *Server) publishQuantJob(r *http.Request, userID, runID int64, jobType string) error {
	return s.publisher.PublishQuantJob(r.Context(), queue.QuantJobEvent{
		RunID:      runID,
		UserID:     userID,
		JobType:    jobType,
		OccurredAt: time.Now().UTC(),
	})
}

func parseRunID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "valid run id is required")
		return 0, false
	}
	return id, true
}

func writeRunDetails(w http.ResponseWriter, s *Server, details any, err error, notFoundMessage string) {
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, notFoundMessage)
		return
	}
	if err != nil {
		s.logger.Error("load quant run failed", "error", err)
		writeError(w, http.StatusInternalServerError, "quant run lookup failed")
		return
	}
	writeJSON(w, http.StatusOK, details)
}

func normalizeAssumptionMethod(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "historical_sample", "capm", "black_litterman", "historical_with_shrinkage":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "demo_static"
	}
}

func normalizeRiskTolerance(value, fallback string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		value = strings.ToLower(strings.TrimSpace(fallback))
	}
	switch value {
	case "conservative", "moderate", "aggressive":
		return value
	default:
		return "moderate"
	}
}

func normalizeAssetClass(value, symbol, name string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "cash", "bonds", "us_equity", "international_equity":
		return value
	}
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	name = strings.ToLower(strings.TrimSpace(name))
	switch symbol {
	case "VTI", "VOO", "SPY":
		return "us_equity"
	case "VXUS", "IXUS":
		return "international_equity"
	case "BND", "AGG":
		return "bonds"
	case "CASH", "CHECKING", "SAVINGS":
		return "cash"
	}
	if strings.Contains(name, "international") {
		return "international_equity"
	}
	if strings.Contains(name, "bond") {
		return "bonds"
	}
	if strings.Contains(name, "cash") || strings.Contains(name, "savings") || strings.Contains(name, "checking") {
		return "cash"
	}
	if strings.Contains(name, "stock") || strings.Contains(name, "equity") || strings.Contains(name, "market") {
		return "us_equity"
	}
	return value
}

func isSupportedAssetClass(value string) bool {
	switch value {
	case "cash", "bonds", "us_equity", "international_equity":
		return true
	default:
		return false
	}
}

func sumHoldingValue(holdings []db.InvestmentHolding) float64 {
	var total float64
	for _, holding := range holdings {
		total += holding.CurrentValue
	}
	return total
}

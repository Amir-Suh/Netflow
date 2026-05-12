from __future__ import annotations

import numpy as np

from assumptions.black_litterman import estimate_black_litterman
from assumptions.capm import estimate_capm
from assumptions.historical import estimate_historical
from debt.solver import Debt, solve_debts
from portfolio.optimizer import optimize_portfolio
from wealth.simulator import simulate_wealth


def test_debt_solver_prefers_avalanche_for_lowest_interest():
    debts = [
        Debt(id=1, name="High APR", balance=1000, apr=0.24, minimum_payment=50),
        Debt(id=2, name="Low APR", balance=1000, apr=0.06, minimum_payment=50),
    ]
    result = solve_debts(debts, monthly_surplus=300, preference="lowest_interest")
    assert result["recommended_strategy"] == "optimized"
    assert result["recommended"].feasible
    assert result["recommended"].payoff_months > 0
    assert result["strategies"]["avalanche"].total_interest <= result["strategies"]["snowball"].total_interest


def test_debt_solver_rejects_negative_values():
    debts = [Debt(id=1, name="Bad", balance=-1, apr=0.1, minimum_payment=10)]
    try:
        solve_debts(debts, monthly_surplus=0)
    except ValueError as exc:
        assert "nonnegative" in str(exc)
    else:
        raise AssertionError("expected ValueError")


def test_monte_carlo_is_reproducible_with_seed():
    weights = np.array([0.2, 0.3, 0.5])
    returns = np.array([0.02, 0.04, 0.07])
    cov = np.diag([0.01, 0.04, 0.09])
    first = simulate_wealth(10000, 500, weights, returns, cov, 0.025, [10], 1000, 123, 100000)
    second = simulate_wealth(10000, 500, weights, returns, cov, 0.025, [10], 1000, 123, 100000)
    assert first["percentiles"] == second["percentiles"]
    assert first["goal_probabilities"] == second["goal_probabilities"]


def test_portfolio_optimizer_weights_sum_to_one():
    asset_classes = ["cash", "bonds", "us_equity", "international_equity"]
    current = np.array([0.05, 0.25, 0.5, 0.2])
    returns = np.array([0.015, 0.035, 0.07, 0.065])
    cov = np.diag([0.0001, 0.0025, 0.0225, 0.0289])
    result = optimize_portfolio(asset_classes, current, returns, cov, risk_tolerance="moderate")
    weights = result["recommended"]["weights"]
    assert abs(sum(weights.values()) - 1.0) < 1e-6
    assert all(value >= 0 for value in weights.values())
    assert len(result["frontier"]) > 3


def test_capm_expected_return_formula():
    assumptions, _ = estimate_capm(["us_equity"])
    expected = assumptions["asset_classes"]["us_equity"]["expected_return"]
    assert round(expected, 4) == 0.09


def test_historical_estimators_return_covariance():
    assumptions, cov = estimate_historical(["cash", "bonds", "us_equity"], shrinkage=True)
    assert assumptions["method"] == "historical_with_shrinkage"
    assert cov.shape == (3, 3)


def test_black_litterman_partial_equilibrium():
    assumptions, cov = estimate_black_litterman(["cash", "bonds", "us_equity"])
    assert assumptions["method"] == "black_litterman"
    assert assumptions["implementation_status"] == "partial_equilibrium_no_view_demo"
    assert cov.shape == (3, 3)

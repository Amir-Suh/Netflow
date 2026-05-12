"""Monte Carlo wealth simulation."""

from __future__ import annotations

import numpy as np


def simulate_wealth(
    current_assets: float,
    monthly_contribution: float,
    weights: np.ndarray,
    expected_returns: np.ndarray,
    covariance: np.ndarray,
    inflation: float,
    horizons: list[int],
    scenario_count: int = 10000,
    seed: int = 424242,
    target_wealth: float = 0.0,
    percentiles: list[int] | None = None,
) -> dict:
    if current_assets < 0 or monthly_contribution < 0:
        raise ValueError("assets and contributions must be nonnegative")
    if scenario_count <= 0:
        raise ValueError("scenario_count must be positive")
    if not horizons or any(horizon <= 0 for horizon in horizons):
        raise ValueError("horizons must be positive")
    percentiles = percentiles or [10, 50, 90]

    weights = np.asarray(weights, dtype=float)
    if weights.sum() <= 0:
        raise ValueError("weights must have positive sum")
    weights = weights / weights.sum()

    monthly_mean = np.power(1.0 + np.asarray(expected_returns, dtype=float), 1.0 / 12.0) - 1.0
    monthly_cov = np.asarray(covariance, dtype=float) / 12.0
    max_months = max(horizons) * 12
    horizon_months = {horizon * 12: horizon for horizon in horizons}

    rng = np.random.default_rng(seed)
    values = np.full(scenario_count, current_assets, dtype=float)
    captures: dict[int, np.ndarray] = {}

    for month in range(1, max_months + 1):
        asset_returns = rng.multivariate_normal(monthly_mean, monthly_cov, scenario_count, check_valid="ignore")
        portfolio_returns = asset_returns @ weights
        values = np.maximum(0.0, values * (1.0 + portfolio_returns) + monthly_contribution)
        if month in horizon_months:
            captures[horizon_months[month]] = values.copy()

    percentile_rows = []
    goal_rows = []
    for horizon in horizons:
        nominal = captures[horizon]
        real_discount = (1.0 + inflation) ** horizon
        for percentile in percentiles:
            nominal_value = float(np.percentile(nominal, percentile))
            percentile_rows.append(
                {
                    "horizon_years": horizon,
                    "percentile": percentile,
                    "nominal_value": round(nominal_value, 2),
                    "real_value": round(nominal_value / real_discount, 2),
                }
            )
        probability = float(np.mean(nominal >= target_wealth)) if target_wealth > 0 else 0.0
        goal_rows.append(
            {
                "horizon_years": horizon,
                "target_wealth": round(float(target_wealth), 2),
                "probability": round(probability, 6),
            }
        )

    return {
        "percentiles": percentile_rows,
        "goal_probabilities": goal_rows,
        "metadata": {
            "seed": seed,
            "scenario_count": scenario_count,
            "horizons": horizons,
            "percentiles": percentiles,
        },
    }

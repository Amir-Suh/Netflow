"""Modern Portfolio Theory optimizer."""

from __future__ import annotations

import numpy as np
from scipy.optimize import minimize


def portfolio_metrics(weights: np.ndarray, expected_returns: np.ndarray, covariance: np.ndarray, risk_free_rate: float = 0.04) -> dict:
    ret = float(weights @ expected_returns)
    vol = float(np.sqrt(max(weights @ covariance @ weights, 0.0)))
    sharpe = float((ret - risk_free_rate) / vol) if vol > 0 else 0.0
    return {"expected_return": ret, "volatility": vol, "sharpe": sharpe}


def optimize_portfolio(
    asset_classes: list[str],
    current_weights: np.ndarray,
    expected_returns: np.ndarray,
    covariance: np.ndarray,
    risk_tolerance: str = "moderate",
    risk_free_rate: float = 0.04,
    max_weight: float = 0.85,
    frontier_points: int = 15,
) -> dict:
    n = len(asset_classes)
    if n == 0:
        raise ValueError("at least one asset class is required")
    current_weights = np.asarray(current_weights, dtype=float)
    expected_returns = np.asarray(expected_returns, dtype=float)
    covariance = np.asarray(covariance, dtype=float)
    if current_weights.sum() <= 0:
        raise ValueError("current weights must have positive sum")
    current_weights = current_weights / current_weights.sum()

    bounds = [(0.0, max_weight) for _ in range(n)]
    constraints = [{"type": "eq", "fun": lambda w: np.sum(w) - 1.0}]
    x0 = np.repeat(1.0 / n, n)

    min_var = _minimize(lambda w: float(w @ covariance @ w), x0, bounds, constraints)
    max_sharpe = _minimize(
        lambda w: -portfolio_metrics(w, expected_returns, covariance, risk_free_rate)["sharpe"],
        x0,
        bounds,
        constraints,
    )
    blend = {"conservative": 0.25, "moderate": 0.50, "aggressive": 0.80}.get(risk_tolerance, 0.50)
    recommended = _project_long_only((1.0 - blend) * min_var + blend * max_sharpe, max_weight)

    frontier = _frontier(asset_classes, expected_returns, covariance, risk_free_rate, bounds, x0, frontier_points)
    return {
        "current": _snapshot(asset_classes, current_weights, expected_returns, covariance, risk_free_rate),
        "min_variance": _snapshot(asset_classes, min_var, expected_returns, covariance, risk_free_rate),
        "max_sharpe": _snapshot(asset_classes, max_sharpe, expected_returns, covariance, risk_free_rate),
        "recommended": _snapshot(asset_classes, recommended, expected_returns, covariance, risk_free_rate),
        "frontier": frontier,
    }


def _minimize(objective, x0, bounds, constraints) -> np.ndarray:
    result = minimize(
        objective,
        x0,
        method="SLSQP",
        bounds=bounds,
        constraints=constraints,
        options={"maxiter": 500, "ftol": 1e-10, "disp": False},
    )
    if not result.success:
        return np.asarray(x0, dtype=float)
    return _project_long_only(np.asarray(result.x, dtype=float), max_weight=max(bound[1] for bound in bounds))


def _project_long_only(weights: np.ndarray, max_weight: float) -> np.ndarray:
    weights = np.clip(weights, 0.0, max_weight)
    if weights.sum() <= 0:
        weights = np.repeat(1.0 / len(weights), len(weights))
    weights = weights / weights.sum()
    for _ in range(10):
        over = weights > max_weight
        if not over.any():
            break
        excess = float(np.sum(weights[over] - max_weight))
        weights[over] = max_weight
        under = ~over
        if under.any():
            weights[under] += excess * weights[under] / weights[under].sum()
    return weights / weights.sum()


def _snapshot(asset_classes, weights, expected_returns, covariance, risk_free_rate) -> dict:
    metrics = portfolio_metrics(weights, expected_returns, covariance, risk_free_rate)
    metrics["weights"] = {asset: round(float(weights[idx]), 6) for idx, asset in enumerate(asset_classes)}
    return metrics


def _frontier(asset_classes, expected_returns, covariance, risk_free_rate, bounds, x0, point_count) -> list[dict]:
    min_ret = float(np.min(expected_returns))
    max_ret = float(np.max(expected_returns))
    targets = np.linspace(min_ret, max_ret, point_count)
    points = []
    for idx, target in enumerate(targets):
        constraints = [
            {"type": "eq", "fun": lambda w: np.sum(w) - 1.0},
            {"type": "eq", "fun": lambda w, target=target: float(w @ expected_returns) - target},
        ]
        weights = _minimize(lambda w: float(w @ covariance @ w), x0, bounds, constraints)
        snapshot = _snapshot(asset_classes, weights, expected_returns, covariance, risk_free_rate)
        points.append(
            {
                "point_index": idx,
                "expected_return": snapshot["expected_return"],
                "volatility": snapshot["volatility"],
                "sharpe": snapshot["sharpe"],
                "weights": snapshot["weights"],
            }
        )
    return points

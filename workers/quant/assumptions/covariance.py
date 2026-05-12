"""Covariance helpers.

Raw sample covariance can be unstable with short histories because each noisy
monthly observation receives full weight. Shrinkage dampens that noise by
pulling the covariance matrix toward a simpler target.
"""

from __future__ import annotations

import numpy as np


def sample_covariance(monthly_returns: np.ndarray) -> np.ndarray:
    if monthly_returns.ndim != 2 or monthly_returns.shape[0] < 2:
        raise ValueError("monthly_returns must have at least two observations")
    return np.cov(monthly_returns, rowvar=False) * 12.0


def ledoit_wolf_covariance(monthly_returns: np.ndarray) -> tuple[np.ndarray, str]:
    try:
        from sklearn.covariance import LedoitWolf

        model = LedoitWolf().fit(monthly_returns)
        return model.covariance_ * 12.0, "ledoit_wolf"
    except Exception:  # noqa: BLE001
        return diagonal_shrinkage(monthly_returns), "diagonal_shrinkage_fallback"


def diagonal_shrinkage(monthly_returns: np.ndarray, alpha: float = 0.35) -> np.ndarray:
    raw = sample_covariance(monthly_returns)
    target = np.diag(np.diag(raw))
    return (1.0 - alpha) * raw + alpha * target


def covariance_from_assumptions(assumptions: dict, asset_classes: list[str]) -> np.ndarray:
    corr = assumptions.get("correlation", {})
    vols = np.array([
        assumptions["asset_classes"][asset]["volatility"]
        for asset in asset_classes
    ], dtype=float)
    matrix = np.eye(len(asset_classes), dtype=float)
    for i, left in enumerate(asset_classes):
        for j, right in enumerate(asset_classes):
            matrix[i, j] = float(corr.get(left, {}).get(right, 1.0 if i == j else 0.0))
    return np.outer(vols, vols) * matrix

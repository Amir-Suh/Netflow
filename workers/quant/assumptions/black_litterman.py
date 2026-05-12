"""Small Black-Litterman implementation for demo-safe quant architecture."""

from __future__ import annotations

import numpy as np

from .demo import ASSET_CLASSES, DEMO_GLOBALS, demo_assumptions
from .covariance import covariance_from_assumptions


def black_litterman_returns(
    covariance: np.ndarray,
    market_weights: np.ndarray,
    risk_aversion: float = 2.5,
    tau: float = 0.05,
    p: np.ndarray | None = None,
    q: np.ndarray | None = None,
    omega: np.ndarray | None = None,
) -> np.ndarray:
    pi = risk_aversion * covariance @ market_weights
    if p is None or q is None:
        return pi
    if omega is None:
        omega = np.diag(np.diag(tau * p @ covariance @ p.T))
    inv_tau_cov = np.linalg.inv(tau * covariance)
    middle = inv_tau_cov + p.T @ np.linalg.inv(omega) @ p
    right = inv_tau_cov @ pi + p.T @ np.linalg.inv(omega) @ q
    return np.linalg.solve(middle, right)


def estimate_black_litterman(asset_classes: list[str] | None = None) -> tuple[dict, np.ndarray]:
    asset_classes = asset_classes or list(ASSET_CLASSES)
    base = demo_assumptions()
    covariance = covariance_from_assumptions(base, asset_classes)
    market_weights = np.array([0.05, 0.25, 0.45, 0.25], dtype=float)
    market_weights = market_weights[: len(asset_classes)]
    market_weights = market_weights / market_weights.sum()
    returns = black_litterman_returns(
        covariance=covariance,
        market_weights=market_weights,
        risk_aversion=2.5,
        tau=0.05,
    )
    assumptions = {
        **base,
        "method": "black_litterman",
        "asset_classes": {
            asset: {
                "expected_return": float(returns[idx]),
                "volatility": base["asset_classes"][asset]["volatility"],
            }
            for idx, asset in enumerate(asset_classes)
        },
        "globals": {
            **base["globals"],
            "risk_aversion": 2.5,
            "tau": 0.05,
            "market_weights": {asset: float(market_weights[idx]) for idx, asset in enumerate(asset_classes)},
        },
        "implementation_status": "partial_equilibrium_no_view_demo",
        "disclaimer": "Simplified Black-Litterman equilibrium estimate; not financial advice.",
    }
    return assumptions, covariance

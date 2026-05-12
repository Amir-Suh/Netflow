"""CAPM expected return estimator."""

from __future__ import annotations

import numpy as np

from .demo import ASSET_CLASSES, DEMO_GLOBALS, DEMO_STATIC_ASSUMPTIONS, demo_assumptions
from .covariance import covariance_from_assumptions


def estimate_capm(
    asset_classes: list[str] | None = None,
    risk_free_rate: float | None = None,
    market_risk_premium: float | None = None,
) -> tuple[dict, np.ndarray]:
    asset_classes = asset_classes or list(ASSET_CLASSES)
    risk_free_rate = DEMO_GLOBALS["risk_free_rate"] if risk_free_rate is None else risk_free_rate
    market_risk_premium = DEMO_GLOBALS["market_risk_premium"] if market_risk_premium is None else market_risk_premium
    base = demo_assumptions()
    assets = {}
    for asset in asset_classes:
        beta = DEMO_STATIC_ASSUMPTIONS[asset]["beta"]
        assets[asset] = {
            "expected_return": float(risk_free_rate + beta * market_risk_premium),
            "volatility": DEMO_STATIC_ASSUMPTIONS[asset]["volatility"],
            "beta": beta,
        }
    assumptions = {
        **base,
        "method": "capm",
        "asset_classes": assets,
        "globals": {
            **base["globals"],
            "risk_free_rate": risk_free_rate,
            "market_risk_premium": market_risk_premium,
        },
        "disclaimer": "CAPM output is an assumption estimate, not a guarantee or financial advice.",
    }
    return assumptions, covariance_from_assumptions(assumptions, asset_classes)

"""Deterministic educational assumptions for local demos and tests."""

from __future__ import annotations

ASSET_CLASSES = ["cash", "bonds", "us_equity", "international_equity"]

DEMO_STATIC_ASSUMPTIONS = {
    "cash": {"expected_return": 0.015, "volatility": 0.010, "beta": 0.0},
    "bonds": {"expected_return": 0.035, "volatility": 0.050, "beta": 0.2},
    "us_equity": {"expected_return": 0.070, "volatility": 0.150, "beta": 1.0},
    "international_equity": {"expected_return": 0.065, "volatility": 0.170, "beta": 1.05},
}

DEMO_CORRELATION = {
    "cash": {"cash": 1.00, "bonds": 0.05, "us_equity": 0.02, "international_equity": 0.02},
    "bonds": {"cash": 0.05, "bonds": 1.00, "us_equity": 0.20, "international_equity": 0.18},
    "us_equity": {"cash": 0.02, "bonds": 0.20, "us_equity": 1.00, "international_equity": 0.82},
    "international_equity": {"cash": 0.02, "bonds": 0.18, "us_equity": 0.82, "international_equity": 1.00},
}

DEMO_GLOBALS = {
    "inflation": 0.025,
    "risk_free_rate": 0.040,
    "market_risk_premium": 0.050,
    "scenario_count": 10000,
    "horizons": [10, 20, 30],
    "percentiles": [10, 50, 90],
}


def demo_assumptions() -> dict:
    assets = {
        asset_class: dict(values)
        for asset_class, values in DEMO_STATIC_ASSUMPTIONS.items()
    }
    return {
        "method": "demo_static",
        "asset_classes": assets,
        "correlation": DEMO_CORRELATION,
        "globals": dict(DEMO_GLOBALS),
        "disclaimer": "Educational projection assumptions; not financial advice.",
    }

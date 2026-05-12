"""Historical estimator backed by local deterministic demo fixtures."""

from __future__ import annotations

import csv
from pathlib import Path

import numpy as np

from .demo import ASSET_CLASSES, DEMO_GLOBALS
from .covariance import ledoit_wolf_covariance, sample_covariance


FIXTURE_PATH = Path(__file__).resolve().parents[1] / "fixtures" / "asset_class_monthly_returns_demo.csv"


def load_fixture_returns(path: Path = FIXTURE_PATH) -> dict[str, list[float]]:
    series = {asset: [] for asset in ASSET_CLASSES}
    with path.open(newline="", encoding="utf-8") as fh:
        reader = csv.DictReader(fh)
        for row in reader:
            asset = row["asset_class"]
            if asset in series:
                series[asset].append(float(row["return_value"]))
    return series


def matrix_from_series(series: dict[str, list[float]], asset_classes: list[str]) -> np.ndarray:
    min_len = min(len(series.get(asset, [])) for asset in asset_classes)
    if min_len < 2:
        raise ValueError("historical return fixture needs at least two rows per asset class")
    return np.array([
        series[asset][-min_len:]
        for asset in asset_classes
    ], dtype=float).T


def estimate_historical(asset_classes: list[str] | None = None, shrinkage: bool = False) -> tuple[dict, np.ndarray]:
    asset_classes = asset_classes or list(ASSET_CLASSES)
    returns = matrix_from_series(load_fixture_returns(), asset_classes)
    annual_mean = np.mean(returns, axis=0) * 12.0
    annual_vol = np.std(returns, axis=0, ddof=1) * np.sqrt(12.0)
    if shrinkage:
        cov, method = ledoit_wolf_covariance(returns)
    else:
        cov = sample_covariance(returns)
        method = "sample_covariance"
    corr = np.corrcoef(returns, rowvar=False)
    assumptions = {
        "method": "historical_with_shrinkage" if shrinkage else "historical_sample",
        "asset_classes": {
            asset: {
                "expected_return": float(annual_mean[idx]),
                "volatility": float(annual_vol[idx]),
            }
            for idx, asset in enumerate(asset_classes)
        },
        "correlation": {
            left: {right: float(corr[i, j]) for j, right in enumerate(asset_classes)}
            for i, left in enumerate(asset_classes)
        },
        "globals": dict(DEMO_GLOBALS),
        "covariance_method": method,
        "source": "demo_fixture",
        "disclaimer": "Synthetic local historical fixture; not live market data and not financial advice.",
    }
    return assumptions, cov

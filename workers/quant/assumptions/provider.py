"""Assumption method selection facade."""

from __future__ import annotations

from .black_litterman import estimate_black_litterman
from .capm import estimate_capm
from .covariance import covariance_from_assumptions
from .demo import ASSET_CLASSES, demo_assumptions
from .historical import estimate_historical


SUPPORTED_METHODS = {
    "demo_static",
    "historical_sample",
    "capm",
    "black_litterman",
    "historical_with_shrinkage",
}


def get_assumptions(method: str = "demo_static", asset_classes: list[str] | None = None):
    asset_classes = asset_classes or list(ASSET_CLASSES)
    method = method if method in SUPPORTED_METHODS else "demo_static"
    if method == "historical_sample":
        return estimate_historical(asset_classes, shrinkage=False)
    if method == "historical_with_shrinkage":
        return estimate_historical(asset_classes, shrinkage=True)
    if method == "capm":
        return estimate_capm(asset_classes)
    if method == "black_litterman":
        return estimate_black_litterman(asset_classes)
    assumptions = demo_assumptions()
    return assumptions, covariance_from_assumptions(assumptions, asset_classes)

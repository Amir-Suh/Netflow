"""Deterministic asset-class mapping for demo holdings."""

from __future__ import annotations


def map_holding(symbol: str, name: str = "", provided: str = "") -> str:
    value = (provided or "").strip().lower()
    if value in {"cash", "bonds", "us_equity", "international_equity"}:
        return value

    symbol = (symbol or "").strip().upper()
    name = (name or "").strip().lower()
    if symbol in {"VTI", "VOO", "SPY"}:
        return "us_equity"
    if symbol in {"VXUS", "IXUS"}:
        return "international_equity"
    if symbol in {"BND", "AGG"}:
        return "bonds"
    if symbol in {"CASH", "CHECKING", "SAVINGS"}:
        return "cash"
    if "international" in name:
        return "international_equity"
    if "bond" in name:
        return "bonds"
    if "cash" in name or "checking" in name or "savings" in name:
        return "cash"
    if "equity" in name or "stock" in name or "market" in name:
        return "us_equity"
    return "cash"

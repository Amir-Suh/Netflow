"""Mapping of merchant patterns to scrape targets.

Each entry describes a deterministic Playwright probe: a public pricing
or comparison page, a CSS selector that returns the headline price, a
human label for the alert, and a per-month rate hint used when a page
quotes annual or weekly amounts.
"""

from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class ProviderTarget:
    name: str
    url: str
    selector: str
    label: str
    monthly: bool = True


# These URLs are public marketing/pricing pages that work in headless
# Chromium without authentication. The selectors are deliberately broad
# so small DOM changes don't break the demo.
PROVIDER_SCRAPE_MAP: dict[str, ProviderTarget] = {
    "netflix": ProviderTarget(
        name="netflix",
        url="https://www.netflix.com/signup/planform",
        selector='[data-uia*="price"], .price, .planGrid__planPrice',
        label="Netflix Standard",
    ),
    "spotify": ProviderTarget(
        name="spotify",
        url="https://www.spotify.com/us/premium/",
        selector='[data-testid="premium-plan-card-price"], .price',
        label="Spotify Individual",
    ),
    "hulu": ProviderTarget(
        name="hulu",
        url="https://www.hulu.com/welcome",
        selector='[data-testid="plan-price"], .plan-price',
        label="Hulu (with ads)",
    ),
    "disney": ProviderTarget(
        name="disney",
        url="https://www.disneyplus.com/welcome/disney-plus",
        selector='[data-testid="price"], .price',
        label="Disney+ Basic",
    ),
    "comcast": ProviderTarget(
        name="comcast",
        url="https://www.broadbandnow.com/Comcast",
        selector='.plan-price, .price-cell, [class*="price"]',
        label="Comcast Internet (entry)",
    ),
    "xfinity": ProviderTarget(
        name="xfinity",
        url="https://www.broadbandnow.com/Comcast",
        selector='.plan-price, .price-cell, [class*="price"]',
        label="Xfinity Internet (entry)",
    ),
    "verizon": ProviderTarget(
        name="verizon",
        url="https://www.broadbandnow.com/Verizon-Fios",
        selector='.plan-price, .price-cell, [class*="price"]',
        label="Verizon Fios (entry)",
    ),
    "att": ProviderTarget(
        name="att",
        url="https://www.broadbandnow.com/AT-T",
        selector='.plan-price, .price-cell, [class*="price"]',
        label="AT&T Internet (entry)",
    ),
    "spectrum": ProviderTarget(
        name="spectrum",
        url="https://www.broadbandnow.com/Spectrum",
        selector='.plan-price, .price-cell, [class*="price"]',
        label="Spectrum Internet (entry)",
    ),
}


def find_provider(merchant: str) -> ProviderTarget | None:
    if not merchant:
        return None
    needle = merchant.lower()
    for key, provider in PROVIDER_SCRAPE_MAP.items():
        if key in needle:
            return provider
    return None

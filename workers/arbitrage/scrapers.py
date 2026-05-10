"""Playwright headless-Chromium scraping helpers."""

from __future__ import annotations

import logging
import re
from typing import Optional

from playwright.sync_api import sync_playwright, TimeoutError as PlaywrightTimeoutError

from providers import ProviderTarget

log = logging.getLogger("arbitrage.scrapers")

_PRICE_REGEX = re.compile(r"\$\s*([0-9]+(?:\.[0-9]{1,2})?)")


def _extract_price(text: str) -> Optional[float]:
    if not text:
        return None
    match = _PRICE_REGEX.search(text)
    if not match:
        return None
    try:
        return float(match.group(1))
    except ValueError:
        return None


def fetch_market_rate(provider: ProviderTarget) -> Optional[float]:
    """Return the lowest price observed at the provider's pricing page.

    Returns None if the page doesn't load or no price element is found.
    The headless browser is launched fresh per call to keep the worker
    stateless and resistant to memory leaks.
    """

    log.info("scraping %s at %s", provider.name, provider.url)
    try:
        with sync_playwright() as p:
            browser = p.chromium.launch(headless=True, args=["--no-sandbox"])
            try:
                context = browser.new_context(
                    user_agent=(
                        "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36"
                        " (KHTML, like Gecko) Chrome/124.0 Safari/537.36"
                    ),
                    viewport={"width": 1280, "height": 800},
                )
                page = context.new_page()
                page.set_default_timeout(20_000)
                try:
                    page.goto(provider.url, wait_until="domcontentloaded")
                except PlaywrightTimeoutError:
                    log.warning("timeout loading %s", provider.url)
                    return None

                prices: list[float] = []
                try:
                    elements = page.locator(provider.selector)
                    count = min(elements.count(), 10)
                    for idx in range(count):
                        text = elements.nth(idx).inner_text(timeout=3_000)
                        price = _extract_price(text)
                        if price is not None:
                            prices.append(price)
                except PlaywrightTimeoutError:
                    pass

                if not prices:
                    body_text = page.inner_text("body", timeout=5_000)
                    price = _extract_price(body_text)
                    if price is not None:
                        prices.append(price)

                if not prices:
                    return None
                return min(prices)
            finally:
                browser.close()
    except Exception as exc:  # noqa: BLE001
        log.exception("playwright failed for %s: %s", provider.name, exc)
        return None

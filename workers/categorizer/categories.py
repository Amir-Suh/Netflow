"""Keyword-to-category mapping used for fuzzy matching.

Categories are aligned with the Cycle 2 spec ("DD DOORDASH" -> Dining,
recurring utilities/subscriptions surface to the arbitrage worker).
The matcher prefers an exact substring hit; rapidfuzz handles the
fuzzier merchant variants (e.g. "DD *DOORDASH 4422").
"""

from __future__ import annotations

CATEGORY_KEYWORDS: dict[str, list[str]] = {
    "Dining": [
        "doordash", "dd doordash", "uber eats", "ubereats", "grubhub", "postmates",
        "chipotle", "starbucks", "mcdonalds", "mcdonald's", "burger king", "wendys",
        "subway", "dominos", "domino's", "pizza hut", "panera", "chick-fil-a",
        "taco bell", "kfc", "dunkin", "cafe", "restaurant", "bistro", "diner",
    ],
    "Groceries": [
        "whole foods", "trader joe", "safeway", "kroger", "publix", "aldi",
        "wegmans", "costco", "sams club", "sam's club", "walmart grocery",
        "instacart", "fresh market", "sprouts", "h-e-b", "heb", "stop & shop",
    ],
    "Transportation": [
        "uber", "lyft", "shell", "exxon", "chevron", "bp", "mobil", "speedway",
        "metro", "mta", "bart", "amtrak", "delta", "united", "southwest",
        "american airlines", "jetblue", "hertz", "avis", "enterprise", "parking",
    ],
    "Entertainment": [
        "amc", "regal", "cinemark", "ticketmaster", "stubhub", "live nation",
        "concert", "theatre", "theater", "fandango",
    ],
    "Subscriptions": [
        "netflix", "hulu", "disney+", "disney plus", "spotify", "apple music",
        "youtube premium", "youtube tv", "hbo max", "max ", "paramount+",
        "peacock", "patreon", "audible", "kindle unlimited", "github",
        "openai", "chatgpt", "notion", "1password", "dropbox", "icloud",
        "google one", "google storage", "adobe", "microsoft 365", "office 365",
    ],
    "Utilities": [
        "comcast", "xfinity", "verizon", "at&t", "att ", "t-mobile", "tmobile",
        "spectrum", "cox communications", "centurylink", "duke energy",
        "pg&e", "pge ", "con edison", "consolidated edison", "national grid",
        "water bill", "utility", "electric company", "gas company",
    ],
    "Healthcare": [
        "cvs", "walgreens", "rite aid", "kaiser", "blue cross", "aetna",
        "united healthcare", "humana", "pharmacy", "hospital", "clinic",
        "dental", "dentist", "optometry",
    ],
    "Shopping": [
        "amazon", "amzn", "target", "walmart", "best buy", "ebay", "etsy",
        "macys", "macy's", "nordstrom", "kohls", "kohl's", "home depot",
        "lowes", "lowe's", "ikea", "wayfair", "shein",
    ],
    "Income": [
        "payroll", "direct deposit", "salary", "wages", "bonus", "stripe payout",
        "venmo cashout", "cash app deposit",
    ],
    "Transfer": [
        "transfer", "zelle", "venmo", "cash app", "paypal", "wire", "ach",
    ],
}


def category_corpus() -> list[tuple[str, str]]:
    """Returns flat list of (keyword, category) pairs for fuzzy matching."""
    pairs: list[tuple[str, str]] = []
    for category, keywords in CATEGORY_KEYWORDS.items():
        for keyword in keywords:
            pairs.append((keyword, category))
    return pairs

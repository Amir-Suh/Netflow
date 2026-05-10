"""Celery task: scrape competing market rates for recurring services."""

from __future__ import annotations

import datetime as dt
import json
import logging
import os
from typing import Optional

import pika
import psycopg2

from app import celery_app
from providers import find_provider
from scrapers import fetch_market_rate

log = logging.getLogger(__name__)


def _exchange() -> str:
    return os.environ.get("RABBITMQ_EXCHANGE", "netflow.events")


def _rabbit_url() -> str:
    return os.environ.get("RABBITMQ_URL", "amqp://netflow:netflow@rabbitmq:5672/")


def _db_url() -> str:
    return os.environ.get("DATABASE_URL", "")


def _publish(routing_key: str, body: dict) -> None:
    conn = pika.BlockingConnection(pika.URLParameters(_rabbit_url()))
    try:
        ch = conn.channel()
        ch.exchange_declare(exchange=_exchange(), exchange_type="topic", durable=True)
        ch.basic_publish(
            exchange=_exchange(),
            routing_key=routing_key,
            body=json.dumps(body, default=str).encode("utf-8"),
            properties=pika.BasicProperties(content_type="application/json", delivery_mode=2),
        )
    finally:
        conn.close()


def _store_opportunity(
    user_id: int,
    transaction_id: int,
    merchant: str,
    category: str,
    current_amount: float,
    market_rate: Optional[float],
    savings: Optional[float],
    provider_url: str,
) -> None:
    conn = psycopg2.connect(_db_url())
    try:
        with conn.cursor() as cur:
            cur.execute(
                """
                INSERT INTO arbitrage_opportunities (
                    user_id, transaction_id, merchant_name, category,
                    current_amount, market_rate, savings_estimate, provider_url
                ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
                """,
                (
                    user_id,
                    transaction_id,
                    merchant,
                    category,
                    current_amount,
                    market_rate,
                    savings,
                    provider_url,
                ),
            )
            conn.commit()
    finally:
        conn.close()


@celery_app.task(name="check_arbitrage", bind=True, max_retries=2, default_retry_delay=30)
def check_arbitrage(
    self,
    transaction_id: int,
    user_id: int,
    merchant_name: str,
    category: str,
    current_amount: float,
) -> dict:
    log.info(
        "arbitrage check tx=%s user=%s merchant=%s category=%s amount=%s",
        transaction_id,
        user_id,
        merchant_name,
        category,
        current_amount,
    )

    provider = find_provider(merchant_name)
    occurred_at = dt.datetime.now(dt.timezone.utc).isoformat()

    payload = {
        "event_type": "transactions.arbitrage.results",
        "transaction_id": transaction_id,
        "user_id": user_id,
        "merchant_name": merchant_name,
        "category": category,
        "current_amount": float(current_amount),
        "occurred_at": occurred_at,
    }

    if provider is None:
        log.info("no scrape target for merchant %s", merchant_name)
        payload["status"] = "no_provider"
        _publish("transactions.arbitrage.results", payload)
        return payload

    market_rate = fetch_market_rate(provider)
    if market_rate is None:
        log.info("market rate not extracted for %s", provider.name)
        payload["status"] = "rate_unavailable"
        payload["provider_url"] = provider.url
        _publish("transactions.arbitrage.results", payload)
        return payload

    savings: Optional[float] = None
    if market_rate < float(current_amount):
        savings = round(float(current_amount) - market_rate, 2)

    _store_opportunity(
        user_id=user_id,
        transaction_id=transaction_id,
        merchant=merchant_name,
        category=category,
        current_amount=float(current_amount),
        market_rate=market_rate,
        savings=savings,
        provider_url=provider.url,
    )

    payload.update(
        {
            "status": "savings_found" if savings else "no_savings",
            "market_rate": market_rate,
            "savings_estimate": savings,
            "provider_url": provider.url,
            "provider_label": provider.label,
        }
    )
    _publish("transactions.arbitrage.results", payload)
    return payload

"""RabbitMQ publisher for quant completion events."""

from __future__ import annotations

import datetime as dt
import json
import os

import pika


def _exchange() -> str:
    return os.environ.get("RABBITMQ_EXCHANGE", "netflow.events")


def _rabbit_url() -> str:
    return os.environ.get("RABBITMQ_URL", "amqp://netflow:netflow@rabbitmq:5672/")


def publish(routing_key: str, body: dict) -> None:
    body.setdefault("event_type", routing_key)
    body.setdefault("occurred_at", dt.datetime.now(dt.timezone.utc).isoformat())
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

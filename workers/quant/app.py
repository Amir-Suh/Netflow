"""Celery application for Cycle 3 quant jobs."""

from __future__ import annotations

import os

from celery import Celery


def _broker_url() -> str:
    return os.environ.get("RABBITMQ_URL", "amqp://netflow:netflow@rabbitmq:5672/")


celery_app = Celery(
    "netflow_quant",
    broker=_broker_url(),
    backend=None,
    include=["tasks"],
)

celery_app.conf.update(
    task_acks_late=True,
    task_default_queue="netflow.quant",
    task_track_started=True,
    worker_prefetch_multiplier=1,
    broker_connection_retry_on_startup=True,
    task_time_limit=300,
)

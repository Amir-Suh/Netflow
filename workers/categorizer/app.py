"""Celery application for the NLP categorization worker.

The Go API publishes ``transactions.ingested`` events to the
``netflow.events`` topic exchange. A Kombu consumer in worker.py reads
those messages and dispatches the ``categorize_transaction`` Celery task
defined in tasks.py. Celery itself uses the same RabbitMQ broker.
"""

from __future__ import annotations

import os

from celery import Celery


def _broker_url() -> str:
    return os.environ.get(
        "RABBITMQ_URL",
        "amqp://netflow:netflow@rabbitmq:5672/",
    )


celery_app = Celery(
    "netflow_categorizer",
    broker=_broker_url(),
    backend=None,
    include=["tasks"],
)

celery_app.conf.update(
    task_acks_late=True,
    task_default_queue="netflow.categorizer",
    task_track_started=True,
    worker_prefetch_multiplier=4,
    broker_connection_retry_on_startup=True,
)

"""Bridge between the Go publisher's ``transactions.ingested`` queue and Celery.

The Go API publishes raw transaction ingest events to the
``netflow.events`` topic exchange with routing key
``transactions.ingested``. Celery has its own routing semantics, so we
spin up a dedicated Kombu consumer that reads each ingest event and
dispatches a ``categorize_transaction`` Celery task. The task itself
runs in the Celery worker pool started alongside this loop.
"""

from __future__ import annotations

import json
import logging
import os
import signal
import sys
import threading

from kombu import Connection, Exchange, Queue

from app import celery_app
from tasks import categorize_transaction

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s :: %(message)s")
log = logging.getLogger("categorizer.worker")


_EXCHANGE_NAME = os.environ.get("RABBITMQ_EXCHANGE", "netflow.events")
_INGEST_QUEUE_NAME = os.environ.get("RABBITMQ_QUEUE", "transactions.ingested")
_RABBITMQ_URL = os.environ.get("RABBITMQ_URL", "amqp://netflow:netflow@rabbitmq:5672/")


def _on_message(body, message) -> None:
    try:
        if isinstance(body, (bytes, bytearray)):
            payload = json.loads(body.decode("utf-8"))
        elif isinstance(body, str):
            payload = json.loads(body)
        else:
            payload = body
        transaction_id = int(payload.get("transaction_id", 0))
        user_id = int(payload.get("user_id", 0))
        if not transaction_id or not user_id:
            log.warning("ingest event missing ids: %s", payload)
            message.ack()
            return
        categorize_transaction.delay(transaction_id, user_id)
        log.info("dispatched categorize_transaction tx=%s user=%s", transaction_id, user_id)
        message.ack()
    except Exception:  # noqa: BLE001
        log.exception("failed to dispatch categorize task")
        message.reject(requeue=True)


def consume_forever(stop_event: threading.Event) -> None:
    exchange = Exchange(_EXCHANGE_NAME, type="topic", durable=True)
    queue = Queue(
        _INGEST_QUEUE_NAME,
        exchange=exchange,
        routing_key="transactions.ingested",
        durable=True,
    )
    while not stop_event.is_set():
        try:
            with Connection(_RABBITMQ_URL, heartbeat=30) as conn:
                with conn.Consumer([queue], callbacks=[_on_message], accept=["json", "application/json"]):
                    log.info("listening on %s", _INGEST_QUEUE_NAME)
                    while not stop_event.is_set():
                        try:
                            conn.drain_events(timeout=2)
                        except TimeoutError:
                            continue
        except Exception:  # noqa: BLE001
            log.exception("kombu consumer crashed; reconnecting in 3s")
            stop_event.wait(3)


def _start_celery_worker(stop_event: threading.Event) -> threading.Thread:
    """Run a Celery worker in-process so this single container handles both
    the Kombu bridge and task execution. Suitable for the local sandbox."""

    def _runner() -> None:
        argv = ["worker", "--loglevel=info", "-Q", "netflow.categorizer", "--concurrency=2"]
        try:
            celery_app.worker_main(argv=argv)
        finally:
            stop_event.set()

    thread = threading.Thread(target=_runner, name="celery-worker", daemon=True)
    thread.start()
    return thread


def main() -> int:
    stop_event = threading.Event()

    def _shutdown(signum, _frame):  # noqa: ARG001
        log.info("received signal %s, shutting down", signum)
        stop_event.set()

    signal.signal(signal.SIGINT, _shutdown)
    signal.signal(signal.SIGTERM, _shutdown)

    _start_celery_worker(stop_event)
    consume_forever(stop_event)
    return 0


if __name__ == "__main__":
    sys.exit(main())

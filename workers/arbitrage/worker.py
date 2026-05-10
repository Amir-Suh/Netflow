"""Bridge ``transactions.arbitrage`` events to the Celery check_arbitrage task."""

from __future__ import annotations

import json
import logging
import os
import signal
import sys
import threading

from kombu import Connection, Exchange, Queue

from app import celery_app
from tasks import check_arbitrage

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s :: %(message)s")
log = logging.getLogger("arbitrage.worker")

_EXCHANGE_NAME = os.environ.get("RABBITMQ_EXCHANGE", "netflow.events")
_QUEUE_NAME = os.environ.get("ARBITRAGE_QUEUE", "transactions.arbitrage")
_RABBITMQ_URL = os.environ.get("RABBITMQ_URL", "amqp://netflow:netflow@rabbitmq:5672/")


def _on_message(body, message) -> None:
    try:
        payload = json.loads(body.decode("utf-8")) if isinstance(body, (bytes, bytearray)) else (
            json.loads(body) if isinstance(body, str) else body
        )
        check_arbitrage.delay(
            int(payload.get("transaction_id", 0)),
            int(payload.get("user_id", 0)),
            str(payload.get("merchant_name", "")),
            str(payload.get("category", "")),
            float(payload.get("current_amount", 0.0)),
        )
        log.info("dispatched check_arbitrage for tx=%s", payload.get("transaction_id"))
        message.ack()
    except Exception:  # noqa: BLE001
        log.exception("failed to dispatch check_arbitrage")
        message.reject(requeue=True)


def consume_forever(stop_event: threading.Event) -> None:
    exchange = Exchange(_EXCHANGE_NAME, type="topic", durable=True)
    queue = Queue(
        _QUEUE_NAME,
        exchange=exchange,
        routing_key="transactions.arbitrage",
        durable=True,
    )
    while not stop_event.is_set():
        try:
            with Connection(_RABBITMQ_URL, heartbeat=30) as conn:
                with conn.Consumer([queue], callbacks=[_on_message], accept=["json", "application/json"]):
                    log.info("listening on %s", _QUEUE_NAME)
                    while not stop_event.is_set():
                        try:
                            conn.drain_events(timeout=2)
                        except TimeoutError:
                            continue
        except Exception:  # noqa: BLE001
            log.exception("kombu consumer crashed; reconnecting in 3s")
            stop_event.wait(3)


def _start_celery_worker(stop_event: threading.Event) -> threading.Thread:
    def _runner() -> None:
        argv = ["worker", "--loglevel=info", "-Q", "netflow.arbitrage", "--concurrency=1"]
        try:
            celery_app.worker_main(argv=argv)
        finally:
            stop_event.set()

    thread = threading.Thread(target=_runner, name="celery-arbitrage", daemon=True)
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

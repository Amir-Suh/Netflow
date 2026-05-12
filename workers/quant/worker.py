"""Bridge quant request routing keys to Celery tasks."""

from __future__ import annotations

import json
import logging
import os
import signal
import sys
import threading

from kombu import Connection, Exchange, Queue

from app import celery_app
from tasks import optimize_debt, optimize_portfolio, simulate_wealth

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s :: %(message)s")
log = logging.getLogger("quant.worker")

_EXCHANGE_NAME = os.environ.get("RABBITMQ_EXCHANGE", "netflow.events")
_RABBITMQ_URL = os.environ.get("RABBITMQ_URL", "amqp://netflow:netflow@rabbitmq:5672/")


def _decode(body):
    if isinstance(body, (bytes, bytearray)):
        return json.loads(body.decode("utf-8"))
    if isinstance(body, str):
        return json.loads(body)
    return body


def _on_message(body, message) -> None:
    try:
        payload = _decode(body)
        run_id = int(payload.get("run_id", 0))
        user_id = int(payload.get("user_id", 0))
        job_type = str(payload.get("job_type", ""))
        if not run_id or not user_id:
            log.warning("quant event missing ids: %s", payload)
            message.ack()
            return
        if job_type == "debt":
            optimize_debt.delay(run_id, user_id)
        elif job_type == "wealth":
            simulate_wealth.delay(run_id, user_id)
        elif job_type == "portfolio":
            optimize_portfolio.delay(run_id, user_id)
        else:
            log.warning("unknown quant job type: %s", payload)
            message.ack()
            return
        log.info("dispatched quant %s run=%s user=%s", job_type, run_id, user_id)
        message.ack()
    except Exception:  # noqa: BLE001
        log.exception("failed to dispatch quant task")
        message.reject(requeue=True)


def consume_forever(stop_event: threading.Event) -> None:
    exchange = Exchange(_EXCHANGE_NAME, type="topic", durable=True)
    queues = [
        Queue("quant.debt", exchange=exchange, routing_key="quant.debt.requested", durable=True),
        Queue("quant.wealth", exchange=exchange, routing_key="quant.wealth.requested", durable=True),
        Queue("quant.portfolio", exchange=exchange, routing_key="quant.portfolio.requested", durable=True),
    ]
    while not stop_event.is_set():
        try:
            with Connection(_RABBITMQ_URL, heartbeat=30) as conn:
                with conn.Consumer(queues, callbacks=[_on_message], accept=["json", "application/json"]):
                    log.info("listening on quant request queues")
                    while not stop_event.is_set():
                        try:
                            conn.drain_events(timeout=2)
                        except TimeoutError:
                            conn.heartbeat_check()
                            continue
                        conn.heartbeat_check()
        except OSError as err:
            log.warning("kombu consumer disconnected; reconnecting in 3s: %s", err)
            stop_event.wait(3)
        except Exception:  # noqa: BLE001
            log.exception("kombu consumer crashed; reconnecting in 3s")
            stop_event.wait(3)


def _start_celery_worker(stop_event: threading.Event) -> threading.Thread:
    def _runner() -> None:
        argv = ["worker", "--loglevel=info", "-Q", "netflow.quant", "--concurrency=1"]
        try:
            celery_app.worker_main(argv=argv)
        finally:
            stop_event.set()

    thread = threading.Thread(target=_runner, name="celery-quant", daemon=True)
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

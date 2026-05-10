"""Celery task that categorizes a single transaction.

Flow:
1. Look up the encrypted description from PostgreSQL.
2. Decrypt with the shared AES-256-GCM key.
3. Categorize via rapidfuzz best-match against the keyword corpus, with
   a TF-IDF cosine similarity fallback when fuzz score is too low.
4. Write the category back to the transactions row.
5. Publish ``transactions.categorized`` to the netflow.events exchange.
6. If category in {Utilities, Subscriptions}, also publish
   ``transactions.arbitrage`` to trigger the Playwright worker.
"""

from __future__ import annotations

import datetime as dt
import json
import logging
import os
from typing import Optional

import pika
import psycopg2
import psycopg2.extras
from rapidfuzz import process, fuzz
from sklearn.feature_extraction.text import TfidfVectorizer
from sklearn.metrics.pairwise import cosine_similarity

from app import celery_app
from categories import CATEGORY_KEYWORDS, category_corpus
from crypto import decrypt

log = logging.getLogger(__name__)

_FUZZ_FLOOR = 65.0  # rapidfuzz score below this triggers TF-IDF fallback
_TFIDF_FLOOR = 0.10  # cosine similarity below this -> "Other"

_corpus = category_corpus()
_keywords_only = [pair[0] for pair in _corpus]
_tfidf_vectorizer = TfidfVectorizer(analyzer="char_wb", ngram_range=(3, 5))
_tfidf_matrix = _tfidf_vectorizer.fit_transform(_keywords_only)


def _exchange() -> str:
    return os.environ.get("RABBITMQ_EXCHANGE", "netflow.events")


def _rabbit_url() -> str:
    return os.environ.get("RABBITMQ_URL", "amqp://netflow:netflow@rabbitmq:5672/")


def _db_url() -> str:
    return os.environ.get("DATABASE_URL", "")


def _categorize(description: str, merchant: str) -> tuple[str, float]:
    target = (description or merchant or "").lower().strip()
    if not target:
        return "Other", 0.0

    # Substring fast-path: if any keyword appears verbatim, that wins.
    for keyword, category in _corpus:
        if keyword in target:
            return category, 1.0

    # Fuzzy match
    best = process.extractOne(target, _keywords_only, scorer=fuzz.WRatio)
    if best is not None:
        keyword, score, idx = best
        if score >= _FUZZ_FLOOR:
            return _corpus[idx][1], float(score) / 100.0

    # TF-IDF fallback
    query_vec = _tfidf_vectorizer.transform([target])
    similarities = cosine_similarity(query_vec, _tfidf_matrix).ravel()
    best_idx = int(similarities.argmax())
    best_score = float(similarities[best_idx])
    if best_score >= _TFIDF_FLOOR:
        return _corpus[best_idx][1], best_score
    return "Other", best_score


def _publish(routing_key: str, body: dict) -> None:
    params = pika.URLParameters(_rabbit_url())
    conn = pika.BlockingConnection(params)
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


def _now_iso() -> str:
    return dt.datetime.now(dt.timezone.utc).isoformat()


@celery_app.task(name="categorize_transaction", bind=True, max_retries=3, default_retry_delay=15)
def categorize_transaction(self, transaction_id: int, user_id: int) -> dict:
    log.info("categorizing transaction id=%s user=%s", transaction_id, user_id)
    conn = psycopg2.connect(_db_url())
    try:
        with conn.cursor(cursor_factory=psycopg2.extras.DictCursor) as cur:
            cur.execute(
                """
                SELECT id, user_id, merchant_name, amount,
                       description_ciphertext, description_nonce, description_algorithm
                  FROM transactions
                 WHERE id = %s
                """,
                (transaction_id,),
            )
            row = cur.fetchone()
            if row is None:
                log.warning("transaction %s not found", transaction_id)
                return {"status": "not_found", "transaction_id": transaction_id}

            try:
                description = decrypt(
                    row["description_ciphertext"],
                    row["description_nonce"],
                    row["description_algorithm"],
                )
            except Exception as exc:  # noqa: BLE001
                log.exception("decrypt failed: %s", exc)
                raise self.retry(exc=exc)

            category, confidence = _categorize(description, row["merchant_name"])
            cur.execute(
                """
                UPDATE transactions
                   SET category = %s,
                       ml_confidence = %s,
                       categorized_at = now(),
                       updated_at = now()
                 WHERE id = %s
                """,
                (category, confidence, transaction_id),
            )
            conn.commit()
    finally:
        conn.close()

    occurred_at = _now_iso()
    _publish(
        "transactions.categorized",
        {
            "event_type": "transactions.categorized",
            "transaction_id": transaction_id,
            "user_id": user_id,
            "category": category,
            "confidence": confidence,
            "merchant_name": row["merchant_name"],
            "occurred_at": occurred_at,
        },
    )

    if category in {"Utilities", "Subscriptions"}:
        _publish(
            "transactions.arbitrage",
            {
                "event_type": "transactions.arbitrage",
                "transaction_id": transaction_id,
                "user_id": user_id,
                "merchant_name": row["merchant_name"],
                "category": category,
                "current_amount": float(row["amount"]),
                "occurred_at": occurred_at,
            },
        )

    return {"status": "ok", "transaction_id": transaction_id, "category": category, "confidence": confidence}

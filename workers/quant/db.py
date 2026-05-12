"""PostgreSQL helpers for quant tasks."""

from __future__ import annotations

import json
import os
from contextlib import contextmanager
from typing import Iterator

import psycopg2
import psycopg2.extras


def _db_url() -> str:
    return os.environ.get("DATABASE_URL", "")


@contextmanager
def connect() -> Iterator[psycopg2.extensions.connection]:
    conn = psycopg2.connect(_db_url())
    try:
        yield conn
    finally:
        conn.close()


def dict_cursor(conn):
    return conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor)


def as_json(value) -> str:
    return json.dumps(value, default=str)


def update_status(conn, table: str, run_id: int, status: str, error: str = "") -> None:
    with conn.cursor() as cur:
        cur.execute(
            f"UPDATE {table} SET status = %s, error_message = %s, updated_at = now() WHERE id = %s",
            (status, error, run_id),
        )
    conn.commit()


def fetch_debt_run(conn, run_id: int, user_id: int) -> dict:
    with dict_cursor(conn) as cur:
        cur.execute(
            """
            SELECT id, user_id, preference, monthly_surplus
              FROM debt_optimization_runs
             WHERE id = %s AND user_id = %s
            """,
            (run_id, user_id),
        )
        row = cur.fetchone()
    if row is None:
        raise ValueError("debt run not found")
    return dict(row)


def fetch_debts(conn, user_id: int) -> list[dict]:
    with dict_cursor(conn) as cur:
        cur.execute(
            """
            SELECT id, name, balance, apr, minimum_payment
              FROM debts
             WHERE user_id = %s AND is_active = true
             ORDER BY name
            """,
            (user_id,),
        )
        return [dict(row) for row in cur.fetchall()]


def fetch_profile(conn, user_id: int) -> dict:
    with dict_cursor(conn) as cur:
        cur.execute(
            """
            SELECT user_id, monthly_surplus, cash_balance, emergency_savings_balance,
                   current_investment_balance, monthly_contribution, target_wealth, risk_tolerance
              FROM quant_profiles
             WHERE user_id = %s
            """,
            (user_id,),
        )
        row = cur.fetchone()
    if row is None:
        return {
            "user_id": user_id,
            "monthly_surplus": 0,
            "cash_balance": 0,
            "emergency_savings_balance": 0,
            "current_investment_balance": 0,
            "monthly_contribution": 0,
            "target_wealth": 0,
            "risk_tolerance": "moderate",
        }
    return dict(row)


def fetch_holdings(conn, user_id: int) -> list[dict]:
    with dict_cursor(conn) as cur:
        cur.execute(
            """
            SELECT id, symbol, name, asset_class, quantity, current_value
              FROM investment_holdings
             WHERE user_id = %s
             ORDER BY asset_class, symbol
            """,
            (user_id,),
        )
        return [dict(row) for row in cur.fetchall()]


def fetch_wealth_run(conn, run_id: int, user_id: int) -> dict:
    with dict_cursor(conn) as cur:
        cur.execute(
            """
            SELECT id, user_id, seed, scenario_count, horizons, current_assets,
                   monthly_contribution, target_wealth, assumption_method
              FROM wealth_simulation_runs
             WHERE id = %s AND user_id = %s
            """,
            (run_id, user_id),
        )
        row = cur.fetchone()
    if row is None:
        raise ValueError("wealth run not found")
    return dict(row)


def fetch_portfolio_run(conn, run_id: int, user_id: int) -> dict:
    with dict_cursor(conn) as cur:
        cur.execute(
            """
            SELECT id, user_id, assumption_method, risk_tolerance, portfolio_value
              FROM portfolio_optimization_runs
             WHERE id = %s AND user_id = %s
            """,
            (run_id, user_id),
        )
        row = cur.fetchone()
    if row is None:
        raise ValueError("portfolio run not found")
    return dict(row)

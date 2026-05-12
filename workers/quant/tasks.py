"""Cycle 3 quant Celery tasks."""

from __future__ import annotations

import datetime as dt
import json
import logging
from decimal import Decimal

import numpy as np
import psycopg2.extras

import db
from app import celery_app
from assumptions.demo import ASSET_CLASSES
from assumptions.mapping import map_holding
from assumptions.provider import get_assumptions
from debt.solver import Debt, solve_debts
from portfolio.optimizer import optimize_portfolio as run_portfolio_optimizer
from publisher import publish
from wealth.simulator import simulate_wealth as run_wealth_simulator

log = logging.getLogger(__name__)


def _float(value) -> float:
    if isinstance(value, Decimal):
        return float(value)
    return float(value or 0)


def _horizons(value) -> list[int]:
    if isinstance(value, str):
        value = json.loads(value)
    return [int(item) for item in value]


def _add_months(start: dt.date, months: int) -> dt.date:
    month = start.month - 1 + months
    year = start.year + month // 12
    month = month % 12 + 1
    day = min(start.day, [31, 29 if year % 4 == 0 and (year % 100 != 0 or year % 400 == 0) else 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31][month - 1])
    return dt.date(year, month, day)


def _asset_weights(holdings: list[dict], asset_classes: list[str] | None = None) -> np.ndarray:
    asset_classes = asset_classes or list(ASSET_CLASSES)
    totals = {asset: 0.0 for asset in asset_classes}
    for holding in holdings:
        asset_class = map_holding(str(holding.get("symbol", "")), str(holding.get("name", "")), str(holding.get("asset_class", "")))
        if asset_class in totals:
            totals[asset_class] += _float(holding.get("current_value"))
    vector = np.array([totals[asset] for asset in asset_classes], dtype=float)
    if vector.sum() <= 0:
        vector[0] = 1.0
    return vector / vector.sum()


def _expected_returns(assumptions: dict, asset_classes: list[str]) -> np.ndarray:
    return np.array([
        float(assumptions["asset_classes"][asset]["expected_return"])
        for asset in asset_classes
    ], dtype=float)


def _snapshot_covariance(covariance: np.ndarray, asset_classes: list[str], method: str | None = None) -> dict:
    return {
        "method": method or "assumption_provider",
        "asset_classes": asset_classes,
        "matrix": covariance.round(10).tolist(),
    }


def _publish_success(run_kind: str, run_id: int, user_id: int, payload: dict) -> None:
    publish(
        f"quant.{run_kind}.completed",
        {
            "run_id": run_id,
            "user_id": user_id,
            "run_kind": run_kind,
            "status": "completed",
            "payload": payload,
        },
    )


def _publish_error(run_kind: str, run_id: int, user_id: int, error: Exception) -> None:
    publish(
        "quant.error",
        {
            "run_id": run_id,
            "user_id": user_id,
            "run_kind": run_kind,
            "status": "failed",
            "error": str(error),
        },
    )


@celery_app.task(name="optimize_debt", bind=True)
def optimize_debt(self, run_id: int, user_id: int) -> dict:  # noqa: ARG001
    log.info("optimizing debt run=%s user=%s", run_id, user_id)
    try:
        with db.connect() as conn:
            db.update_status(conn, "debt_optimization_runs", run_id, "running")
            run = db.fetch_debt_run(conn, run_id, user_id)
            debt_rows = db.fetch_debts(conn, user_id)
            debts = [
                Debt(
                    id=int(row["id"]),
                    name=str(row["name"]),
                    balance=_float(row["balance"]),
                    apr=_float(row["apr"]),
                    minimum_payment=_float(row["minimum_payment"]),
                )
                for row in debt_rows
            ]
            result = solve_debts(debts, _float(run["monthly_surplus"]), str(run["preference"]))
            assumptions = {
                "method": "deterministic_amortization",
                "monthly_surplus_is_extra_after_minimums": True,
                "strategies": ["minimum_only", "avalanche", "snowball", "optimized"],
                "disclaimer": "Educational debt payoff estimate; not financial advice.",
            }
            today = dt.date.today()
            with conn.cursor() as cur:
                cur.execute("DELETE FROM debt_optimization_steps WHERE run_id = %s", (run_id,))
                cur.execute("DELETE FROM debt_optimization_strategy_summaries WHERE run_id = %s", (run_id,))
                for strategy_result in result["strategies"].values():
                    payoff_date = _add_months(today, strategy_result.payoff_months) if strategy_result.feasible else None
                    cur.execute(
                        """
                        INSERT INTO debt_optimization_strategy_summaries (
                            run_id, strategy, feasible, payoff_months, payoff_date, total_interest
                        ) VALUES (%s, %s, %s, %s, %s, %s)
                        """,
                        (
                            run_id,
                            strategy_result.strategy,
                            strategy_result.feasible,
                            strategy_result.payoff_months,
                            payoff_date,
                            strategy_result.total_interest,
                        ),
                    )
                    psycopg2.extras.execute_batch(
                        cur,
                        """
                        INSERT INTO debt_optimization_steps (
                            run_id, strategy, month_index, debt_id, debt_name,
                            starting_balance, payment, interest, principal, ending_balance
                        ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                        """,
                        [
                            (
                                run_id,
                                step.strategy,
                                step.month_index,
                                step.debt_id,
                                step.debt_name,
                                step.starting_balance,
                                step.payment,
                                step.interest,
                                step.principal,
                                step.ending_balance,
                            )
                            for step in strategy_result.steps
                        ],
                        page_size=500,
                    )
                recommended = result["recommended"]
                payoff_date = _add_months(today, recommended.payoff_months) if recommended.feasible else None
                cur.execute(
                    """
                    UPDATE debt_optimization_runs
                       SET status = 'completed',
                           recommended_strategy = %s,
                           payoff_months = %s,
                           payoff_date = %s,
                           total_interest = %s,
                           assumptions_snapshot = %s::jsonb,
                           error_message = '',
                           updated_at = now()
                     WHERE id = %s AND user_id = %s
                    """,
                    (
                        result["recommended_strategy"],
                        recommended.payoff_months,
                        payoff_date,
                        recommended.total_interest,
                        db.as_json(assumptions),
                        run_id,
                        user_id,
                    ),
                )
            conn.commit()
        payload = {
            "recommended_strategy": result["recommended_strategy"],
            "payoff_months": result["recommended"].payoff_months,
            "total_interest": result["recommended"].total_interest,
        }
        _publish_success("debt", run_id, user_id, payload)
        return {"status": "completed", **payload}
    except Exception as exc:  # noqa: BLE001
        log.exception("debt optimization failed")
        with db.connect() as conn:
            db.update_status(conn, "debt_optimization_runs", run_id, "failed", str(exc))
        _publish_error("debt", run_id, user_id, exc)
        raise


@celery_app.task(name="simulate_wealth", bind=True)
def simulate_wealth(self, run_id: int, user_id: int) -> dict:  # noqa: ARG001
    log.info("simulating wealth run=%s user=%s", run_id, user_id)
    try:
        with db.connect() as conn:
            db.update_status(conn, "wealth_simulation_runs", run_id, "running")
            run = db.fetch_wealth_run(conn, run_id, user_id)
            holdings = db.fetch_holdings(conn, user_id)
            asset_classes = list(ASSET_CLASSES)
            assumptions, covariance = get_assumptions(str(run["assumption_method"]), asset_classes)
            weights = _asset_weights(holdings, asset_classes)
            expected = _expected_returns(assumptions, asset_classes)
            inflation = float(assumptions.get("globals", {}).get("inflation", 0.025))
            output = run_wealth_simulator(
                current_assets=_float(run["current_assets"]),
                monthly_contribution=_float(run["monthly_contribution"]),
                weights=weights,
                expected_returns=expected,
                covariance=covariance,
                inflation=inflation,
                horizons=_horizons(run["horizons"]),
                scenario_count=int(run["scenario_count"]),
                seed=int(run["seed"]),
                target_wealth=_float(run["target_wealth"]),
            )
            cov_snapshot = _snapshot_covariance(covariance, asset_classes, assumptions.get("covariance_method"))
            with conn.cursor() as cur:
                cur.execute("DELETE FROM wealth_simulation_percentiles WHERE run_id = %s", (run_id,))
                cur.execute("DELETE FROM wealth_simulation_goal_probabilities WHERE run_id = %s", (run_id,))
                psycopg2.extras.execute_batch(
                    cur,
                    """
                    INSERT INTO wealth_simulation_percentiles (
                        run_id, horizon_years, percentile, nominal_value, real_value
                    ) VALUES (%s, %s, %s, %s, %s)
                    """,
                    [
                        (run_id, row["horizon_years"], row["percentile"], row["nominal_value"], row["real_value"])
                        for row in output["percentiles"]
                    ],
                )
                psycopg2.extras.execute_batch(
                    cur,
                    """
                    INSERT INTO wealth_simulation_goal_probabilities (
                        run_id, horizon_years, target_wealth, probability
                    ) VALUES (%s, %s, %s, %s)
                    """,
                    [
                        (run_id, row["horizon_years"], row["target_wealth"], row["probability"])
                        for row in output["goal_probabilities"]
                    ],
                )
                cur.execute(
                    """
                    UPDATE wealth_simulation_runs
                       SET status = 'completed',
                           assumptions_snapshot = %s::jsonb,
                           covariance_snapshot = %s::jsonb,
                           error_message = '',
                           updated_at = now()
                     WHERE id = %s AND user_id = %s
                    """,
                    (db.as_json(assumptions), db.as_json(cov_snapshot), run_id, user_id),
                )
            conn.commit()
        payload = {"scenario_count": int(run["scenario_count"]), "horizons": _horizons(run["horizons"])}
        _publish_success("wealth", run_id, user_id, payload)
        return {"status": "completed", **payload}
    except Exception as exc:  # noqa: BLE001
        log.exception("wealth simulation failed")
        with db.connect() as conn:
            db.update_status(conn, "wealth_simulation_runs", run_id, "failed", str(exc))
        _publish_error("wealth", run_id, user_id, exc)
        raise


@celery_app.task(name="optimize_portfolio", bind=True)
def optimize_portfolio(self, run_id: int, user_id: int) -> dict:  # noqa: ARG001
    log.info("optimizing portfolio run=%s user=%s", run_id, user_id)
    try:
        with db.connect() as conn:
            db.update_status(conn, "portfolio_optimization_runs", run_id, "running")
            run = db.fetch_portfolio_run(conn, run_id, user_id)
            holdings = db.fetch_holdings(conn, user_id)
            asset_classes = list(ASSET_CLASSES)
            weights = _asset_weights(holdings, asset_classes)
            assumptions, covariance = get_assumptions(str(run["assumption_method"]), asset_classes)
            expected = _expected_returns(assumptions, asset_classes)
            risk_free = float(assumptions.get("globals", {}).get("risk_free_rate", 0.04))
            output = run_portfolio_optimizer(
                asset_classes=asset_classes,
                current_weights=weights,
                expected_returns=expected,
                covariance=covariance,
                risk_tolerance=str(run["risk_tolerance"]),
                risk_free_rate=risk_free,
            )
            portfolio_value = _float(run["portfolio_value"])
            cov_snapshot = _snapshot_covariance(covariance, asset_classes, assumptions.get("covariance_method"))
            with conn.cursor() as cur:
                cur.execute("DELETE FROM portfolio_frontier_points WHERE run_id = %s", (run_id,))
                cur.execute("DELETE FROM portfolio_recommendations WHERE run_id = %s", (run_id,))
                psycopg2.extras.execute_batch(
                    cur,
                    """
                    INSERT INTO portfolio_frontier_points (
                        run_id, point_index, expected_return, volatility, sharpe, weights
                    ) VALUES (%s, %s, %s, %s, %s, %s::jsonb)
                    """,
                    [
                        (
                            run_id,
                            point["point_index"],
                            point["expected_return"],
                            point["volatility"],
                            point["sharpe"],
                            db.as_json(point["weights"]),
                        )
                        for point in output["frontier"]
                    ],
                )
                recommended_weights = output["recommended"]["weights"]
                current_weights = output["current"]["weights"]
                recommendation_rows = []
                for asset in asset_classes:
                    current_weight = float(current_weights.get(asset, 0.0))
                    target_weight = float(recommended_weights.get(asset, 0.0))
                    current_value = current_weight * portfolio_value
                    target_value = target_weight * portfolio_value
                    recommendation_rows.append(
                        (
                            run_id,
                            asset,
                            current_weight,
                            target_weight,
                            round(current_value, 2),
                            round(target_value, 2),
                            round(target_value - current_value, 2),
                        )
                    )
                psycopg2.extras.execute_batch(
                    cur,
                    """
                    INSERT INTO portfolio_recommendations (
                        run_id, asset_class, current_weight, target_weight,
                        current_value, target_value, dollar_delta
                    ) VALUES (%s, %s, %s, %s, %s, %s, %s)
                    """,
                    recommendation_rows,
                )
                current = output["current"]
                cur.execute(
                    """
                    UPDATE portfolio_optimization_runs
                       SET status = 'completed',
                           current_return = %s,
                           current_volatility = %s,
                           current_sharpe = %s,
                           min_variance_snapshot = %s::jsonb,
                           max_sharpe_snapshot = %s::jsonb,
                           recommended_snapshot = %s::jsonb,
                           assumptions_snapshot = %s::jsonb,
                           covariance_snapshot = %s::jsonb,
                           error_message = '',
                           updated_at = now()
                     WHERE id = %s AND user_id = %s
                    """,
                    (
                        current["expected_return"],
                        current["volatility"],
                        current["sharpe"],
                        db.as_json(output["min_variance"]),
                        db.as_json(output["max_sharpe"]),
                        db.as_json(output["recommended"]),
                        db.as_json(assumptions),
                        db.as_json(cov_snapshot),
                        run_id,
                        user_id,
                    ),
                )
            conn.commit()
        payload = {
            "current": output["current"],
            "recommended": output["recommended"],
        }
        _publish_success("portfolio", run_id, user_id, payload)
        return {"status": "completed", "frontier_points": len(output["frontier"])}
    except Exception as exc:  # noqa: BLE001
        log.exception("portfolio optimization failed")
        with db.connect() as conn:
            db.update_status(conn, "portfolio_optimization_runs", run_id, "failed", str(exc))
        _publish_error("portfolio", run_id, user_id, exc)
        raise

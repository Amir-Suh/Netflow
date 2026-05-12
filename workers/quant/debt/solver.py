"""Deterministic debt payoff solver."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Callable


@dataclass(frozen=True)
class Debt:
    id: int
    name: str
    balance: float
    apr: float
    minimum_payment: float


@dataclass(frozen=True)
class PayoffStep:
    strategy: str
    month_index: int
    debt_id: int
    debt_name: str
    starting_balance: float
    payment: float
    interest: float
    principal: float
    ending_balance: float


@dataclass(frozen=True)
class StrategyResult:
    strategy: str
    feasible: bool
    payoff_months: int
    total_interest: float
    steps: list[PayoffStep]
    reason: str = ""


def validate_inputs(debts: list[Debt], monthly_surplus: float) -> None:
    if monthly_surplus < 0:
        raise ValueError("monthly_surplus must be nonnegative")
    for debt in debts:
        if debt.balance < 0 or debt.apr < 0 or debt.minimum_payment < 0:
            raise ValueError("debt values must be nonnegative")


def solve_debts(debts: list[Debt], monthly_surplus: float, preference: str = "lowest_interest") -> dict:
    validate_inputs(debts, monthly_surplus)
    active = [debt for debt in debts if debt.balance > 0]
    if not active:
        empty = StrategyResult("optimized", True, 0, 0.0, [])
        return {"recommended_strategy": "optimized", "recommended": empty, "strategies": {"optimized": empty}}

    strategies = {
        "minimum_only": _simulate(active, 0.0, "minimum_only", lambda values: values),
        "avalanche": _simulate(active, monthly_surplus, "avalanche", lambda values: sorted(values, key=lambda d: (-d.apr, d.balance, d.name))),
        "snowball": _simulate(active, monthly_surplus, "snowball", lambda values: sorted(values, key=lambda d: (d.balance, -d.apr, d.name))),
    }
    recommended_strategy = "avalanche" if preference != "fastest_payoff" else "snowball"
    if not strategies[recommended_strategy].feasible:
        feasible = [result for result in strategies.values() if result.feasible]
        recommended_strategy = min(feasible, key=lambda r: (r.total_interest, r.payoff_months)).strategy if feasible else recommended_strategy
    base = strategies[recommended_strategy]
    strategies["optimized"] = StrategyResult(
        strategy="optimized",
        feasible=base.feasible,
        payoff_months=base.payoff_months,
        total_interest=base.total_interest,
        steps=[
            PayoffStep(
                strategy="optimized",
                month_index=step.month_index,
                debt_id=step.debt_id,
                debt_name=step.debt_name,
                starting_balance=step.starting_balance,
                payment=step.payment,
                interest=step.interest,
                principal=step.principal,
                ending_balance=step.ending_balance,
            )
            for step in base.steps
        ],
        reason=base.reason,
    )
    return {
        "recommended_strategy": "optimized",
        "recommended": strategies["optimized"],
        "strategies": strategies,
    }


def _simulate(
    debts: list[Debt],
    monthly_surplus: float,
    strategy: str,
    order_fn: Callable[[list[Debt]], list[Debt]],
    max_months: int = 600,
) -> StrategyResult:
    balances = {debt.id: float(debt.balance) for debt in debts}
    total_interest = 0.0
    steps: list[PayoffStep] = []

    if sum(debt.minimum_payment for debt in debts) + monthly_surplus <= 0:
        return StrategyResult(strategy, False, 0, 0.0, [], "no positive payment available")

    for month in range(1, max_months + 1):
        if all(balance <= 0.005 for balance in balances.values()):
            return StrategyResult(strategy, True, month - 1, round(total_interest, 2), steps)

        month_records: dict[int, dict[str, float]] = {}
        for debt in debts:
            start = max(balances[debt.id], 0.0)
            if start <= 0:
                month_records[debt.id] = {"start": 0.0, "interest": 0.0, "payment": 0.0, "end": 0.0}
                continue
            interest = start * (debt.apr / 12.0)
            total_interest += interest
            balances[debt.id] = start + interest
            payment = min(debt.minimum_payment, balances[debt.id])
            balances[debt.id] -= payment
            month_records[debt.id] = {"start": start, "interest": interest, "payment": payment, "end": balances[debt.id]}

        extra = monthly_surplus
        if extra > 0:
            ordered = order_fn([debt for debt in debts if balances[debt.id] > 0.005])
            for debt in ordered:
                if extra <= 0:
                    break
                payment = min(extra, balances[debt.id])
                balances[debt.id] -= payment
                month_records[debt.id]["payment"] += payment
                month_records[debt.id]["end"] = balances[debt.id]
                extra -= payment

        no_progress = True
        for debt in debts:
            record = month_records[debt.id]
            principal = record["payment"] - record["interest"]
            if principal > 0:
                no_progress = False
            steps.append(
                PayoffStep(
                    strategy=strategy,
                    month_index=month,
                    debt_id=debt.id,
                    debt_name=debt.name,
                    starting_balance=round(record["start"], 2),
                    payment=round(record["payment"], 2),
                    interest=round(record["interest"], 2),
                    principal=round(principal, 2),
                    ending_balance=round(max(record["end"], 0.0), 2),
                )
            )
        if no_progress and any(balances[debt.id] > 0.005 for debt in debts):
            return StrategyResult(strategy, False, month, round(total_interest, 2), steps, "payments do not reduce principal")

    return StrategyResult(strategy, False, max_months, round(total_interest, 2), steps, "payoff exceeded max_months")

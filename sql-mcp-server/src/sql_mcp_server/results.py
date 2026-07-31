"""Typed MCP results and Arrow-to-JSON conversion."""

from __future__ import annotations

import math
from datetime import date, datetime, time
from decimal import Decimal
from typing import Any

from pydantic import BaseModel, Field


class ModelSummary(BaseModel):
    name: str
    description: str | None = None
    column_count: int


class ModelDescription(BaseModel):
    name: str
    columns: list[dict[str, Any]]
    relationships: list[dict[str, Any]] = Field(default_factory=list)


class PlanResult(BaseModel):
    sql: str


class DryRunResult(BaseModel):
    ok: bool = True


class QueryResult(BaseModel):
    columns: list[str]
    rows: list[dict[str, Any]]
    row_count: int
    truncated: bool


class ExecutionDeclined(BaseModel):
    executed: bool = False
    reason: str


def normalize_value(value: Any) -> Any:
    """Recursively convert database values to JSON-native values."""
    if isinstance(value, (datetime, date, time)):
        return value.isoformat()
    if isinstance(value, Decimal):
        return float(value)
    if isinstance(value, (bytes, bytearray)):
        return value.decode(errors="replace")
    if isinstance(value, float) and (math.isnan(value) or math.isinf(value)):
        return None
    if hasattr(value, "item"):
        return normalize_value(value.item())
    if isinstance(value, dict):
        return {str(key): normalize_value(item) for key, item in value.items()}
    if isinstance(value, (list, tuple)):
        return [normalize_value(item) for item in value]
    return value


def table_to_result(table: Any, *, limit: int) -> QueryResult:
    """Serialize an Arrow table, consuming an optional N+1 probe row."""
    truncated = table.num_rows > limit
    visible = table.slice(0, limit) if truncated else table
    columns = [field.name for field in visible.schema]
    rows = [
        {column: normalize_value(value) for column, value in row.items()}
        for row in visible.to_pylist()
    ]
    return QueryResult(
        columns=columns,
        rows=rows,
        row_count=len(rows),
        truncated=truncated,
    )

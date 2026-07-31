"""Read-only SQL validation at the MCP trust boundary."""

from __future__ import annotations

import re

from sqlglot import exp, parse
from sqlglot.errors import ParseError

_EXPLAIN_PREFIX = re.compile(r"^\s*EXPLAIN\s+", re.IGNORECASE)
_FORBIDDEN_TYPES = tuple(
    node_type
    for name in (
        "Alter",
        "Command",
        "Copy",
        "Create",
        "Delete",
        "Drop",
        "Insert",
        "Merge",
        "Set",
        "Transaction",
        "TruncateTable",
        "Update",
    )
    if (node_type := getattr(exp, name, None)) is not None
)


class UnsafeSQLError(ValueError):
    """Raised when SQL is not a single read-only query."""


def _parse_one_query(sql: str) -> exp.Query:
    try:
        statements = [
            statement for statement in parse(sql, read="postgres") if statement
        ]
    except ParseError as exc:
        raise UnsafeSQLError(f"invalid SQL: {exc}") from exc
    if len(statements) != 1:
        raise UnsafeSQLError("exactly one SQL statement is required")
    statement = statements[0]
    if not isinstance(statement, exp.Query):
        raise UnsafeSQLError("only read-only query statements are allowed")
    forbidden = next(statement.find_all(_FORBIDDEN_TYPES), None)
    if forbidden is not None:
        raise UnsafeSQLError(
            f"SQL contains forbidden operation {type(forbidden).__name__}"
        )
    return statement


def validate_read_only_sql(sql: str) -> str:
    """Validate and normalize one SELECT-like PostgreSQL statement."""
    candidate = sql.strip()
    if not candidate:
        raise UnsafeSQLError("SQL must not be empty")

    explained = bool(_EXPLAIN_PREFIX.match(candidate))
    query_sql = _EXPLAIN_PREFIX.sub("", candidate, count=1) if explained else candidate
    statement = _parse_one_query(query_sql)
    normalized = statement.sql(dialect="postgres")
    return f"EXPLAIN {normalized}" if explained else normalized


def normalize_limit(limit: int | None, *, default: int, maximum: int) -> int:
    """Apply the default and hard maximum to a caller-supplied row limit."""
    effective = default if limit is None else limit
    if effective <= 0:
        raise ValueError("row limit must be positive")
    return min(effective, maximum)

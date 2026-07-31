import pytest

from sql_mcp_server.policy import (
    UnsafeSQLError,
    normalize_limit,
    validate_read_only_sql,
)


@pytest.mark.parametrize(
    "sql",
    [
        "SELECT 1",
        "WITH active AS (SELECT 1 AS id) SELECT * FROM active",
        "EXPLAIN SELECT * FROM orders",
    ],
)
def test_accepts_read_only_sql(sql: str) -> None:
    assert validate_read_only_sql(sql)


@pytest.mark.parametrize(
    "sql",
    [
        "INSERT INTO orders VALUES (1)",
        "UPDATE orders SET total = 0",
        "DELETE FROM orders",
        "CREATE TABLE unsafe (id integer)",
        "DROP TABLE orders",
        "ALTER TABLE orders ADD COLUMN unsafe integer",
        "COPY orders TO '/tmp/orders.csv'",
        "CALL refresh_orders()",
        "SET statement_timeout = 0",
        "BEGIN",
        "COMMIT",
        "SELECT 1; SELECT 2",
        "",
    ],
)
def test_rejects_unsafe_sql(sql: str) -> None:
    with pytest.raises(UnsafeSQLError):
        validate_read_only_sql(sql)


def test_normalize_limit_defaults_and_caps() -> None:
    assert normalize_limit(None, default=1000, maximum=10000) == 1000
    assert normalize_limit(25, default=1000, maximum=10000) == 25
    assert normalize_limit(20000, default=1000, maximum=10000) == 10000


@pytest.mark.parametrize("limit", [-1, 0])
def test_normalize_limit_rejects_non_positive_values(limit: int) -> None:
    with pytest.raises(ValueError, match="positive"):
        normalize_limit(limit, default=1000, maximum=10000)

from __future__ import annotations

import time
from pathlib import Path

import psycopg
import pytest

from sql_mcp_server.settings import Settings
from sql_mcp_server.wren_service import create_wren_service

PROJECT_ROOT = Path(__file__).parents[2]

pytestmark = pytest.mark.integration


@pytest.fixture(scope="module", autouse=True)
def load_demo_schema(test_database_url: str) -> None:
    migration = (PROJECT_ROOT / "migrations" / "001_demo.sql").read_text()
    with psycopg.connect(test_database_url, autocommit=True) as connection:
        connection.execute(migration)


def make_settings(test_database_url: str, **overrides: object) -> Settings:
    return Settings(
        database_url=test_database_url,
        MCP_REQUEST_STATE_KEY="integration-test-" + ("x" * 32),
        default_row_limit=2,
        max_row_limit=10,
        **overrides,
    )


def test_wren_service_queries_demo_postgres(test_database_url: str) -> None:
    service = create_wren_service(make_settings(test_database_url))

    try:
        assert [model.name for model in service.list_models()] == [
            "customers",
            "orders",
        ]

        plan = service.dry_plan("SELECT id, name FROM customers ORDER BY id")
        assert "mcp_demo" in plan.sql

        service.dry_run("SELECT id, total FROM orders ORDER BY id")
        result = service.run_sql(
            "SELECT id, name FROM customers ORDER BY id",
            limit=2,
        )
    finally:
        service.close()

    assert result.rows == [
        {"id": 1, "name": "Ada Lovelace"},
        {"id": 2, "name": "Grace Hopper"},
    ]
    assert result.truncated is True


def test_query_timeout_is_enforced(test_database_url: str) -> None:
    service = create_wren_service(
        make_settings(test_database_url, query_timeout_seconds=1)
    )
    started = time.monotonic()

    try:
        with pytest.raises(
            Exception, match="(?i)(timeout|statement timeout|canceling)"
        ):
            service.run_sql("SELECT pg_sleep(5)", limit=1)
    finally:
        service.close()

    assert time.monotonic() - started < 4

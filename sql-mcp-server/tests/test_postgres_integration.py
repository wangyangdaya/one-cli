from __future__ import annotations

import os
from pathlib import Path

import psycopg
import pytest

from sql_mcp_server.settings import Settings
from sql_mcp_server.wren_service import create_wren_service

PROJECT_ROOT = Path(__file__).parents[1]
TEST_DATABASE_URL = os.getenv("SQL_MCP_TEST_DATABASE_URL")

pytestmark = pytest.mark.integration


@pytest.fixture(scope="module", autouse=True)
def load_demo_schema() -> None:
    if not TEST_DATABASE_URL:
        pytest.skip("SQL_MCP_TEST_DATABASE_URL is not configured")
    migration = (PROJECT_ROOT / "migrations" / "001_demo.sql").read_text()
    with psycopg.connect(TEST_DATABASE_URL, autocommit=True) as connection:
        connection.execute(migration)


def test_wren_service_queries_demo_postgres() -> None:
    assert TEST_DATABASE_URL is not None
    settings = Settings(
        database_url=TEST_DATABASE_URL,
        MCP_REQUEST_STATE_KEY="integration-test-" + ("x" * 32),
        default_row_limit=2,
        max_row_limit=10,
    )
    service = create_wren_service(settings)

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

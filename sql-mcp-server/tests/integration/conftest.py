from __future__ import annotations

import os

import pytest


@pytest.fixture(scope="session")
def test_database_url() -> str:
    url = os.getenv("SQL_MCP_TEST_DATABASE_URL")
    if not url:
        pytest.skip("SQL_MCP_TEST_DATABASE_URL is not configured")
    return url

from __future__ import annotations

import os
import sys
from collections.abc import AsyncIterator
from contextlib import asynccontextmanager
from pathlib import Path

import httpx2
import pytest
from mcp import ClientSession, StdioServerParameters
from mcp.client.stdio import stdio_client
from mcp.client.streamable_http import streamable_http_client

from sql_mcp_server.server import build_server

EXPECTED_TOOLS = {
    "describe_model",
    "dry_plan",
    "dry_run",
    "list_models",
    "run_sql",
}
PROJECT_ROOT = Path(__file__).parents[2]


class ProtocolService:
    def list_models(self) -> list[object]:
        return []

    def describe_model(self, name: str) -> object:
        raise ValueError(name)

    def dry_plan(self, sql: str) -> object:
        raise AssertionError(sql)

    def dry_run(self, sql: str) -> object:
        raise AssertionError(sql)

    def run_sql(self, sql: str, limit: int | None = None) -> object:
        raise AssertionError((sql, limit))


class RecordingASGITransport(httpx2.AsyncBaseTransport):
    def __init__(self, app: object) -> None:
        self._inner = httpx2.ASGITransport(app=app)
        self.request_headers: list[dict[str, str]] = []

    async def handle_async_request(self, request: httpx2.Request) -> httpx2.Response:
        self.request_headers.append(dict(request.headers))
        return await self._inner.handle_async_request(request)

    async def aclose(self) -> None:
        await self._inner.aclose()


async def negotiate(client: ClientSession, modern: bool) -> None:
    if modern:
        await client.discover()
        assert client.protocol_version == "2026-07-28"
    else:
        await client.initialize()
        assert client.protocol_version != "2026-07-28"


@asynccontextmanager
async def http_session() -> AsyncIterator[tuple[ClientSession, RecordingASGITransport]]:
    server = build_server(ProtocolService(), request_state_key="x" * 32)
    app = server.streamable_http_app(streamable_http_path="/mcp", host="test")
    transport = RecordingASGITransport(app)

    async with app.router.lifespan_context(app):
        async with httpx2.AsyncClient(
            transport=transport,
            base_url="http://test",
        ) as http_client:
            async with streamable_http_client(
                "http://test/mcp",
                http_client=http_client,
                terminate_on_close=False,
            ) as streams:
                async with ClientSession(*streams) as client:
                    yield client, transport


@pytest.mark.asyncio
@pytest.mark.parametrize("modern", [True, False], ids=["modern", "legacy"])
async def test_streamable_http_transport_negotiates_and_lists_tools(
    modern: bool,
) -> None:
    async with http_session() as (client, _):
        await negotiate(client, modern)
        tools = await client.list_tools()

    assert {tool.name for tool in tools.tools} == EXPECTED_TOOLS


@pytest.mark.asyncio
async def test_modern_http_path_does_not_send_legacy_session_header() -> None:
    async with http_session() as (client, transport):
        await negotiate(client, modern=True)
        await client.list_tools()

    assert transport.request_headers
    assert all("mcp-session-id" not in headers for headers in transport.request_headers)
    assert [headers["mcp-method"] for headers in transport.request_headers] == [
        "server/discover",
        "tools/list",
    ]


@pytest.mark.asyncio
@pytest.mark.parametrize("modern", [True, False], ids=["modern", "legacy"])
async def test_stdio_transport_negotiates_and_lists_tools(modern: bool) -> None:
    environment = os.environ.copy()
    environment.update(
        {
            "DATABASE_URL": "postgresql://demo:demo@127.0.0.1:1/demo",
            "MCP_REQUEST_STATE_KEY": "x" * 32,
        }
    )
    parameters = StdioServerParameters(
        command=sys.executable,
        args=["-m", "sql_mcp_server.cli", "--transport", "stdio"],
        cwd=PROJECT_ROOT,
        env=environment,
    )

    async with stdio_client(parameters) as streams:
        async with ClientSession(*streams) as client:
            await negotiate(client, modern)
            tools = await client.list_tools()

    assert {tool.name for tool in tools.tools} == EXPECTED_TOOLS

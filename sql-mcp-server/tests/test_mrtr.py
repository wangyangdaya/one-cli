from unittest.mock import Mock

import pytest
from mcp import Client
from mcp_types import ElicitResult

from sql_mcp_server.results import QueryResult
from sql_mcp_server.server import build_server


def make_service() -> Mock:
    service = Mock()
    service.run_sql.return_value = QueryResult(
        columns=["id"],
        rows=[{"id": 1}],
        row_count=1,
        truncated=False,
    )
    return service


@pytest.mark.asyncio
@pytest.mark.parametrize("mode", ["2026-07-28", "legacy"])
async def test_run_sql_requires_and_honors_approval_in_both_eras(mode: str) -> None:
    service = make_service()
    prompts = []

    async def approve(_context, params):
        prompts.append(params.message)
        return ElicitResult(action="accept", content={"approved": True})

    server = build_server(service, request_state_key="x" * 32)
    async with Client(server, mode=mode, elicitation_callback=approve) as client:
        result = await client.call_tool(
            "run_sql",
            {"sql": "select id from orders", "limit": 5},
        )

    assert result.is_error is False
    assert result.structured_content["result"]["rows"] == [{"id": 1}]
    assert prompts and "select id from orders" in prompts[0].lower()
    assert "5" in prompts[0]
    service.run_sql.assert_called_once_with("select id from orders", 5)


@pytest.mark.asyncio
@pytest.mark.parametrize("action", ["decline", "cancel"])
async def test_decline_or_cancel_never_executes_sql(action: str) -> None:
    service = make_service()

    async def refuse(_context, _params):
        return ElicitResult(action=action)

    server = build_server(service, request_state_key="x" * 32)
    async with Client(
        server,
        mode="2026-07-28",
        elicitation_callback=refuse,
    ) as client:
        result = await client.call_tool("run_sql", {"sql": "select 1"})

    assert result.is_error is False
    assert result.structured_content == {
        "result": {
            "executed": False,
            "reason": "execution cancelled"
            if action == "cancel"
            else "execution declined",
        }
    }
    service.run_sql.assert_not_called()


@pytest.mark.asyncio
async def test_explicit_false_approval_never_executes_sql() -> None:
    service = make_service()

    async def reject(_context, _params):
        return ElicitResult(action="accept", content={"approved": False})

    server = build_server(service, request_state_key="x" * 32)
    async with Client(
        server,
        mode="2026-07-28",
        elicitation_callback=reject,
    ) as client:
        result = await client.call_tool("run_sql", {"sql": "select 1"})

    assert result.structured_content == {
        "result": {
            "executed": False,
            "reason": "execution declined",
        }
    }
    service.run_sql.assert_not_called()

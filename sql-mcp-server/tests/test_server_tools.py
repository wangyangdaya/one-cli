from unittest.mock import Mock

import pytest
from mcp import Client

from sql_mcp_server.results import (
    DryRunResult,
    ModelDescription,
    ModelSummary,
    PlanResult,
)
from sql_mcp_server.server import build_server


def make_service() -> Mock:
    service = Mock()
    service.list_models.return_value = [
        ModelSummary(name="orders", description="Demo orders", column_count=5)
    ]
    service.describe_model.return_value = ModelDescription(
        name="orders",
        columns=[{"name": "id", "type": "BIGINT", "description": None}],
    )
    service.dry_plan.return_value = PlanResult(sql="SELECT 1")
    service.dry_run.return_value = DryRunResult()
    return service


@pytest.mark.asyncio
@pytest.mark.parametrize("mode", ["2026-07-28", "legacy"])
async def test_discovers_same_tools_in_both_protocol_eras(mode: str) -> None:
    server = build_server(make_service(), request_state_key="x" * 32)

    async with Client(server, mode=mode) as client:
        result = await client.list_tools()

    assert {tool.name for tool in result.tools} == {
        "list_models",
        "describe_model",
        "dry_plan",
        "dry_run",
        "run_sql",
    }


@pytest.mark.asyncio
async def test_calls_typed_read_only_tool_in_memory() -> None:
    service = make_service()
    server = build_server(service, request_state_key="x" * 32)

    async with Client(server, mode="2026-07-28") as client:
        result = await client.call_tool("dry_plan", {"sql": "select 1"})

    assert result.is_error is False
    assert result.structured_content == {"sql": "SELECT 1"}
    service.dry_plan.assert_called_once_with("select 1")

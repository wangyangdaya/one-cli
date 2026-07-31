from unittest.mock import Mock

import pytest
from mcp import Client
from mcp.server.apps import APP_MIME_TYPE

from sql_mcp_server.server import APP_URI, build_server


@pytest.mark.asyncio
async def test_sql_explorer_is_bound_to_run_sql() -> None:
    server = build_server(Mock(), request_state_key="x" * 32)

    async with Client(server, mode="2026-07-28") as client:
        tools = await client.list_tools()
        resources = await client.list_resources()
        content = await client.read_resource(APP_URI)

    run_sql = next(tool for tool in tools.tools if tool.name == "run_sql")
    app_resource = next(
        resource for resource in resources.resources if str(resource.uri) == APP_URI
    )
    html = content.contents[0].text

    assert run_sql.meta["ui"]["resourceUri"] == APP_URI
    assert app_resource.mime_type == APP_MIME_TYPE
    assert "SQL Explorer" in html
    assert 'id="sql"' in html
    assert 'id="dry-plan"' in html
    assert 'id="dry-run"' in html
    assert 'id="run-sql"' in html
    assert 'id="results"' in html
    assert "https://" not in html
    assert "http://" not in html
    assert ".innerHTML" not in html
    assert "event.source !== window.parent" in html


@pytest.mark.asyncio
async def test_server_advertises_apps_extension_and_restrictive_csp() -> None:
    server = build_server(Mock(), request_state_key="x" * 32)

    async with Client(server, mode="2026-07-28") as client:
        resources = await client.list_resources()

    app_resource = next(
        resource for resource in resources.resources if str(resource.uri) == APP_URI
    )
    csp = app_resource.meta["ui"]["csp"]
    assert csp == {
        "connectDomains": [],
        "resourceDomains": [],
        "frameDomains": [],
        "baseUriDomains": [],
    }

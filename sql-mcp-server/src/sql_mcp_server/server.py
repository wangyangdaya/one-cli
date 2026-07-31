"""Official MCP Python SDK v2 server composition."""

from __future__ import annotations

from pathlib import Path
from typing import Annotated

from mcp.server import MCPServer
from mcp.server.apps import Apps, ResourceCsp
from mcp.server.mcpserver import (
    AcceptedElicitation,
    CancelledElicitation,
    Elicit,
    ElicitationResult,
    Resolve,
)
from mcp.server.request_state import RequestStateSecurity
from mcp_types import ToolAnnotations
from pydantic import BaseModel, Field

from sql_mcp_server.results import (
    DryRunResult,
    ExecutionDeclined,
    ModelDescription,
    ModelSummary,
    PlanResult,
    QueryResult,
)
from sql_mcp_server.wren_service import WrenService

APP_URI = "ui://sql-explorer/index.html"
PACKAGE_ROOT = Path(__file__).resolve().parent
PACKAGED_APP_PATH = PACKAGE_ROOT / "app" / "sql-explorer.html"
SOURCE_APP_PATH = PACKAGE_ROOT.parents[1] / "app" / "sql-explorer.html"
APP_PATH = PACKAGED_APP_PATH if PACKAGED_APP_PATH.exists() else SOURCE_APP_PATH
READ_ONLY = ToolAnnotations(
    readOnlyHint=True,
    destructiveHint=False,
    idempotentHint=True,
    openWorldHint=False,
)


class QueryApproval(BaseModel):
    """Explicit approval returned by the MCP client or App host."""

    approved: bool = Field(description="Approve execution of this read-only query.")


def request_query_approval(sql: str, limit: int | None) -> Elicit[QueryApproval]:
    """Create the SDK-managed MRTR/legacy elicitation request."""
    limit_text = "the configured default" if limit is None else str(limit)
    return Elicit(
        message=(
            "Approve this read-only PostgreSQL query?\n\n"
            f"SQL: {sql}\n"
            f"Requested row limit: {limit_text}"
        ),
        schema=QueryApproval,
    )


def _app_html() -> str:
    if APP_PATH.exists():
        return APP_PATH.read_text(encoding="utf-8")
    return "<!doctype html><title>SQL Explorer</title><p>SQL Explorer</p>"


def build_server(
    service: WrenService,
    *,
    request_state_key: str,
) -> MCPServer:
    """Build one server that the SDK serves to modern and legacy clients."""
    apps = Apps()

    @apps.tool(
        resource_uri=APP_URI,
        visibility=["model", "app"],
        name="run_sql",
        title="Run read-only SQL",
        description="Execute a bounded read-only SQL query after user approval.",
        annotations=READ_ONLY,
    )
    def run_sql(
        sql: str,
        approval: Annotated[
            ElicitationResult[QueryApproval],
            Resolve(request_query_approval),
        ],
        limit: int | None = None,
    ) -> QueryResult | ExecutionDeclined:
        if not isinstance(approval, AcceptedElicitation):
            reason = (
                "execution cancelled"
                if isinstance(approval, CancelledElicitation)
                else "execution declined"
            )
            return ExecutionDeclined(reason=reason)
        if not approval.data.approved:
            return ExecutionDeclined(reason="execution declined")
        return service.run_sql(sql, limit)

    apps.add_html_resource(
        APP_URI,
        _app_html(),
        name="sql-explorer",
        title="SQL Explorer",
        description="Plan, validate, approve, and inspect read-only SQL.",
        csp=ResourceCsp(
            connect_domains=[],
            resource_domains=[],
            frame_domains=[],
            base_uri_domains=[],
        ),
        prefers_border=True,
    )

    server = MCPServer(
        name="sql-mcp-server",
        title="SQL MCP Server Demo",
        description="Dual-era read-only PostgreSQL MCP server backed by Wren.",
        version="0.1.0",
        extensions=[apps],
        request_state_security=RequestStateSecurity(
            keys=[request_state_key],
            audience="sql-mcp-server",
        ),
    )

    @server.tool(annotations=READ_ONLY)
    def list_models() -> list[ModelSummary]:
        """List semantic models available to query."""
        return service.list_models()

    @server.tool(annotations=READ_ONLY)
    def describe_model(name: str) -> ModelDescription:
        """Describe a semantic model's columns and relationships."""
        return service.describe_model(name)

    @server.tool(annotations=READ_ONLY)
    def dry_plan(sql: str) -> PlanResult:
        """Expand semantic SQL to PostgreSQL SQL without database access."""
        return service.dry_plan(sql)

    @server.tool(annotations=READ_ONLY)
    def dry_run(sql: str) -> DryRunResult:
        """Validate read-only SQL against PostgreSQL without returning rows."""
        return service.dry_run(sql)

    return server

"""Protocol-neutral WrenEngine facade for MCP tool handlers."""

from __future__ import annotations

import base64
import json
from pathlib import Path
from typing import Any
from urllib.parse import parse_qsl, unquote, urlsplit

from wren.context import build_json, load_models, load_relationships
from wren.engine import WrenEngine
from wren.model.data_source import DataSource

from sql_mcp_server.policy import normalize_limit, validate_read_only_sql
from sql_mcp_server.results import (
    DryRunResult,
    ModelDescription,
    ModelSummary,
    PlanResult,
    QueryResult,
    table_to_result,
)
from sql_mcp_server.settings import Settings


class WrenService:
    """Own query policy, Wren calls, and result bounds."""

    def __init__(
        self,
        *,
        engine: Any,
        project: Path,
        default_row_limit: int,
        max_row_limit: int,
    ) -> None:
        self._engine = engine
        self._project = project
        self._default_row_limit = default_row_limit
        self._max_row_limit = max_row_limit

    def list_models(self) -> list[ModelSummary]:
        summaries = []
        for model in load_models(self._project):
            description = model.get("description") or (
                model.get("properties") or {}
            ).get("description")
            summaries.append(
                ModelSummary(
                    name=model["name"],
                    description=description,
                    column_count=len(model.get("columns") or []),
                )
            )
        return sorted(summaries, key=lambda item: item.name)

    def describe_model(self, name: str) -> ModelDescription:
        model = next(
            (item for item in load_models(self._project) if item.get("name") == name),
            None,
        )
        if model is None:
            raise ValueError(f"model {name!r} was not found")
        columns = [
            {
                "name": column.get("name"),
                "type": column.get("type"),
                "description": column.get("description")
                or (column.get("properties") or {}).get("description"),
            }
            for column in model.get("columns") or []
        ]
        relationships = [
            relationship
            for relationship in load_relationships(self._project)
            if name in (relationship.get("models") or [])
        ]
        return ModelDescription(
            name=name,
            columns=columns,
            relationships=relationships,
        )

    def dry_plan(self, sql: str) -> PlanResult:
        normalized = validate_read_only_sql(sql)
        return PlanResult(sql=self._engine.dry_plan(normalized))

    def dry_run(self, sql: str) -> DryRunResult:
        normalized = validate_read_only_sql(sql)
        self._engine.dry_run(normalized)
        return DryRunResult()

    def run_sql(self, sql: str, limit: int | None = None) -> QueryResult:
        normalized = validate_read_only_sql(sql)
        effective_limit = normalize_limit(
            limit,
            default=self._default_row_limit,
            maximum=self._max_row_limit,
        )
        table = self._engine.query(normalized, effective_limit + 1)
        return table_to_result(table, limit=effective_limit)

    def close(self) -> None:
        self._engine.close()


def _postgres_connection_info(
    database_url: str, timeout_seconds: int
) -> dict[str, Any]:
    parsed = urlsplit(database_url)
    if parsed.scheme not in {"postgres", "postgresql"}:
        raise ValueError("DATABASE_URL must use postgres:// or postgresql://")
    if not parsed.hostname or not parsed.path.strip("/"):
        raise ValueError("DATABASE_URL must include a host and database")
    kwargs = dict(parse_qsl(parsed.query, keep_blank_values=True))
    timeout_options = (
        f"-c statement_timeout={timeout_seconds * 1000} "
        "-c default_transaction_read_only=on"
    )
    kwargs["options"] = " ".join(
        value for value in (kwargs.get("options"), timeout_options) if value
    )
    return {
        "host": parsed.hostname,
        "port": str(parsed.port or 5432),
        "database": parsed.path.lstrip("/"),
        "user": unquote(parsed.username or ""),
        "password": unquote(parsed.password) if parsed.password else None,
        "kwargs": kwargs,
    }


def create_wren_service(settings: Settings) -> WrenService:
    """Build WrenEngine from the bundled project and environment settings."""
    manifest = base64.b64encode(
        json.dumps(build_json(settings.wren_project)).encode("utf-8")
    ).decode("ascii")
    connection_info = _postgres_connection_info(
        settings.database_url.get_secret_value(),
        settings.query_timeout_seconds,
    )
    engine = WrenEngine(
        manifest_str=manifest,
        data_source=DataSource.postgres,
        connection_info=connection_info,
    )
    return WrenService(
        engine=engine,
        project=settings.wren_project,
        default_row_limit=settings.default_row_limit,
        max_row_limit=settings.max_row_limit,
    )

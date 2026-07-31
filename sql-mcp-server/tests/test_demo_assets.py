import base64
import json
from pathlib import Path

import yaml
from wren.context import build_json
from wren.engine import WrenEngine
from wren.model.data_source import DataSource

PROJECT_ROOT = Path(__file__).parents[1]


def test_demo_migration_is_scoped_and_seeded() -> None:
    migration = (PROJECT_ROOT / "migrations" / "001_demo.sql").read_text()

    assert "CREATE SCHEMA IF NOT EXISTS mcp_demo" in migration
    assert "mcp_demo.customers" in migration
    assert "mcp_demo.orders" in migration
    assert "DROP DATABASE" not in migration.upper()
    assert "DROP SCHEMA" not in migration.upper()


def test_wren_project_maps_both_demo_models() -> None:
    project = yaml.safe_load(
        (PROJECT_ROOT / "demo-wren" / "wren_project.yml").read_text()
    )
    customers = yaml.safe_load(
        (
            PROJECT_ROOT / "demo-wren" / "models" / "customers" / "metadata.yml"
        ).read_text()
    )
    orders = yaml.safe_load(
        (PROJECT_ROOT / "demo-wren" / "models" / "orders" / "metadata.yml").read_text()
    )

    assert project["data_source"] == "postgres"
    assert project["schema"] == "mcp_demo"
    assert customers["table_reference"]["schema"] == "mcp_demo"
    assert orders["table_reference"]["schema"] == "mcp_demo"
    assert customers["name"] == "customers"
    assert orders["name"] == "orders"


def test_wren_project_compiles_to_engine_manifest() -> None:
    project = PROJECT_ROOT / "demo-wren"

    manifest = build_json(project)
    encoded = base64.b64encode(json.dumps(manifest).encode()).decode()
    engine = WrenEngine(
        manifest_str=encoded,
        data_source=DataSource.postgres,
        connection_info={},
    )

    assert {model["name"] for model in manifest["models"]} == {
        "customers",
        "orders",
    }
    assert "mcp_demo" in engine.dry_plan("SELECT id, name FROM customers ORDER BY id")

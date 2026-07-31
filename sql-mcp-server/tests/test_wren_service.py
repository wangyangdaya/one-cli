from pathlib import Path
from unittest.mock import Mock

import pyarrow as pa
import pytest

from sql_mcp_server.policy import UnsafeSQLError
from sql_mcp_server.wren_service import WrenService

DEMO_PROJECT = Path(__file__).parents[1] / "demo-wren"


def make_service(engine: Mock) -> WrenService:
    return WrenService(
        engine=engine,
        project=DEMO_PROJECT,
        default_row_limit=2,
        max_row_limit=3,
    )


def test_lists_and_describes_bundled_models() -> None:
    service = make_service(Mock())

    models = service.list_models()
    orders = service.describe_model("orders")

    assert [model.name for model in models] == ["customers", "orders"]
    assert orders.name == "orders"
    assert [column["name"] for column in orders.columns] == [
        "id",
        "customer_id",
        "ordered_at",
        "status",
        "total",
    ]
    assert orders.relationships[0]["name"] == "orders_customer"


def test_dry_plan_validates_then_calls_wren_once() -> None:
    engine = Mock()
    engine.dry_plan.return_value = 'SELECT * FROM "mcp_demo"."orders"'
    service = make_service(engine)

    result = service.dry_plan("select * from orders")

    assert result.sql == 'SELECT * FROM "mcp_demo"."orders"'
    engine.dry_plan.assert_called_once_with("SELECT * FROM orders")


def test_dry_run_validates_then_calls_wren_once() -> None:
    engine = Mock()
    service = make_service(engine)

    result = service.dry_run("select 1")

    assert result.ok is True
    engine.dry_run.assert_called_once_with("SELECT 1")


def test_run_sql_probes_one_extra_row_and_reports_truncation() -> None:
    engine = Mock()
    engine.query.return_value = pa.table({"id": [1, 2, 3]})
    service = make_service(engine)

    result = service.run_sql("select id from orders", limit=2)

    engine.query.assert_called_once_with("SELECT id FROM orders", 3)
    assert result.rows == [{"id": 1}, {"id": 2}]
    assert result.truncated is True


def test_run_sql_caps_requested_limit() -> None:
    engine = Mock()
    engine.query.return_value = pa.table({"id": [1]})
    service = make_service(engine)

    service.run_sql("select id from orders", limit=100)

    engine.query.assert_called_once_with("SELECT id FROM orders", 4)


def test_unsafe_sql_never_reaches_wren() -> None:
    engine = Mock()
    service = make_service(engine)

    with pytest.raises(UnsafeSQLError):
        service.run_sql("delete from orders", limit=2)

    engine.query.assert_not_called()


def test_close_delegates_to_engine() -> None:
    engine = Mock()
    service = make_service(engine)

    service.close()

    engine.close.assert_called_once_with()

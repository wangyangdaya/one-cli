from datetime import date, datetime, time, timezone
from decimal import Decimal

import pyarrow as pa

from sql_mcp_server.results import QueryResult, table_to_result


def test_table_to_result_normalizes_json_values() -> None:
    table = pa.table(
        {
            "decimal": pa.array([Decimal("12.50")], type=pa.decimal128(10, 2)),
            "timestamp": [datetime(2026, 7, 31, 12, 0, tzinfo=timezone.utc)],
            "date": [date(2026, 7, 31)],
            "time": [time(12, 30)],
            "bytes": [b"hello"],
            "missing": [None],
            "nan": [float("nan")],
            "infinity": [float("inf")],
        }
    )

    result = table_to_result(table, limit=10)

    assert result == QueryResult(
        columns=[
            "decimal",
            "timestamp",
            "date",
            "time",
            "bytes",
            "missing",
            "nan",
            "infinity",
        ],
        rows=[
            {
                "decimal": 12.5,
                "timestamp": "2026-07-31T12:00:00+00:00",
                "date": "2026-07-31",
                "time": "12:30:00",
                "bytes": "hello",
                "missing": None,
                "nan": None,
                "infinity": None,
            }
        ],
        row_count=1,
        truncated=False,
    )


def test_table_to_result_uses_probe_row_for_truncation() -> None:
    table = pa.table({"id": [1, 2, 3]})

    result = table_to_result(table, limit=2)

    assert result.rows == [{"id": 1}, {"id": 2}]
    assert result.row_count == 2
    assert result.truncated is True

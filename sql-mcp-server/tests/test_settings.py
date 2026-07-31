from pathlib import Path

import pytest
from pydantic import ValidationError

from sql_mcp_server.settings import Settings


def test_database_url_is_required(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.delenv("DATABASE_URL", raising=False)
    monkeypatch.setenv("MCP_REQUEST_STATE_KEY", "x" * 32)

    with pytest.raises(ValidationError):
        Settings(_env_file=None)


def test_secrets_are_redacted(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv(
        "DATABASE_URL",
        "postgresql://demo:secret@localhost:5432/demo",
    )
    monkeypatch.setenv("MCP_REQUEST_STATE_KEY", "request-state-secret-" + "x" * 16)

    rendered = repr(Settings(_env_file=None))

    assert "secret" not in rendered
    assert "request-state-secret" not in rendered


def test_default_wren_project_is_bundled(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    monkeypatch.setenv("DATABASE_URL", "postgresql://localhost/demo")
    monkeypatch.setenv("MCP_REQUEST_STATE_KEY", "x" * 32)

    settings = Settings(_env_file=None)

    assert settings.wren_project == Path(__file__).parents[1] / "demo-wren"


@pytest.mark.parametrize(
    ("default_limit", "maximum_limit"),
    [(0, 100), (101, 100)],
)
def test_invalid_row_limits_are_rejected(
    monkeypatch: pytest.MonkeyPatch,
    default_limit: int,
    maximum_limit: int,
) -> None:
    monkeypatch.setenv("DATABASE_URL", "postgresql://localhost/demo")
    monkeypatch.setenv("MCP_REQUEST_STATE_KEY", "x" * 32)

    with pytest.raises(ValidationError):
        Settings(
            _env_file=None,
            default_row_limit=default_limit,
            max_row_limit=maximum_limit,
        )

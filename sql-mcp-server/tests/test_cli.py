from __future__ import annotations

from types import SimpleNamespace

import pytest
from pydantic import SecretStr

from sql_mcp_server import cli


class FakeService:
    def __init__(self) -> None:
        self.closed = False

    def close(self) -> None:
        self.closed = True


class FakeServer:
    def __init__(self, *, error: Exception | None = None) -> None:
        self.calls: list[tuple[str, dict[str, object]]] = []
        self.error = error

    def run(self, transport: str, **kwargs: object) -> None:
        self.calls.append((transport, kwargs))
        if self.error is not None:
            raise self.error


def install_fakes(
    monkeypatch: pytest.MonkeyPatch,
    *,
    server: FakeServer,
    service: FakeService,
) -> None:
    settings = SimpleNamespace(
        host="127.0.0.1",
        port=8080,
        request_state_key=SecretStr("x" * 32),
    )
    monkeypatch.setattr(cli, "Settings", lambda: settings)
    monkeypatch.setattr(cli, "create_wren_service", lambda value: service)
    monkeypatch.setattr(
        cli,
        "build_server",
        lambda value, *, request_state_key: server,
    )


def test_main_runs_stdio_and_closes_service(monkeypatch: pytest.MonkeyPatch) -> None:
    service = FakeService()
    server = FakeServer()
    install_fakes(monkeypatch, server=server, service=service)

    assert cli.main(["--transport", "stdio"]) == 0

    assert server.calls == [("stdio", {})]
    assert service.closed is True


def test_main_runs_streamable_http_with_overrides(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    service = FakeService()
    server = FakeServer()
    install_fakes(monkeypatch, server=server, service=service)

    assert (
        cli.main(
            [
                "--transport",
                "http",
                "--host",
                "0.0.0.0",
                "--port",
                "9090",
            ]
        )
        == 0
    )

    assert server.calls == [
        (
            "streamable-http",
            {
                "host": "0.0.0.0",
                "port": 9090,
                "streamable_http_path": "/mcp",
            },
        )
    ]
    assert service.closed is True


def test_main_closes_service_if_server_fails(monkeypatch: pytest.MonkeyPatch) -> None:
    service = FakeService()
    server = FakeServer(error=RuntimeError("boom"))
    install_fakes(monkeypatch, server=server, service=service)

    with pytest.raises(RuntimeError, match="boom"):
        cli.main([])

    assert service.closed is True

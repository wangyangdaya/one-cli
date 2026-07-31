"""Command-line entry point for stdio and Streamable HTTP transports."""

from __future__ import annotations

import argparse
from collections.abc import Sequence

from sql_mcp_server.server import build_server
from sql_mcp_server.settings import Settings
from sql_mcp_server.wren_service import create_wren_service


def _parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="sql-mcp-server",
        description="Read-only PostgreSQL MCP server backed by WrenEngine.",
    )
    parser.add_argument(
        "--transport",
        choices=("stdio", "http"),
        default="stdio",
        help="MCP transport to serve (default: stdio).",
    )
    parser.add_argument("--host", help="HTTP bind host; defaults to SQL_MCP_HOST.")
    parser.add_argument(
        "--port",
        type=int,
        choices=range(1, 65536),
        metavar="PORT",
        help="HTTP bind port; defaults to SQL_MCP_PORT.",
    )
    return parser


def main(argv: Sequence[str] | None = None) -> int:
    """Build the shared server and run the selected official SDK transport."""
    args = _parser().parse_args(argv)
    settings = Settings()
    service = create_wren_service(settings)
    server = build_server(
        service,
        request_state_key=settings.request_state_key.get_secret_value(),
    )

    try:
        if args.transport == "stdio":
            server.run("stdio")
        else:
            server.run(
                "streamable-http",
                host=args.host or settings.host,
                port=args.port or settings.port,
                streamable_http_path="/mcp",
            )
    finally:
        service.close()
    return 0


if __name__ == "__main__":  # pragma: no cover
    raise SystemExit(main())

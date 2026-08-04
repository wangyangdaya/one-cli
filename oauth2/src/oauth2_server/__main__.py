"""Command-line entry point for the local-only server."""

import uvicorn


def main() -> None:
    uvicorn.run(
        "oauth2_server.app:create_app",
        factory=True,
        host="127.0.0.1",
        port=18080,
        log_level="info",
        access_log=False,
    )


if __name__ == "__main__":
    main()

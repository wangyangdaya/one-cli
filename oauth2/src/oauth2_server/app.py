"""FastAPI application composition for the Feishu token broker."""

import httpx
from fastapi import FastAPI

from .broker import create_broker_router
from .config import Settings


def create_app(
    settings: Settings | None = None,
    *,
    upstream_transport: httpx.AsyncBaseTransport | None = None,
) -> FastAPI:
    settings = settings or Settings.from_env()
    app = FastAPI(
        title="one-cli Feishu OAuth Token Broker",
        version="0.1.0",
        docs_url="/docs",
        redoc_url=None,
    )

    @app.get("/healthz")
    async def healthz() -> dict[str, str]:
        return {"status": "ok"}

    app.include_router(
        create_broker_router(settings, upstream_transport=upstream_transport)
    )
    return app

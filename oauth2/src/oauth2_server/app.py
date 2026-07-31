"""FastAPI application composition."""

from collections.abc import Callable
from datetime import UTC, datetime

from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse

from .authorization import create_authorization_router
from .config import Settings
from .resources import (
    ExpenseRepository,
    ResourceAuthorizationError,
    create_resource_router,
    resource_error_response,
)
from .store import InMemoryGrantStore
from .tokens import TokenService


def _utc_now() -> datetime:
    return datetime.now(UTC)


def create_app(
    settings: Settings | None = None,
    *,
    now: Callable[[], datetime] = _utc_now,
) -> FastAPI:
    settings = settings or Settings()
    store = InMemoryGrantStore(now=now)
    tokens = TokenService(settings, now=now)
    expenses = ExpenseRepository()
    app = FastAPI(
        title="one-cli Local OIDC Service",
        version="0.1.0",
        docs_url="/docs",
        redoc_url=None,
    )
    app.state.settings = settings
    app.state.store = store
    app.state.tokens = tokens

    @app.exception_handler(ResourceAuthorizationError)
    async def resource_authorization_error(
        _request: Request, exc: ResourceAuthorizationError
    ) -> JSONResponse:
        return resource_error_response(exc)

    @app.get("/healthz")
    async def healthz():
        return {"status": "ok"}

    @app.get("/.well-known/openid-configuration")
    async def discovery():
        issuer = settings.issuer
        return JSONResponse(
            {
                "issuer": issuer,
                "authorization_endpoint": f"{issuer}/oauth/authorize",
                "token_endpoint": f"{issuer}/oauth/token",
                "revocation_endpoint": f"{issuer}/oauth/revoke",
                "jwks_uri": f"{issuer}/oauth/jwks",
                "userinfo_endpoint": f"{issuer}/oauth/userinfo",
                "response_types_supported": ["code"],
                "subject_types_supported": ["public"],
                "id_token_signing_alg_values_supported": ["RS256"],
                "grant_types_supported": ["authorization_code", "refresh_token"],
                "scopes_supported": list(settings.allowed_scopes),
                "code_challenge_methods_supported": ["S256"],
                "token_endpoint_auth_methods_supported": ["none"],
            },
            headers={"Cache-Control": "no-store"},
        )

    @app.get("/oauth/jwks")
    async def jwks():
        return JSONResponse(
            tokens.jwks(), headers={"Cache-Control": "public, max-age=300"}
        )

    app.include_router(create_authorization_router(settings, store, tokens, now=now))
    app.include_router(create_resource_router(tokens, store, expenses))
    return app


app = create_app()

"""Protected OIDC user information and demo business resources."""

from threading import RLock
from typing import Any

from fastapi import APIRouter, Request
from fastapi.responses import JSONResponse
from pydantic import BaseModel, Field

from .store import InMemoryGrantStore
from .tokens import TokenService, TokenValidationError
from .users import display_name


class ResourceAuthorizationError(Exception):
    def __init__(
        self,
        *,
        status_code: int,
        subtype: str,
        message: str,
        required_scopes: tuple[str, ...] = (),
    ) -> None:
        super().__init__(message)
        self.status_code = status_code
        self.subtype = subtype
        self.message = message
        self.required_scopes = required_scopes


class ExpenseInput(BaseModel):
    description: str = Field(min_length=1, max_length=200)
    amount: float = Field(gt=0)
    owner: str | None = None
    tenant_id: str | None = None


class ExpenseRepository:
    def __init__(self) -> None:
        self._lock = RLock()
        self._next = 3
        self._items: list[dict[str, Any]] = [
            {
                "id": "EXP-001",
                "description": "Alice taxi",
                "amount": 42.0,
                "owner": "user-alice",
                "tenant_id": "company-a",
            },
            {
                "id": "EXP-002",
                "description": "Bob hotel",
                "amount": 320.0,
                "owner": "user-bob",
                "tenant_id": "company-a",
            },
        ]

    def list_for(self, subject: str, tenant_id: str) -> list[dict[str, Any]]:
        with self._lock:
            return [
                item.copy()
                for item in self._items
                if item["owner"] == subject and item["tenant_id"] == tenant_id
            ]

    def create(
        self, subject: str, tenant_id: str, value: ExpenseInput
    ) -> dict[str, Any]:
        with self._lock:
            item = {
                "id": f"EXP-{self._next:03d}",
                "description": value.description,
                "amount": value.amount,
                "owner": subject,
                "tenant_id": tenant_id,
            }
            self._next += 1
            self._items.append(item)
            return item.copy()


def resource_error_response(exc: ResourceAuthorizationError) -> JSONResponse:
    error: dict[str, Any] = {
        "type": "authentication" if exc.status_code == 401 else "authorization",
        "subtype": exc.subtype,
        "message": exc.message,
        "retryable": False,
    }
    if exc.required_scopes:
        error["required_scopes"] = list(exc.required_scopes)
    return JSONResponse(
        {"error": error},
        status_code=exc.status_code,
        headers={"Cache-Control": "no-store"},
    )


def _claims(
    request: Request,
    tokens: TokenService,
    store: InMemoryGrantStore,
    *,
    required_scope: str | None = None,
) -> dict[str, Any]:
    authorization = request.headers.get("Authorization", "")
    scheme, _, value = authorization.partition(" ")
    if scheme.lower() != "bearer" or not value:
        raise ResourceAuthorizationError(
            status_code=401,
            subtype="login_required",
            message="Bearer access token is required",
        )
    try:
        claims = tokens.verify_access_token(value)
    except TokenValidationError as exc:
        raise ResourceAuthorizationError(
            status_code=401,
            subtype="invalid_token",
            message="access token is invalid or expired",
        ) from exc
    if store.is_session_revoked(str(claims.get("sid", ""))):
        raise ResourceAuthorizationError(
            status_code=401,
            subtype="invalid_token",
            message="authorization session is revoked",
        )
    scopes = set(str(claims.get("scope", "")).split())
    if required_scope and required_scope not in scopes:
        raise ResourceAuthorizationError(
            status_code=403,
            subtype="insufficient_scope",
            message="required scope is missing",
            required_scopes=(required_scope,),
        )
    return claims


def create_resource_router(
    tokens: TokenService,
    store: InMemoryGrantStore,
    expenses: ExpenseRepository,
) -> APIRouter:
    router = APIRouter()

    @router.get("/oauth/userinfo")
    async def userinfo(request: Request):
        claims = _claims(request, tokens, store, required_scope="openid")
        result = {
            "sub": claims["sub"],
            "tenant_id": claims["tenant_id"],
        }
        if "profile" in str(claims.get("scope", "")).split():
            result["name"] = display_name(claims["sub"])
        return result

    @router.get("/api/v1/me")
    async def me(request: Request):
        claims = _claims(request, tokens, store, required_scope="openid")
        return {
            "subject": claims["sub"],
            "tenant_id": claims["tenant_id"],
            "client_id": claims["azp"],
        }

    @router.get("/api/v1/me/expenses")
    async def list_expenses(request: Request):
        claims = _claims(request, tokens, store, required_scope="expense:read:self")
        items = expenses.list_for(claims["sub"], claims["tenant_id"])
        return {"items": items, "total": len(items)}

    @router.post("/api/v1/me/expenses", status_code=201)
    async def create_expense(request: Request, value: ExpenseInput):
        claims = _claims(request, tokens, store, required_scope="expense:submit:self")
        return expenses.create(claims["sub"], claims["tenant_id"], value)

    return router

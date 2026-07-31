"""Authorization, token, refresh, and revocation endpoints."""

import base64
import hashlib
import hmac
import secrets
from collections.abc import Callable
from datetime import UTC, datetime, timedelta
from pathlib import Path
from urllib.parse import parse_qsl, urlencode, urlsplit, urlunsplit

from fastapi import APIRouter, Request
from fastapi.responses import HTMLResponse, JSONResponse, RedirectResponse
from fastapi.templating import Jinja2Templates

from .config import Settings
from .models import AuthorizationCode, RefreshGrant
from .store import GrantError, InMemoryGrantStore
from .tokens import TokenService, TokenValidationError

TEMPLATES = Jinja2Templates(directory=Path(__file__).resolve().parents[2] / "templates")
PASSWORD_HASHES = {
    "alice": (
        "4e40e8ffe0ee32fa53e139147ed559229a5930f89c2204706fc174beb36210b3",
        "user-alice",
        "company-a",
        "Alice",
    ),
    "bob": (
        "8d059c3640b97180dd2ee453e20d34ab0cb0f2eccbe87d01915a8e578a202b11",
        "user-bob",
        "company-a",
        "Bob",
    ),
}


def _utc_now() -> datetime:
    return datetime.now(UTC)


def _oauth_error(error: str, description: str, *, status: int = 400) -> JSONResponse:
    return JSONResponse(
        {"error": error, "error_description": description},
        status_code=status,
        headers={"Cache-Control": "no-store", "Pragma": "no-cache"},
    )


def _redirect_with(redirect_uri: str, values: dict[str, str]) -> str:
    parsed = urlsplit(redirect_uri)
    query = dict(parse_qsl(parsed.query, keep_blank_values=True))
    query.update(values)
    return urlunsplit(parsed._replace(query=urlencode(query)))


def _validate_redirect_uri(value: str, settings: Settings) -> bool:
    try:
        parsed = urlsplit(value)
        return (
            parsed.scheme == "http"
            and parsed.hostname in {"127.0.0.1", "::1"}
            and parsed.port is not None
            and parsed.path == settings.callback_path
            and not parsed.username
            and not parsed.password
            and not parsed.fragment
        )
    except ValueError:
        return False


def _validate_authorization(
    values: dict[str, str], settings: Settings
) -> tuple[str, ...]:
    if values.get("response_type") != "code":
        raise ValueError("response_type must be code")
    if values.get("client_id") != settings.client_id:
        raise ValueError("client_id is not registered")
    if not _validate_redirect_uri(values.get("redirect_uri", ""), settings):
        raise ValueError("redirect_uri must be a registered loopback callback")
    if not values.get("state"):
        raise ValueError("state is required")
    if not values.get("nonce"):
        raise ValueError("nonce is required")
    if values.get("code_challenge_method") != "S256":
        raise ValueError("PKCE code_challenge_method must be S256")
    challenge = values.get("code_challenge", "")
    if not challenge or len(challenge) < 43:
        raise ValueError("PKCE code_challenge is invalid")
    scopes = tuple(dict.fromkeys(values.get("scope", "").split()))
    if "openid" not in scopes:
        raise ValueError("openid scope is required")
    unknown = set(scopes) - set(settings.allowed_scopes)
    if unknown:
        raise ValueError(f"unsupported scopes: {' '.join(sorted(unknown))}")
    return scopes


def _password_matches(username: str, password: str) -> bool:
    record = PASSWORD_HASHES.get(username)
    if record is None:
        return False
    actual = hashlib.sha256(password.encode()).hexdigest()
    return hmac.compare_digest(actual, record[0])


def create_authorization_router(
    settings: Settings,
    store: InMemoryGrantStore,
    tokens: TokenService,
    *,
    now: Callable[[], datetime] = _utc_now,
) -> APIRouter:
    router = APIRouter()

    @router.get("/oauth/authorize", response_class=HTMLResponse)
    async def authorize_page(request: Request):
        values = dict(request.query_params)
        try:
            scopes = _validate_authorization(values, settings)
        except ValueError as exc:
            return _oauth_error("invalid_request", str(exc))
        return TEMPLATES.TemplateResponse(
            request,
            "authorize.html",
            {
                "client_id": settings.client_id,
                "scopes": scopes,
                "values": values,
                "error": "",
            },
        )

    @router.post("/oauth/authorize", response_class=HTMLResponse)
    async def authorize_submit(request: Request):
        form = {key: str(value) for key, value in (await request.form()).items()}
        try:
            scopes = _validate_authorization(form, settings)
        except ValueError as exc:
            return _oauth_error("invalid_request", str(exc))
        username = form.get("username", "")
        if not _password_matches(username, form.get("password", "")):
            return TEMPLATES.TemplateResponse(
                request,
                "authorize.html",
                {
                    "client_id": settings.client_id,
                    "scopes": scopes,
                    "values": form,
                    "error": "登录失败，请检查用户名和密码",
                },
                status_code=401,
            )
        if form.get("decision") != "allow":
            location = _redirect_with(
                form["redirect_uri"],
                {"error": "access_denied", "state": form["state"]},
            )
            return RedirectResponse(location, status_code=303)

        _, subject, tenant_id, _ = PASSWORD_HASHES[username]
        code = secrets.token_urlsafe(32)
        store.put_code(
            AuthorizationCode(
                value=code,
                client_id=form["client_id"],
                redirect_uri=form["redirect_uri"],
                subject=subject,
                tenant_id=tenant_id,
                scopes=scopes,
                nonce=form["nonce"],
                code_challenge=form["code_challenge"],
                expires_at=now()
                + timedelta(seconds=settings.authorization_code_seconds),
            )
        )
        location = _redirect_with(
            form["redirect_uri"], {"code": code, "state": form["state"]}
        )
        return RedirectResponse(location, status_code=303)

    @router.post("/oauth/token")
    async def token_endpoint(request: Request):
        form = {key: str(value) for key, value in (await request.form()).items()}
        if form.get("client_secret"):
            return _oauth_error(
                "invalid_client", "public client must not send client_secret"
            )
        if form.get("client_id") != settings.client_id:
            return _oauth_error("invalid_client", "client_id is not registered")
        grant_type = form.get("grant_type")
        if grant_type == "authorization_code":
            return _exchange_code(form, settings, store, tokens, now)
        if grant_type == "refresh_token":
            return _refresh(form, settings, store, tokens)
        return _oauth_error("unsupported_grant_type", "grant_type is not supported")

    @router.post("/oauth/revoke")
    async def revoke_endpoint(request: Request):
        form = {key: str(value) for key, value in (await request.form()).items()}
        if form.get("client_id") != settings.client_id:
            return _oauth_error("invalid_client", "client_id is not registered")
        value = form.get("token", "")
        store.revoke_token(value)
        try:
            claims = tokens.verify_access_token(value)
            if isinstance(claims.get("sid"), str):
                store.revoke_session(claims["sid"])
        except TokenValidationError:
            pass
        return JSONResponse(
            {}, headers={"Cache-Control": "no-store", "Pragma": "no-cache"}
        )

    return router


def _exchange_code(
    form: dict[str, str],
    settings: Settings,
    store: InMemoryGrantStore,
    tokens: TokenService,
    now: Callable[[], datetime],
) -> JSONResponse:
    try:
        code = store.consume_code(form.get("code", ""))
    except GrantError:
        return _oauth_error("invalid_grant", "authorization code is invalid")
    if code.client_id != form.get("client_id") or code.redirect_uri != form.get(
        "redirect_uri"
    ):
        return _oauth_error("invalid_grant", "authorization code binding is invalid")
    verifier = form.get("code_verifier", "")
    if not (43 <= len(verifier) <= 128):
        return _oauth_error("invalid_grant", "PKCE code_verifier is invalid")
    actual = base64.urlsafe_b64encode(hashlib.sha256(verifier.encode()).digest())
    actual = actual.rstrip(b"=").decode()
    if not hmac.compare_digest(actual, code.code_challenge):
        return _oauth_error("invalid_grant", "PKCE verification failed")

    session_id = secrets.token_urlsafe(24)
    refresh_token = secrets.token_urlsafe(32)
    store.put_refresh(
        RefreshGrant(
            token=refresh_token,
            session_id=session_id,
            client_id=code.client_id,
            subject=code.subject,
            tenant_id=code.tenant_id,
            scopes=code.scopes,
            expires_at=now() + timedelta(seconds=settings.refresh_token_seconds),
        )
    )
    return _token_response(
        settings,
        tokens,
        subject=code.subject,
        tenant_id=code.tenant_id,
        scopes=code.scopes,
        session_id=session_id,
        refresh_token=refresh_token,
        nonce=code.nonce,
    )


def _refresh(
    form: dict[str, str],
    settings: Settings,
    store: InMemoryGrantStore,
    tokens: TokenService,
) -> JSONResponse:
    old_token = form.get("refresh_token", "")
    try:
        current = store.get_refresh(old_token)
        if current.client_id != form.get("client_id"):
            return _oauth_error("invalid_grant", "refresh token binding is invalid")
        requested = tuple(dict.fromkeys(form.get("scope", "").split()))
        if requested and not set(requested).issubset(current.scopes):
            return _oauth_error("invalid_scope", "refresh cannot expand scope")
        scopes = requested or current.scopes
        new_token = secrets.token_urlsafe(32)
        rotated = store.rotate_refresh(old_token, new_token)
    except GrantError:
        return _oauth_error("invalid_grant", "refresh token is invalid")
    return _token_response(
        settings,
        tokens,
        subject=rotated.subject,
        tenant_id=rotated.tenant_id,
        scopes=scopes,
        session_id=rotated.session_id,
        refresh_token=new_token,
    )


def _token_response(
    settings: Settings,
    tokens: TokenService,
    *,
    subject: str,
    tenant_id: str,
    scopes: tuple[str, ...],
    session_id: str,
    refresh_token: str,
    nonce: str | None = None,
) -> JSONResponse:
    body = {
        "access_token": tokens.issue_access_token(
            subject=subject,
            tenant_id=tenant_id,
            client_id=settings.client_id,
            scopes=scopes,
            session_id=session_id,
        ),
        "token_type": "Bearer",
        "expires_in": settings.access_token_seconds,
        "refresh_token": refresh_token,
        "scope": " ".join(scopes),
    }
    if nonce is not None:
        body["id_token"] = tokens.issue_id_token(
            subject=subject,
            client_id=settings.client_id,
            nonce=nonce,
            session_id=session_id,
        )
    return JSONResponse(
        body, headers={"Cache-Control": "no-store", "Pragma": "no-cache"}
    )

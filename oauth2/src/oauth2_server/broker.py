"""Minimal confidential-client broker for Feishu OAuth token grants."""

import logging
import time
from collections.abc import Mapping, Sequence
from urllib.parse import urlsplit

import httpx
from fastapi import APIRouter, Request
from fastapi.responses import JSONResponse

from .config import Settings

SAFE_ERROR_FIELDS = ("code", "error", "error_description", "msg")
NO_STORE_HEADERS = {"Cache-Control": "no-store", "Pragma": "no-cache"}
LOGGER = logging.getLogger("uvicorn.error.oauth2")


def create_broker_router(
    settings: Settings,
    *,
    upstream_transport: httpx.AsyncBaseTransport | None = None,
) -> APIRouter:
    router = APIRouter()

    @router.post("/oauth/token")
    async def token(request: Request) -> JSONResponse:
        form = {key: str(value) for key, value in (await request.form()).items()}
        grant_type = form.get("grant_type") or "<missing>"
        LOGGER.info("oauth_token_request_received grant_type=%s", grant_type)
        error = _validate_request(form, settings)
        if error is not None:
            LOGGER.warning(
                "oauth_token_request_rejected grant_type=%s reason=%s",
                grant_type,
                error,
            )
            return _oauth_error("invalid_request", error)

        payload = _upstream_payload(form, settings)
        upstream = urlsplit(settings.feishu_token_url)
        upstream_host = upstream.hostname or "<unknown>"
        upstream_path = upstream.path or "/"
        LOGGER.info(
            "feishu_token_request_started grant_type=%s method=POST "
            "upstream_host=%s upstream_path=%s",
            grant_type,
            upstream_host,
            upstream_path,
        )
        started_at = time.monotonic()
        try:
            async with httpx.AsyncClient(
                transport=upstream_transport,
                timeout=settings.upstream_timeout_seconds,
            ) as client:
                response = await client.post(settings.feishu_token_url, json=payload)
        except httpx.HTTPError as exc:
            duration_ms = round((time.monotonic() - started_at) * 1000)
            detail = _sanitized_error_detail(
                exc,
                (
                    settings.app_secret,
                    settings.feishu_token_url,
                    form.get("code", ""),
                    form.get("refresh_token", ""),
                ),
            )
            LOGGER.error(
                "feishu_token_request_failed grant_type=%s upstream_host=%s "
                "upstream_path=%s duration_ms=%d error_type=%s error_detail=%s",
                grant_type,
                upstream_host,
                upstream_path,
                duration_ms,
                type(exc).__name__,
                detail,
            )
            return _oauth_error(
                "temporarily_unavailable",
                "Feishu token endpoint request failed",
                status=502,
            )

        duration_ms = round((time.monotonic() - started_at) * 1000)
        LOGGER.info(
            "feishu_token_response_received grant_type=%s upstream_host=%s "
            "upstream_path=%s status_code=%d duration_ms=%d",
            grant_type,
            upstream_host,
            upstream_path,
            response.status_code,
            duration_ms,
        )

        try:
            body = response.json()
        except ValueError:
            return _oauth_error(
                "server_error",
                "Feishu token endpoint returned invalid JSON",
                status=502,
            )
        if not isinstance(body, Mapping):
            return _oauth_error(
                "server_error",
                "Feishu token endpoint returned invalid JSON",
                status=502,
            )
        if not response.is_success or body.get("code", 0) != 0:
            safe = {key: body[key] for key in SAFE_ERROR_FIELDS if key in body}
            if "error" not in safe:
                safe["error"] = "token_exchange_failed"
            return JSONResponse(
                safe,
                status_code=response.status_code if not response.is_success else 400,
                headers=NO_STORE_HEADERS,
            )
        return JSONResponse(dict(body), headers=NO_STORE_HEADERS)

    return router


def _validate_request(form: dict[str, str], settings: Settings) -> str | None:
    if form.get("client_id") != settings.app_id:
        return "client_id is not registered"
    grant_type = form.get("grant_type")
    if grant_type == "authorization_code":
        if form.get("redirect_uri") != settings.redirect_uri:
            return "redirect_uri is not registered"
        if not form.get("code"):
            return "code is required"
        return None
    if grant_type == "refresh_token":
        if not form.get("refresh_token"):
            return "refresh_token is required"
        return None
    return "grant_type must be authorization_code or refresh_token"


def _upstream_payload(form: dict[str, str], settings: Settings) -> dict[str, str]:
    payload = {
        "grant_type": form["grant_type"],
        "client_id": settings.app_id,
        "client_secret": settings.app_secret,
    }
    if form["grant_type"] == "authorization_code":
        payload["code"] = form["code"]
        payload["redirect_uri"] = settings.redirect_uri
    else:
        payload["refresh_token"] = form["refresh_token"]
    return payload


def _oauth_error(error: str, description: str, *, status: int = 400) -> JSONResponse:
    return JSONResponse(
        {"error": error, "error_description": description},
        status_code=status,
        headers=NO_STORE_HEADERS,
    )


def _sanitized_error_detail(error: httpx.HTTPError, secrets: Sequence[str]) -> str:
    detail = " ".join(str(error).split()) or "<empty>"
    for secret in secrets:
        if secret:
            detail = detail.replace(secret, "<redacted>")
    return detail[:500]

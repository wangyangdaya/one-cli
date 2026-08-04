import json

import httpx
import pytest
from fastapi.testclient import TestClient

from oauth2_server.app import create_app
from oauth2_server.config import Settings

REDIRECT_URI = "http://127.0.0.1:18081/oauth/callback"


def make_client(handler) -> TestClient:
    settings = Settings(
        app_id="cli_test",
        app_secret="server-secret",
        redirect_uri=REDIRECT_URI,
        feishu_token_url="https://open.feishu.cn/open-apis/authen/v2/oauth/token",
    )
    transport = httpx.MockTransport(handler)
    return TestClient(create_app(settings, upstream_transport=transport))


def test_authorization_code_exchange_adds_server_secret() -> None:
    captured: dict[str, object] = {}

    def upstream(request: httpx.Request) -> httpx.Response:
        captured["url"] = str(request.url)
        captured["body"] = json.loads(request.content)
        return httpx.Response(
            200,
            json={
                "code": 0,
                "access_token": "access-1",
                "refresh_token": "refresh-1",
                "token_type": "Bearer",
                "scope": "offline_access",
                "expires_in": 7200,
                "refresh_token_expires_in": 604800,
            },
        )

    response = make_client(upstream).post(
        "/oauth/token",
        data={
            "grant_type": "authorization_code",
            "client_id": "cli_test",
            "code": "one-time-code",
            "redirect_uri": REDIRECT_URI,
        },
    )

    assert response.status_code == 200
    assert response.headers["cache-control"] == "no-store"
    assert response.json()["refresh_token"] == "refresh-1"
    assert captured == {
        "url": "https://open.feishu.cn/open-apis/authen/v2/oauth/token",
        "body": {
            "grant_type": "authorization_code",
            "client_id": "cli_test",
            "client_secret": "server-secret",
            "code": "one-time-code",
            "redirect_uri": REDIRECT_URI,
        },
    }


def test_refresh_exchange_forwards_rotating_token() -> None:
    captured: dict[str, object] = {}

    def upstream(request: httpx.Request) -> httpx.Response:
        captured.update(json.loads(request.content))
        return httpx.Response(
            200,
            json={
                "code": 0,
                "access_token": "access-2",
                "refresh_token": "refresh-2",
                "token_type": "Bearer",
                "scope": "offline_access",
                "expires_in": 7200,
                "refresh_token_expires_in": 604800,
            },
        )

    response = make_client(upstream).post(
        "/oauth/token",
        data={
            "grant_type": "refresh_token",
            "client_id": "cli_test",
            "refresh_token": "refresh-1",
        },
    )

    assert response.status_code == 200
    assert response.json()["refresh_token"] == "refresh-2"
    assert captured == {
        "grant_type": "refresh_token",
        "client_id": "cli_test",
        "client_secret": "server-secret",
        "refresh_token": "refresh-1",
    }


@pytest.mark.parametrize(
    ("form", "description"),
    [
        (
            {
                "grant_type": "authorization_code",
                "client_id": "wrong-client",
                "code": "code",
                "redirect_uri": REDIRECT_URI,
            },
            "client_id is not registered",
        ),
        (
            {
                "grant_type": "authorization_code",
                "client_id": "cli_test",
                "code": "code",
                "redirect_uri": "http://127.0.0.1:9999/oauth/callback",
            },
            "redirect_uri is not registered",
        ),
    ],
)
def test_broker_rejects_unbound_requests(
    form: dict[str, str], description: str
) -> None:
    def must_not_call(_request: httpx.Request) -> httpx.Response:
        raise AssertionError("upstream must not be called")

    response = make_client(must_not_call).post("/oauth/token", data=form)

    assert response.status_code == 400
    assert response.json() == {
        "error": "invalid_request",
        "error_description": description,
    }


def test_upstream_error_is_forwarded_without_credentials() -> None:
    def upstream(_request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            400,
            json={
                "code": 20065,
                "error": "invalid_grant",
                "error_description": "authorization code was used",
                "debug_secret": "must-not-leak",
            },
        )

    response = make_client(upstream).post(
        "/oauth/token",
        data={
            "grant_type": "authorization_code",
            "client_id": "cli_test",
            "code": "used-code",
            "redirect_uri": REDIRECT_URI,
        },
    )

    assert response.status_code == 400
    assert response.json() == {
        "code": 20065,
        "error": "invalid_grant",
        "error_description": "authorization code was used",
    }
    assert "must-not-leak" not in response.text
    assert "used-code" not in response.text
    assert "server-secret" not in response.text


def test_settings_require_feishu_credentials(monkeypatch: pytest.MonkeyPatch) -> None:
    for name in ("FEISHU_APP_ID", "FEISHU_APP_SECRET", "FEISHU_REDIRECT_URI"):
        monkeypatch.delenv(name, raising=False)

    with pytest.raises(ValueError, match="FEISHU_APP_ID"):
        Settings.from_env()

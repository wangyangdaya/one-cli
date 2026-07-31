from datetime import timedelta
from urllib.parse import parse_qs, urlparse

from conftest import (
    NOW,
    REDIRECT_URI,
    VERIFIER,
    authorization_params,
    exchange_code,
    login_and_exchange,
    login_and_get_code,
)
from fastapi.testclient import TestClient

from oauth2_server.app import create_app
from oauth2_server.config import Settings


def test_authorize_page_describes_client_and_scopes(client: TestClient) -> None:
    response = client.get("/oauth/authorize", params=authorization_params())

    assert response.status_code == 200
    assert "one-cli-demo" in response.text
    assert "expense:read:self" in response.text


def test_authorize_rejects_non_loopback_redirect(client: TestClient) -> None:
    params = authorization_params()
    params["redirect_uri"] = "https://evil.example/oauth/callback"

    response = client.get("/oauth/authorize", params=params)

    assert response.status_code == 400
    assert response.json()["error"] == "invalid_request"


def test_authorize_rejects_plain_pkce(client: TestClient) -> None:
    params = authorization_params()
    params["code_challenge_method"] = "plain"

    response = client.get("/oauth/authorize", params=params)

    assert response.status_code == 400
    assert "S256" in response.json()["error_description"]


def test_authorize_rejects_missing_pkce(client: TestClient) -> None:
    params = authorization_params()
    params.pop("code_challenge")

    response = client.get("/oauth/authorize", params=params)

    assert response.status_code == 400
    assert "code_challenge" in response.json()["error_description"]


def test_login_failure_does_not_redirect_with_code(client: TestClient) -> None:
    form = authorization_params()
    form.update({"username": "alice", "password": "wrong", "decision": "allow"})

    response = client.post("/oauth/authorize", data=form)

    assert response.status_code == 401
    assert "登录失败" in response.text
    assert "wrong" not in response.text


def test_user_can_deny_authorization(client: TestClient) -> None:
    form = authorization_params()
    form.update({"username": "alice", "password": "alice123", "decision": "deny"})

    response = client.post("/oauth/authorize", data=form, follow_redirects=False)

    assert response.status_code == 303
    query = parse_qs(urlparse(response.headers["location"]).query)
    assert query["error"] == ["access_denied"]
    assert query["state"] == ["state-1"]


def test_code_exchange_returns_access_id_and_refresh_tokens(
    client: TestClient,
) -> None:
    result = login_and_exchange(client)

    assert result["token_type"] == "Bearer"
    assert result["expires_in"] == 900
    assert result["access_token"]
    assert result["id_token"]
    assert result["refresh_token"]
    assert "expense:read:self" in result["scope"]


def test_code_exchange_rejects_wrong_verifier_and_burns_code(
    client: TestClient,
) -> None:
    code = login_and_get_code(client)

    response = client.post(
        "/oauth/token",
        data={
            "grant_type": "authorization_code",
            "client_id": "one-cli-demo",
            "code": code,
            "redirect_uri": REDIRECT_URI,
            "code_verifier": "x" * 64,
        },
    )
    replay = client.post(
        "/oauth/token",
        data={
            "grant_type": "authorization_code",
            "client_id": "one-cli-demo",
            "code": code,
            "redirect_uri": REDIRECT_URI,
            "code_verifier": VERIFIER,
        },
    )

    assert response.status_code == 400
    assert response.json()["error"] == "invalid_grant"
    assert "x" * 64 not in response.text
    assert replay.status_code == 400


def test_successful_authorization_code_cannot_be_replayed(
    client: TestClient,
) -> None:
    code = login_and_get_code(client)
    exchange_code(client, code)

    replay = client.post(
        "/oauth/token",
        data={
            "grant_type": "authorization_code",
            "client_id": "one-cli-demo",
            "code": code,
            "redirect_uri": REDIRECT_URI,
            "code_verifier": VERIFIER,
        },
    )

    assert replay.status_code == 400
    assert replay.json()["error"] == "invalid_grant"


def test_expired_authorization_code_is_rejected() -> None:
    clock = {"now": NOW}
    app = create_app(Settings(), now=lambda: clock["now"])
    expiring_client = TestClient(app)
    code = login_and_get_code(expiring_client)
    clock["now"] = NOW + timedelta(minutes=3)

    response = expiring_client.post(
        "/oauth/token",
        data={
            "grant_type": "authorization_code",
            "client_id": "one-cli-demo",
            "code": code,
            "redirect_uri": REDIRECT_URI,
            "code_verifier": VERIFIER,
        },
    )

    assert response.status_code == 400
    assert response.json()["error"] == "invalid_grant"


def test_public_client_rejects_client_secret(client: TestClient) -> None:
    code = login_and_get_code(client)

    response = client.post(
        "/oauth/token",
        data={
            "grant_type": "authorization_code",
            "client_id": "one-cli-demo",
            "client_secret": "must-not-exist",
            "code": code,
            "redirect_uri": REDIRECT_URI,
            "code_verifier": VERIFIER,
        },
    )

    assert response.status_code == 400
    assert response.json()["error"] == "invalid_client"


def test_refresh_rotates_token_and_reuse_revokes_session(
    client: TestClient,
) -> None:
    tokens = login_and_exchange(client)
    first = tokens["refresh_token"]
    refreshed = client.post(
        "/oauth/token",
        data={
            "grant_type": "refresh_token",
            "client_id": "one-cli-demo",
            "refresh_token": first,
        },
    )
    assert refreshed.status_code == 200
    second = refreshed.json()["refresh_token"]
    assert second != first

    reuse = client.post(
        "/oauth/token",
        data={
            "grant_type": "refresh_token",
            "client_id": "one-cli-demo",
            "refresh_token": first,
        },
    )
    after_reuse = client.post(
        "/oauth/token",
        data={
            "grant_type": "refresh_token",
            "client_id": "one-cli-demo",
            "refresh_token": second,
        },
    )

    assert reuse.status_code == 400
    assert reuse.json()["error"] == "invalid_grant"
    assert after_reuse.status_code == 400


def test_refresh_scope_narrowing_cannot_be_expanded_later(
    client: TestClient,
) -> None:
    first = login_and_exchange(client)["refresh_token"]
    narrowed = client.post(
        "/oauth/token",
        data={
            "grant_type": "refresh_token",
            "client_id": "one-cli-demo",
            "refresh_token": first,
            "scope": "openid profile",
        },
    )
    assert narrowed.status_code == 200
    second = narrowed.json()["refresh_token"]

    expanded = client.post(
        "/oauth/token",
        data={
            "grant_type": "refresh_token",
            "client_id": "one-cli-demo",
            "refresh_token": second,
            "scope": "openid profile expense:read:self",
        },
    )

    assert expanded.status_code == 400
    assert expanded.json()["error"] == "invalid_scope"


def test_revoke_is_idempotent_and_prevents_refresh(client: TestClient) -> None:
    refresh_token = login_and_exchange(client)["refresh_token"]

    first = client.post(
        "/oauth/revoke",
        data={"token": refresh_token, "client_id": "one-cli-demo"},
    )
    second = client.post(
        "/oauth/revoke",
        data={"token": refresh_token, "client_id": "one-cli-demo"},
    )
    refresh = client.post(
        "/oauth/token",
        data={
            "grant_type": "refresh_token",
            "client_id": "one-cli-demo",
            "refresh_token": refresh_token,
        },
    )

    assert first.status_code == 200
    assert second.status_code == 200
    assert refresh.status_code == 400


def test_sensitive_values_are_not_logged(client: TestClient, caplog) -> None:
    password_marker = "password-must-not-be-logged"
    verifier_marker = "z" * 64
    form = authorization_params()
    form.update(
        {
            "username": "alice",
            "password": password_marker,
            "decision": "allow",
        }
    )
    client.post("/oauth/authorize", data=form)
    client.post(
        "/oauth/token",
        data={
            "grant_type": "authorization_code",
            "client_id": "one-cli-demo",
            "code": "code-must-not-be-logged",
            "redirect_uri": REDIRECT_URI,
            "code_verifier": verifier_marker,
        },
    )

    logs = caplog.text
    assert password_marker not in logs
    assert verifier_marker not in logs
    assert "code-must-not-be-logged" not in logs

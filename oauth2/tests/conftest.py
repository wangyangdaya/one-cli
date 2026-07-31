import base64
import hashlib
from datetime import UTC, datetime
from urllib.parse import parse_qs, urlparse

import pytest
from fastapi.testclient import TestClient

from oauth2_server.app import create_app
from oauth2_server.config import Settings

NOW = datetime(2026, 7, 31, 10, 0, tzinfo=UTC)
REDIRECT_URI = "http://127.0.0.1:49152/oauth/callback"
VERIFIER = "v" * 64


def pkce_challenge(verifier: str = VERIFIER) -> str:
    digest = hashlib.sha256(verifier.encode()).digest()
    return base64.urlsafe_b64encode(digest).rstrip(b"=").decode()


@pytest.fixture
def client() -> TestClient:
    app = create_app(Settings(), now=lambda: NOW)
    return TestClient(app)


def authorization_params(
    *,
    scopes: str = "openid profile expense:read:self expense:submit:self",
) -> dict[str, str]:
    return {
        "response_type": "code",
        "client_id": "one-cli-demo",
        "redirect_uri": REDIRECT_URI,
        "scope": scopes,
        "state": "state-1",
        "nonce": "nonce-1",
        "code_challenge": pkce_challenge(),
        "code_challenge_method": "S256",
    }


def login_and_get_code(
    client: TestClient,
    *,
    username: str = "alice",
    password: str = "alice123",
    scopes: str = "openid profile expense:read:self expense:submit:self",
) -> str:
    form = authorization_params(scopes=scopes)
    form.update({"username": username, "password": password, "decision": "allow"})
    response = client.post("/oauth/authorize", data=form, follow_redirects=False)
    assert response.status_code == 303, response.text
    query = parse_qs(urlparse(response.headers["location"]).query)
    assert query["state"] == ["state-1"]
    return query["code"][0]


def exchange_code(client: TestClient, code: str, *, verifier: str = VERIFIER) -> dict:
    response = client.post(
        "/oauth/token",
        data={
            "grant_type": "authorization_code",
            "client_id": "one-cli-demo",
            "code": code,
            "redirect_uri": REDIRECT_URI,
            "code_verifier": verifier,
        },
    )
    assert response.status_code == 200, response.text
    return response.json()


def login_and_exchange(
    client: TestClient,
    *,
    username: str = "alice",
    password: str = "alice123",
    scopes: str = "openid profile expense:read:self expense:submit:self",
) -> dict:
    return exchange_code(
        client,
        login_and_get_code(client, username=username, password=password, scopes=scopes),
    )

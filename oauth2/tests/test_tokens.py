from datetime import UTC, datetime, timedelta

import pytest

from oauth2_server.config import Settings
from oauth2_server.tokens import TokenService, TokenValidationError

NOW = datetime(2026, 7, 31, 10, 0, tzinfo=UTC)


def service() -> TokenService:
    return TokenService(Settings(), now=lambda: NOW)


def test_access_token_targets_the_business_api() -> None:
    tokens = service()

    encoded = tokens.issue_access_token(
        subject="user-alice",
        tenant_id="company-a",
        client_id="one-cli-demo",
        scopes=("openid", "expense:read:self"),
        session_id="session-1",
    )
    claims = tokens.verify_access_token(encoded)

    assert claims["sub"] == "user-alice"
    assert claims["aud"] == "demo-api"
    assert claims["azp"] == "one-cli-demo"
    assert claims["tenant_id"] == "company-a"
    assert claims["scope"] == "openid expense:read:self"


def test_id_token_targets_the_public_client_and_contains_nonce() -> None:
    tokens = service()

    encoded = tokens.issue_id_token(
        subject="user-alice",
        client_id="one-cli-demo",
        nonce="nonce-1",
        session_id="session-1",
    )
    claims = tokens.verify_id_token(encoded, client_id="one-cli-demo")

    assert claims["aud"] == "one-cli-demo"
    assert claims["nonce"] == "nonce-1"


def test_access_token_rejects_wrong_audience() -> None:
    tokens = service()
    encoded = tokens.issue_access_token(
        subject="user-alice",
        tenant_id="company-a",
        client_id="one-cli-demo",
        scopes=("openid",),
        session_id="session-1",
    )

    with pytest.raises(TokenValidationError):
        tokens.verify_access_token(encoded, audience="another-api")


def test_access_token_rejects_expired_token() -> None:
    clock = {"now": NOW}
    tokens = TokenService(Settings(), now=lambda: clock["now"])
    encoded = tokens.issue_access_token(
        subject="user-alice",
        tenant_id="company-a",
        client_id="one-cli-demo",
        scopes=("openid",),
        session_id="session-1",
    )
    clock["now"] = NOW + timedelta(minutes=16)

    with pytest.raises(TokenValidationError):
        tokens.verify_access_token(encoded)


def test_access_token_rejects_another_signers_signature() -> None:
    issuer = service()
    verifier = service()
    encoded = issuer.issue_access_token(
        subject="user-alice",
        tenant_id="company-a",
        client_id="one-cli-demo",
        scopes=("openid",),
        session_id="session-1",
    )

    with pytest.raises(TokenValidationError):
        verifier.verify_access_token(encoded)


def test_jwks_publishes_the_active_rsa_key() -> None:
    tokens = service()

    jwks = tokens.jwks()

    assert jwks["keys"][0]["alg"] == "RS256"
    assert jwks["keys"][0]["kid"] == tokens.key_id
    assert jwks["keys"][0]["kty"] == "RSA"
    assert jwks["keys"][0]["n"]
    assert jwks["keys"][0]["e"] == "AQAB"

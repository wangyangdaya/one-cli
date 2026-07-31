from datetime import UTC, datetime, timedelta

import pytest

from oauth2_server.models import AuthorizationCode, RefreshGrant
from oauth2_server.store import (
    GrantExpired,
    GrantNotFound,
    InMemoryGrantStore,
    RefreshTokenReuse,
    SessionRevoked,
)


def fixed_now() -> datetime:
    return datetime(2026, 7, 31, 10, 0, tzinfo=UTC)


def code(value: str = "code-1", *, expires_in: int = 120) -> AuthorizationCode:
    return AuthorizationCode(
        value=value,
        client_id="one-cli-demo",
        redirect_uri="http://127.0.0.1:49152/oauth/callback",
        subject="user-alice",
        tenant_id="company-a",
        scopes=("openid", "expense:read:self"),
        nonce="nonce-1",
        code_challenge="challenge-1",
        expires_at=fixed_now() + timedelta(seconds=expires_in),
    )


def refresh(token: str = "refresh-1") -> RefreshGrant:
    return RefreshGrant(
        token=token,
        session_id="session-1",
        client_id="one-cli-demo",
        subject="user-alice",
        tenant_id="company-a",
        scopes=("openid", "expense:read:self"),
        expires_at=fixed_now() + timedelta(hours=8),
    )


def test_authorization_code_can_only_be_consumed_once() -> None:
    store = InMemoryGrantStore(now=fixed_now)
    store.put_code(code())

    assert store.consume_code("code-1").subject == "user-alice"
    with pytest.raises(GrantNotFound):
        store.consume_code("code-1")


def test_expired_authorization_code_is_rejected() -> None:
    store = InMemoryGrantStore(now=fixed_now)
    store.put_code(code(expires_in=-1))

    with pytest.raises(GrantExpired):
        store.consume_code("code-1")


def test_refresh_token_rotation_invalidates_old_token() -> None:
    store = InMemoryGrantStore(now=fixed_now)
    store.put_refresh(refresh())

    rotated = store.rotate_refresh("refresh-1", "refresh-2")

    assert rotated.token == "refresh-2"
    assert store.get_refresh("refresh-2").subject == "user-alice"


def test_refresh_token_reuse_revokes_the_session() -> None:
    store = InMemoryGrantStore(now=fixed_now)
    store.put_refresh(refresh())
    store.rotate_refresh("refresh-1", "refresh-2")

    with pytest.raises(RefreshTokenReuse):
        store.rotate_refresh("refresh-1", "refresh-3")
    with pytest.raises(SessionRevoked):
        store.get_refresh("refresh-2")

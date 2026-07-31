"""Concurrent in-memory OAuth grant storage."""

from collections.abc import Callable
from datetime import UTC, datetime
from hashlib import sha256
from threading import RLock

from .models import AuthorizationCode, RefreshGrant


class GrantError(Exception):
    """Base class for expected grant failures."""


class GrantNotFound(GrantError):
    pass


class GrantExpired(GrantError):
    pass


class RefreshTokenReuse(GrantError):
    pass


class SessionRevoked(GrantError):
    pass


def _utc_now() -> datetime:
    return datetime.now(UTC)


def _token_hash(value: str) -> str:
    return sha256(value.encode("utf-8")).hexdigest()


class InMemoryGrantStore:
    def __init__(self, *, now: Callable[[], datetime] = _utc_now) -> None:
        self._now = now
        self._lock = RLock()
        self._codes: dict[str, AuthorizationCode] = {}
        self._refresh: dict[str, RefreshGrant] = {}
        self._used_refresh: dict[str, str] = {}
        self._revoked_refresh: dict[str, str] = {}
        self._revoked_sessions: set[str] = set()

    def put_code(self, code: AuthorizationCode) -> None:
        with self._lock:
            self._codes[code.value] = code

    def consume_code(self, value: str) -> AuthorizationCode:
        with self._lock:
            code = self._codes.pop(value, None)
            if code is None:
                raise GrantNotFound("authorization code was not found")
            if code.expires_at <= self._now():
                raise GrantExpired("authorization code is expired")
            return code

    def put_refresh(self, grant: RefreshGrant) -> None:
        with self._lock:
            self._refresh[_token_hash(grant.token)] = grant.without_token()

    def get_refresh(self, token: str) -> RefreshGrant:
        with self._lock:
            token_hash = _token_hash(token)
            grant = self._refresh.get(token_hash)
            if grant is None:
                if token_hash in self._revoked_refresh:
                    raise SessionRevoked("authorization session is revoked")
                if token_hash in self._used_refresh:
                    self._revoke_session_locked(self._used_refresh[token_hash])
                    raise RefreshTokenReuse("refresh token has already been used")
                raise GrantNotFound("refresh token was not found")
            if grant.session_id in self._revoked_sessions:
                raise SessionRevoked("authorization session is revoked")
            if grant.expires_at <= self._now():
                self._refresh.pop(token_hash, None)
                raise GrantExpired("refresh token is expired")
            return grant

    def rotate_refresh(self, old_token: str, new_token: str) -> RefreshGrant:
        with self._lock:
            old_hash = _token_hash(old_token)
            if old_hash in self._used_refresh:
                session_id = self._used_refresh[old_hash]
                self._revoke_session_locked(session_id)
                raise RefreshTokenReuse("refresh token reuse revoked the session")

            grant = self.get_refresh(old_token)
            self._refresh.pop(old_hash)
            self._used_refresh[old_hash] = grant.session_id
            rotated = grant.rotated(new_token)
            self._refresh[_token_hash(new_token)] = rotated.without_token()
            return rotated

    def revoke_token(self, token: str) -> None:
        with self._lock:
            token_hash = _token_hash(token)
            grant = self._refresh.get(token_hash)
            session_id = (
                grant.session_id if grant else self._used_refresh.get(token_hash)
            )
            if session_id:
                self._revoke_session_locked(session_id)

    def revoke_session(self, session_id: str) -> None:
        with self._lock:
            self._revoke_session_locked(session_id)

    def is_session_revoked(self, session_id: str) -> bool:
        with self._lock:
            return session_id in self._revoked_sessions

    def _revoke_session_locked(self, session_id: str) -> None:
        self._revoked_sessions.add(session_id)
        for token_hash, grant in list(self._refresh.items()):
            if grant.session_id == session_id:
                self._refresh.pop(token_hash)
                self._revoked_refresh[token_hash] = session_id

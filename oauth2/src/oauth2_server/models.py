"""Protocol-neutral grant models."""

from dataclasses import dataclass, replace
from datetime import datetime


@dataclass(frozen=True, slots=True)
class AuthorizationCode:
    value: str
    client_id: str
    redirect_uri: str
    subject: str
    tenant_id: str
    scopes: tuple[str, ...]
    nonce: str
    code_challenge: str
    expires_at: datetime


@dataclass(frozen=True, slots=True)
class RefreshGrant:
    token: str
    session_id: str
    client_id: str
    subject: str
    tenant_id: str
    scopes: tuple[str, ...]
    expires_at: datetime

    def without_token(self) -> "RefreshGrant":
        return replace(self, token="")

    def rotated(self, token: str) -> "RefreshGrant":
        return replace(self, token=token)

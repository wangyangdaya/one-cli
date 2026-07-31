"""RS256 JWT issuing and verification for the local OIDC service."""

import base64
import secrets
from collections.abc import Callable
from datetime import UTC, datetime
from hashlib import sha256
from typing import Any

import jwt
from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric import rsa
from jwt import InvalidTokenError

from .config import Settings


class TokenValidationError(ValueError):
    pass


def _utc_now() -> datetime:
    return datetime.now(UTC)


def _b64uint(value: int) -> str:
    size = max(1, (value.bit_length() + 7) // 8)
    return base64.urlsafe_b64encode(value.to_bytes(size, "big")).rstrip(b"=").decode()


class TokenService:
    def __init__(
        self,
        settings: Settings,
        *,
        now: Callable[[], datetime] = _utc_now,
        private_key: rsa.RSAPrivateKey | None = None,
    ) -> None:
        self.settings = settings
        self._now = now
        self._private_key = private_key or rsa.generate_private_key(
            public_exponent=65537, key_size=2048
        )
        self._public_key = self._private_key.public_key()
        public_der = self._public_key.public_bytes(
            serialization.Encoding.DER,
            serialization.PublicFormat.SubjectPublicKeyInfo,
        )
        self.key_id = (
            base64.urlsafe_b64encode(sha256(public_der).digest()[:12])
            .rstrip(b"=")
            .decode()
        )

    def issue_access_token(
        self,
        *,
        subject: str,
        tenant_id: str,
        client_id: str,
        scopes: tuple[str, ...],
        session_id: str,
    ) -> str:
        now = self._now()
        claims = {
            "iss": self.settings.issuer,
            "sub": subject,
            "aud": self.settings.api_audience,
            "azp": client_id,
            "tenant_id": tenant_id,
            "scope": " ".join(scopes),
            "sid": session_id,
            "iat": int(now.timestamp()),
            "exp": int(now.timestamp()) + self.settings.access_token_seconds,
            "jti": secrets.token_urlsafe(18),
        }
        return self._encode(claims)

    def issue_id_token(
        self,
        *,
        subject: str,
        client_id: str,
        nonce: str,
        session_id: str,
    ) -> str:
        now = self._now()
        claims = {
            "iss": self.settings.issuer,
            "sub": subject,
            "aud": client_id,
            "nonce": nonce,
            "sid": session_id,
            "iat": int(now.timestamp()),
            "exp": int(now.timestamp()) + self.settings.access_token_seconds,
            "jti": secrets.token_urlsafe(18),
        }
        return self._encode(claims)

    def verify_access_token(
        self, encoded: str, *, audience: str | None = None
    ) -> dict[str, Any]:
        return self._decode(encoded, audience or self.settings.api_audience)

    def verify_id_token(self, encoded: str, *, client_id: str) -> dict[str, Any]:
        return self._decode(encoded, client_id)

    def jwks(self) -> dict[str, list[dict[str, str]]]:
        numbers = self._public_key.public_numbers()
        return {
            "keys": [
                {
                    "kty": "RSA",
                    "use": "sig",
                    "alg": "RS256",
                    "kid": self.key_id,
                    "n": _b64uint(numbers.n),
                    "e": _b64uint(numbers.e),
                }
            ]
        }

    def _encode(self, claims: dict[str, Any]) -> str:
        return jwt.encode(
            claims,
            self._private_key,
            algorithm="RS256",
            headers={"kid": self.key_id, "typ": "JWT"},
        )

    def _decode(self, encoded: str, audience: str) -> dict[str, Any]:
        try:
            claims = jwt.decode(
                encoded,
                self._public_key,
                algorithms=["RS256"],
                issuer=self.settings.issuer,
                audience=audience,
                options={"verify_exp": False, "verify_iat": False},
            )
        except InvalidTokenError as exc:
            raise TokenValidationError("token validation failed") from exc
        now_timestamp = int(self._now().timestamp())
        expires_at = claims.get("exp")
        if not isinstance(expires_at, int) or expires_at <= now_timestamp:
            raise TokenValidationError("token is expired")
        issued_at = claims.get("iat")
        if not isinstance(issued_at, int) or issued_at > now_timestamp + 60:
            raise TokenValidationError("token issued-at time is invalid")
        return claims

"""Fixed local-only service configuration."""

from dataclasses import dataclass


@dataclass(frozen=True, slots=True)
class Settings:
    issuer: str = "http://127.0.0.1:18080"
    client_id: str = "one-cli-demo"
    api_audience: str = "demo-api"
    callback_path: str = "/oauth/callback"
    access_token_seconds: int = 900
    authorization_code_seconds: int = 120
    refresh_token_seconds: int = 28_800
    allowed_scopes: tuple[str, ...] = (
        "openid",
        "profile",
        "expense:read:self",
        "expense:submit:self",
    )

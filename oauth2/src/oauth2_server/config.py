"""Configuration for the local Feishu OAuth token broker."""

import os
from dataclasses import dataclass
from urllib.parse import urlsplit

DEFAULT_FEISHU_TOKEN_URL = "https://open.feishu.cn/open-apis/authen/v2/oauth/token"


@dataclass(frozen=True, slots=True)
class Settings:
    app_id: str
    app_secret: str
    redirect_uri: str
    feishu_token_url: str = DEFAULT_FEISHU_TOKEN_URL
    upstream_timeout_seconds: float = 15.0

    def __post_init__(self) -> None:
        if not self.app_id.strip():
            raise ValueError("FEISHU_APP_ID is required")
        if not self.app_secret.strip():
            raise ValueError("FEISHU_APP_SECRET is required")
        if not _valid_loopback_redirect(self.redirect_uri):
            raise ValueError(
                "FEISHU_REDIRECT_URI must be an HTTP 127.0.0.1 URL with a port"
            )
        token_url = urlsplit(self.feishu_token_url)
        if token_url.scheme != "https" or not token_url.netloc:
            raise ValueError("FEISHU_TOKEN_URL must be an HTTPS URL")

    @classmethod
    def from_env(cls) -> "Settings":
        return cls(
            app_id=os.getenv("FEISHU_APP_ID", ""),
            app_secret=os.getenv("FEISHU_APP_SECRET", ""),
            redirect_uri=os.getenv("FEISHU_REDIRECT_URI", ""),
            feishu_token_url=os.getenv("FEISHU_TOKEN_URL", DEFAULT_FEISHU_TOKEN_URL),
        )


def _valid_loopback_redirect(value: str) -> bool:
    try:
        parsed = urlsplit(value)
        return (
            parsed.scheme == "http"
            and parsed.hostname == "127.0.0.1"
            and parsed.port is not None
            and parsed.path not in {"", "/"}
            and parsed.query == ""
            and parsed.fragment == ""
            and parsed.username is None
            and parsed.password is None
        )
    except ValueError:
        return False

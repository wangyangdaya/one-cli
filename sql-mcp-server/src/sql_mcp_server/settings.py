"""Validated runtime settings for the SQL MCP demo."""

from pathlib import Path

from pydantic import Field, SecretStr, model_validator
from pydantic_settings import BaseSettings, SettingsConfigDict

PACKAGE_ROOT = Path(__file__).resolve().parent
PROJECT_ROOT = PACKAGE_ROOT.parents[1]
PACKAGED_WREN_PROJECT = PACKAGE_ROOT / "demo-wren"
SOURCE_WREN_PROJECT = PROJECT_ROOT / "demo-wren"
DEFAULT_WREN_PROJECT = (
    PACKAGED_WREN_PROJECT if PACKAGED_WREN_PROJECT.exists() else SOURCE_WREN_PROJECT
)


class Settings(BaseSettings):
    """Environment-backed settings with secrets hidden from representations."""

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
    )

    database_url: SecretStr
    request_state_key: SecretStr = Field(
        validation_alias="MCP_REQUEST_STATE_KEY",
        min_length=32,
    )
    wren_project: Path = DEFAULT_WREN_PROJECT
    host: str = Field(default="127.0.0.1", validation_alias="SQL_MCP_HOST")
    port: int = Field(default=8080, validation_alias="SQL_MCP_PORT", ge=1, le=65535)
    default_row_limit: int = Field(default=1000, ge=1)
    max_row_limit: int = Field(default=10000, ge=1)
    query_timeout_seconds: int = Field(default=15, ge=1)

    @model_validator(mode="after")
    def validate_row_limits(self) -> "Settings":
        if self.default_row_limit > self.max_row_limit:
            raise ValueError("default_row_limit must not exceed max_row_limit")
        return self

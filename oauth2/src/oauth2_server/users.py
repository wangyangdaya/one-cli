"""Fixed local users used by the validation service."""

import hashlib
import hmac
from dataclasses import dataclass


@dataclass(frozen=True, slots=True)
class UserRecord:
    username: str
    password_hash: str
    subject: str
    tenant_id: str
    display_name: str


USERS = {
    "alice": UserRecord(
        username="alice",
        password_hash=(
            "4e40e8ffe0ee32fa53e139147ed559229a5930f89c2204706fc174beb36210b3"
        ),
        subject="user-alice",
        tenant_id="company-a",
        display_name="Alice",
    ),
    "bob": UserRecord(
        username="bob",
        password_hash=(
            "8d059c3640b97180dd2ee453e20d34ab0cb0f2eccbe87d01915a8e578a202b11"
        ),
        subject="user-bob",
        tenant_id="company-a",
        display_name="Bob",
    ),
}
USERS_BY_SUBJECT = {user.subject: user for user in USERS.values()}


def authenticate(username: str, password: str) -> UserRecord | None:
    user = USERS.get(username)
    if user is None:
        return None
    actual = hashlib.sha256(password.encode()).hexdigest()
    return user if hmac.compare_digest(actual, user.password_hash) else None


def display_name(subject: str) -> str | None:
    user = USERS_BY_SUBJECT.get(subject)
    return user.display_name if user else None

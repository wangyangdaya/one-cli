from conftest import login_and_exchange
from fastapi.testclient import TestClient


def bearer(token: str) -> dict[str, str]:
    return {"Authorization": f"Bearer {token}"}


def test_protected_api_requires_bearer_token(client: TestClient) -> None:
    response = client.get("/api/v1/me")

    assert response.status_code == 401
    assert response.json()["error"]["subtype"] == "login_required"


def test_userinfo_and_me_return_the_token_subject(client: TestClient) -> None:
    token = login_and_exchange(client)["access_token"]

    userinfo = client.get("/oauth/userinfo", headers=bearer(token))
    me = client.get("/api/v1/me", headers=bearer(token))

    assert userinfo.json()["sub"] == "user-alice"
    assert me.json()["subject"] == "user-alice"
    assert me.json()["tenant_id"] == "company-a"


def test_userinfo_only_returns_name_with_profile_scope(client: TestClient) -> None:
    token = login_and_exchange(client, scopes="openid")["access_token"]

    userinfo = client.get("/oauth/userinfo", headers=bearer(token))

    assert userinfo.status_code == 200
    assert userinfo.json()["sub"] == "user-alice"
    assert "name" not in userinfo.json()


def test_expenses_are_isolated_by_token_subject(client: TestClient) -> None:
    alice = login_and_exchange(client, username="alice", password="alice123")
    bob = login_and_exchange(client, username="bob", password="bob123")

    alice_expenses = client.get(
        "/api/v1/me/expenses", headers=bearer(alice["access_token"])
    ).json()["items"]
    bob_expenses = client.get(
        "/api/v1/me/expenses", headers=bearer(bob["access_token"])
    ).json()["items"]

    assert {item["owner"] for item in alice_expenses} == {"user-alice"}
    assert {item["owner"] for item in bob_expenses} == {"user-bob"}
    assert {item["id"] for item in alice_expenses}.isdisjoint(
        {item["id"] for item in bob_expenses}
    )


def test_missing_read_scope_returns_structured_error(client: TestClient) -> None:
    token = login_and_exchange(client, scopes="openid profile expense:submit:self")[
        "access_token"
    ]

    response = client.get("/api/v1/me/expenses", headers=bearer(token))

    assert response.status_code == 403
    error = response.json()["error"]
    assert error["subtype"] == "insufficient_scope"
    assert error["required_scopes"] == ["expense:read:self"]


def test_create_expense_derives_owner_from_token(client: TestClient) -> None:
    token = login_and_exchange(client)["access_token"]

    response = client.post(
        "/api/v1/me/expenses",
        headers=bearer(token),
        json={"description": "Train", "amount": 88.5, "owner": "user-bob"},
    )

    assert response.status_code == 201
    assert response.json()["owner"] == "user-alice"
    assert response.json()["tenant_id"] == "company-a"

from fastapi.testclient import TestClient


def test_discovery_publishes_oidc_pkce_metadata(client: TestClient) -> None:
    response = client.get("/.well-known/openid-configuration")

    assert response.status_code == 200
    document = response.json()
    assert document["issuer"] == "http://127.0.0.1:18080"
    assert document["authorization_endpoint"].endswith("/oauth/authorize")
    assert document["token_endpoint"].endswith("/oauth/token")
    assert document["revocation_endpoint"].endswith("/oauth/revoke")
    assert document["jwks_uri"].endswith("/oauth/jwks")
    assert document["response_types_supported"] == ["code"]
    assert "authorization_code" in document["grant_types_supported"]
    assert document["code_challenge_methods_supported"] == ["S256"]


def test_jwks_and_health_are_available(client: TestClient) -> None:
    jwks = client.get("/oauth/jwks")
    health = client.get("/healthz")

    assert jwks.status_code == 200
    assert jwks.json()["keys"][0]["alg"] == "RS256"
    assert health.json() == {"status": "ok"}

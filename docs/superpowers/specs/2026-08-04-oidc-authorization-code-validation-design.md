# OIDC Authorization Code Validation Design

## Goal

Add optional OpenID Connect identity validation to the generated OAuth 2.0
Authorization Code login flow for both Go and Rust targets. OIDC remains an
extension of the existing authorization-code flow rather than a separate grant
type or a separate authentication runtime.

## Configuration

OIDC is enabled explicitly inside an OAuth 2.0 authorization-code
configuration:

```yaml
auth:
  type: oauth2
  grant_type: authorization_code
  client_id: business-cli
  authorization_url: https://identity.example.com/oauth/authorize
  token_url: https://identity.example.com/oauth/token
  scopes:
    - openid
  oidc:
    enabled: true
    issuer: https://identity.example.com
```

Generation rejects OIDC configuration unless:

- the grant type is `authorization_code`;
- `openid` is present in the requested scopes;
- `issuer` is an absolute HTTPS URL without user information, query, or
  fragment. HTTP is allowed only for `127.0.0.1` or `localhost` development.

OIDC is never inferred from scopes. Without `oidc.enabled: true`, the generated
CLI retains its existing OAuth-only behavior.

## Login and Validation Flow

When OIDC is enabled, the generated CLI:

1. Generates a cryptographically random nonce for each login attempt.
2. Adds the nonce to the authorization request while retaining the existing
   state and redirect-URI behavior.
3. Exchanges the authorization code through the configured token endpoint.
4. Requires an `id_token` in the token response. Custom token-response mappings
   may identify where `id_token` is returned.
5. Loads `/.well-known/openid-configuration` from the configured issuer and
   requires the discovered issuer to match exactly.
6. Loads the advertised JWKS and selects exactly one compatible signing key for
   the ID Token `kid`.
7. Verifies the ID Token before persisting any access or refresh token.

The initial implementation supports RS256 ID Tokens. It rejects unsupported
algorithms rather than accepting an unsigned or unrecognized token.

## Required ID Token Checks

Both generated targets must verify:

- JWT signature against the selected JWKS key;
- `iss` equals the configured issuer;
- `aud` contains the configured OAuth client ID;
- `azp` equals the client ID when `aud` has multiple values or `azp` is present;
- `exp` is present and has not expired, allowing at most 60 seconds of clock
  skew;
- `iat` is present and is not more than 60 seconds in the future;
- `nonce` exactly matches the value generated for the current login attempt.

Validation errors must be actionable but must not include the ID Token, access
token, refresh token, authorization code, or nonce.

## Storage and Refresh

The ID Token is used only to validate the interactive login. It is not required
for authenticated API requests and is not added to the persisted OAuth token
record. Existing access-token storage, refresh-token rotation, status, and
logout semantics remain unchanged.

OIDC validation occurs before token persistence. A discovery, JWKS, signature,
claim, or nonce failure leaves the previous local session untouched.

Automatic refresh remains an OAuth operation. Refresh responses are not
required to contain a new ID Token in this scope.

## Security Boundaries

- Discovery and JWKS responses are limited to one MiB and fetched with bounded
  timeouts.
- The discovered `jwks_uri` must use HTTPS, except for loopback development,
  and must not contain user information or a fragment.
- Redirect-URI runtime validation remains restricted to HTTP on `127.0.0.1`
  with an explicit port and callback path.
- The CLI does not log request bodies or token material.
- The resource server remains responsible for validating access tokens and
  enforcing audience, scope, tenant, and business authorization.

## Go and Rust Parity

Go and Rust generated CLIs expose the same configuration and observable
behavior. They may use different JWT libraries or standard-library primitives,
but must apply the same algorithm, key-selection, issuer, audience, authorized
party, time, and nonce rules.

## Verification

Tests must cover:

- generation-time acceptance of a valid OIDC configuration;
- rejection of missing `openid`, missing issuer, invalid issuer, and OIDC on a
  non-authorization-code grant;
- authorization requests containing a fresh nonce;
- successful Go and Rust login with Discovery, JWKS, and a valid signed ID
  Token;
- rejection of missing ID Token, invalid signature, issuer mismatch, audience
  mismatch, invalid `azp`, expired token, future `iat`, and nonce mismatch;
- proof that failed OIDC validation does not save returned OAuth credentials;
- unchanged OAuth-only login behavior when OIDC is disabled.

Targeted tests run first during development, followed by the complete Go test
suite and the OAuth broker's Python tests and lint checks.

## Out of Scope

- a separate `auth.type: oidc` runtime;
- implicit OIDC activation from the `openid` scope;
- algorithms other than RS256;
- UserInfo requests, identity display, account selection, or multi-account
  storage;
- validating access tokens inside the generated CLI;
- requiring a new ID Token during refresh.

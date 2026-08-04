# Feishu OAuth CLI Login MVP Design

## Goal

Validate a generated CLI's OAuth 2.0 authorization-code login lifecycle against
Feishu: interactive login, local access/refresh token storage, status reporting,
automatic refresh, and logout.

This is a development demo. Feishu is the authorization server. The service in
`oauth2/` is only a confidential token broker that protects the Feishu App
Secret; it does not implement an authorization server, user database, login
page, OIDC discovery, JWT issuance, or PKCE.

## Scope

The generated CLI exposes exactly three login-related commands:

- `login`: open Feishu authorization in a browser, receive the authorization
  callback, exchange the code through the demo broker, and save the returned
  token set.
- `status`: report `valid`, `needs_refresh`, `expired`, or `not_logged_in` from
  locally stored token metadata without refreshing it.
- `logout`: remove the local token set. Server-side token revocation is outside
  this MVP.

Before an authenticated API call, the CLI uses the current access token when it
has more than five minutes remaining. Otherwise, while the refresh token is
still valid, it refreshes through the broker and atomically replaces the stored
token set. A missing, expired, or rejected refresh token causes an actionable
error directing the user to run `login` again.

The MVP does not include PKCE, OS keychain integration, multiple accounts,
`auth list`, scope inspection, QR codes, device authorization, OIDC, or a custom
session/token format.

## Architecture

### Generated CLI

The CLI is an OAuth public-facing client but never receives the Feishu App
Secret. Its generated runtime configuration contains:

- the Feishu authorization endpoint;
- the local demo broker token endpoint;
- the Feishu App ID as `client_id`;
- requested scopes, including `offline_access`;
- one fixed loopback callback URL registered in the Feishu developer console.

For this demo, the default callback URL is
`http://127.0.0.1:18081/oauth/callback`. The listener binds only to
`127.0.0.1`. The callback path and port come from the configured redirect URL;
the CLI does not allocate a dynamic port.

### Token broker

The FastAPI service in `oauth2/` reads `FEISHU_APP_ID` and
`FEISHU_APP_SECRET` from its environment. It exposes a single token endpoint
for two grant types:

- `authorization_code`: accepts the code and exact redirect URI, adds the App
  Secret, and forwards the exchange to Feishu.
- `refresh_token`: accepts the current refresh token, adds the App Secret, and
  forwards the refresh request to Feishu.

The broker calls Feishu's current token endpoint:

`POST https://open.feishu.cn/open-apis/authen/v2/oauth/token`

It forwards the successful Feishu token response without inventing another
credential format. It never logs authorization codes, App Secrets, access
tokens, or refresh tokens. It rejects client IDs other than its configured
Feishu App ID and only accepts the configured loopback redirect URI.

The broker is intentionally a local validation service, not a production-ready
multi-tenant OAuth backend.

## Login Flow

1. `login` loads the runtime OAuth configuration.
2. It starts an HTTP listener on the configured loopback address.
3. It generates a cryptographically random `state` value and keeps it only in
   memory for the duration of login.
4. It opens Feishu's authorization endpoint:
   `https://accounts.feishu.cn/open-apis/authen/v1/authorize`.
5. The request includes `client_id`, `response_type=code`, the URL-encoded fixed
   `redirect_uri`, the requested scopes, and `state`.
6. Feishu redirects the browser to the loopback callback with either `code` and
   `state`, or `error=access_denied` and `state`.
7. The CLI rejects a mismatched state, a missing code, an authorization error,
   or a callback received after the two-minute timeout.
8. The CLI sends the code, `grant_type=authorization_code`, `client_id`, and
   the exact `redirect_uri` to the demo broker.
9. The broker adds the App Secret and exchanges the code with Feishu.
10. The CLI validates the returned token fields and saves the token set before
    printing `login successful`.

Authorization codes are treated as one-time values. The CLI does not retry a
failed code exchange automatically.

## Local Token Model

The CLI stores one JSON document in its existing per-user application config
directory. The file is created with mode `0600` and contains:

```json
{
  "access_token": "...",
  "refresh_token": "...",
  "token_type": "Bearer",
  "scope": "offline_access ...",
  "expires_at": "2026-08-04T12:00:00Z",
  "refresh_expires_at": "2026-08-11T10:00:00Z"
}
```

The absolute expiration timestamps are calculated from Feishu's returned
`expires_in` and `refresh_token_expires_in` values. The CLI does not hard-code
token lifetimes. Token values must never appear in `status`, normal logs, trace
logs, or errors.

An atomic write uses a temporary file in the destination directory followed by
a rename, so interruption cannot leave a partially written credential file.

## Status and Refresh Semantics

`status` is read-only and does not make a network request:

- `not_logged_in`: no token file exists;
- `valid`: the access token expires more than five minutes from now;
- `needs_refresh`: the access token is expired or within the five-minute
  refresh window, while the refresh token remains valid;
- `expired`: the refresh token is missing or expired.

Authenticated API calls resolve credentials as follows:

1. Load the local token set.
2. Return the access token when status is `valid`.
3. When status is `needs_refresh`, send `grant_type=refresh_token`, `client_id`,
   and the current refresh token to the broker.
4. Require a new access token and refresh token in the response. Feishu refresh
   tokens are single-use, so the response atomically replaces both old tokens.
5. Return the refreshed access token to the pending API request.
6. For `expired` or any terminal refresh error, return an error that asks the
   user to run `login` again.

Concurrent commands in this MVP are protected only by atomic file replacement;
cross-process refresh locking is outside scope.

## Configuration

The demo service requires:

```text
FEISHU_APP_ID
FEISHU_APP_SECRET
FEISHU_REDIRECT_URI=http://127.0.0.1:18081/oauth/callback
```

The generated CLI runtime example points authorization directly to Feishu and
token exchange to the local broker. It requests `offline_access` so Feishu can
return a refresh token. The README explains that the callback URL must be added
to the application's security settings and the requested scopes must be enabled
and published before testing.

## Error Handling

The CLI presents concise errors for browser launch failure, callback listener
failure, timeout, user rejection, state mismatch, malformed broker responses,
expired login, and refresh failure. The broker preserves Feishu's HTTP status
and safe OAuth error fields but redacts upstream bodies when they could contain
credentials.

Configuration validation fails at startup when the broker is missing App ID,
App Secret, or a valid loopback redirect URL. Upstream calls use a finite
timeout and do not retry authorization-code exchange.

## Testing

Broker tests use an injected HTTP transport or mock server and cover:

- forwarding an authorization-code exchange with the server-side App Secret;
- forwarding refresh-token exchange and returning the rotated refresh token;
- rejecting an unexpected client ID or redirect URI;
- surfacing safe Feishu OAuth errors without leaking credentials;
- startup validation for missing environment configuration.

Generated CLI integration tests cover:

- the exact Feishu authorization URL parameters and fixed callback listener;
- state mismatch, access denial, missing code, and timeout handling;
- storage of both token types and their absolute expirations;
- `status` classification at the five-minute boundary;
- automatic refresh before a business request;
- atomic replacement with the rotated refresh token;
- the re-login error when refresh is unavailable or rejected;
- `logout` removing the local token file;
- absence of tokens and secrets from CLI output and trace logs.

Manual validation uses a real Feishu test application only after all automated
tests pass. It verifies `login`, `status`, one authenticated API request, forced
expiry followed by refresh, and `logout`.

# Semantic Command Collision Resolution

## Goal

When several OpenAPI operations in one command group simplify to the same generic command name, preserve the semantic subject from each `operationId` instead of immediately appending numeric suffixes.

For example, these operations:

- `getMis`
- `getQuality`
- `getSale`
- `getSend`

must generate:

- `mis`
- `quality`
- `sale`
- `send`

instead of `get`, `get-2`, `get-3`, and `get-4`.

## Scope

This change applies only when two or more operations in the same group collide after normal command-name simplification. Existing non-colliding names remain unchanged. Explicit `naming.operation_alias` values continue to take precedence over inferred names.

Generated projects are treated as replaceable output, so the old numbered command names will not be retained as compatibility aliases.

## Design

Planning will resolve command names in two stages:

1. Build the existing simplified candidate name for every operation.
2. Within each group, detect candidates that occur more than once.

For a colliding inferred candidate, derive a semantic fallback from the `operationId`:

- Split the identifier into normalized words.
- Remove the leading generic HTTP-style verb when present, such as `get`.
- Join the remaining words with hyphens.
- Use the existing candidate when no meaningful subject remains.

All operations in the collision set use their semantic fallback, including the first operation. This prevents the output from depending on OpenAPI traversal order.

An explicit operation alias is never replaced by a semantic fallback. If an inferred candidate collides with an explicit alias, preserve the alias and apply the semantic fallback only to the inferred operation. If multiple explicit aliases intentionally use the same name, preserve their configured base and let the final uniqueness guard append a numeric suffix.

After semantic fallbacks are selected, the existing uniqueness mechanism remains the final guard. If two semantic fallbacks still collide, numeric suffixes are allowed as the last resort.

## Examples

| Group | Operation IDs | Commands |
| --- | --- | --- |
| `analytics` | `getMis`, `getQuality`, `getSale`, `getSend` | `mis`, `quality`, `sale`, `send` |
| `pet` | `getPet` | `get` |
| `dashboard` | alias `getMis: get`, plus inferred `getQuality` | `get`, `quality` |
| `report` | `getStatus`, `getStatus_2` | `status`, `status-2` |

The `pet getPet` case demonstrates that a single non-colliding operation retains the current CLI surface.

## Implementation Boundaries

- Keep identifier parsing and semantic fallback derivation in `internal/planner/naming.go`.
- Preserve whether a candidate came from `naming.operation_alias` so collision resolution can honor explicit configuration.
- Perform group-level collision detection in `internal/planner/plan.go`, where the complete operation group context is available.
- Do not change MCP tool naming.
- Do not change explicit operation aliases.
- Do not modify generated files under `tmp/`; verify behavior by regenerating them in tests or disposable directories.

## Testing

Add planner unit coverage for:

1. A single `getPet` remains `get`.
2. `getMis`, `getQuality`, `getSale`, and `getSend` become semantic commands in one group.
3. Output is independent of operation order.
4. Explicit operation aliases retain priority.
5. A collision between an explicit alias and an inferred name changes only the inferred name.
6. Duplicate semantic fallbacks still receive a numeric suffix.
7. MCP command naming remains unchanged.

Run targeted planner tests, then `go test ./...`.

# Cross-target parameter naming design

## Problem

OpenAPI parameter names may contain nested-property and array notation such as
`dept.children[0].createBy`. The Go templates currently interpolate these names
through `pascal`, which leaves brackets in generated identifiers and produces
invalid Go source. Rust sanitizes punctuation, but collapses camel-case word
boundaries, so the two targets do not follow one naming contract.

## Naming contract

Generated identifiers treat every non-letter and non-digit as a separator. For
example, `dept.children[0].createBy` becomes a valid Go identifier such as
`DeptChildren0Createby` and a valid Rust identifier such as
`dept_children_0_createby`.

The original OpenAPI name remains the HTTP path/query/header key. Existing
target-specific CLI flag conventions remain unchanged; this fix does not reinterpret
query parameters as a JSON request body.

Identifiers must also remain valid when the source begins with a digit, contains
only punctuation, or matches a target-language keyword. Existing target-specific
fallback and keyword behavior remains in force.

## Implementation shape

The existing Go `pascal` helper is broadened to use the same punctuation boundary
rule already used by Rust. No new normalization layer or JSON-to-query adapter is
introduced.

This change is limited to parameter and body-field naming. Command/group naming
and HTTP serialization behavior are outside its scope.

## Testing

A focused generation test asserts that both targets accept nested array notation
such as `dept.children[0].createBy`, emit legal source identifiers, and preserve
the original HTTP query key. Final verification generates `tw.openapi.yaml` for
both targets, rebuilds `dist/opencli`, and repeats the user's command.

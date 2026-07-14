# Cross-target parameter naming design

## Problem

OpenAPI parameter names may contain nested-property and array notation such as
`dept.children[0].createBy`. The Go templates currently interpolate these names
through `pascal`, which leaves brackets in generated identifiers and produces
invalid Go source. Rust sanitizes punctuation, but collapses camel-case word
boundaries, so the two targets do not follow one naming contract.

## Naming contract

Parameter names are split into words at punctuation, array brackets, underscores,
dashes, whitespace, and lower-to-upper camel-case transitions. For example,
`dept.children[0].createBy` becomes the words `dept`, `children`, `0`, `create`,
`by`.

Those words are rendered per destination:

- Go identifier: `DeptChildren0CreateBy`
- Rust identifier: `dept_children_0_create_by`
- CLI flag: `dept-children-0-create-by`

The original OpenAPI name remains the HTTP path/query/header key. Sanitized names
are only used for generated source identifiers and CLI-facing flag names.

Identifiers must also remain valid when the source begins with a digit, contains
only punctuation, or matches a target-language keyword. Existing target-specific
fallback and keyword behavior remains in force.

## Implementation shape

A shared word-splitting helper defines punctuation and camel-case boundaries.
Small target-specific helpers render those words as Go PascalCase, Rust snake_case,
or CLI kebab-case. Templates call the destination-specific helper instead of using
raw parameter names for identifiers.

This change is limited to parameter and body-field naming. Command/group naming
and HTTP serialization behavior are outside its scope.

## Testing

Regression tests cover the shared word sequence and generated Go/Rust naming for:

- nested array notation (`dept.children[0].createBy`)
- repeated/nested punctuation
- leading digits
- keyword and empty-name fallbacks

Generation tests assert that both targets successfully generate the problematic
OpenAPI shape, preserve the original HTTP key, and expose matching kebab-case CLI
flags. Final verification generates `tw.openapi.yaml` for both targets, compiles
the generated projects, rebuilds `dist/opencli`, and repeats the user's command.

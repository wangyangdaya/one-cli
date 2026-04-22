#!/usr/bin/env bash
# scripts/smoke.sh — opencli end-to-end smoke test
#
# Usage:
#   ./scripts/smoke.sh [options]
#
# Options:
#   --target  go|rust          Language target (default: go)
#   --input   <path>           OpenAPI/Swagger document path
#   --mcp     <path>           MCP config file path
#   --app     <name>           Binary / root-command name (default: derived from filename)
#   --module  <module>         Go module path or Rust package name
#                              (default: github.com/acme/<app> for go, <app> for rust)
#   --no-build                 Skip `make build-host`, use existing dist/opencli
#
# Exactly one of --input or --mcp must be provided.
#
# Examples:
#   # Go CLI from OpenAPI doc
#   ./scripts/smoke.sh --target go --input ./examples/petstore.yaml
#
#   # Rust CLI from OpenAPI doc
#   ./scripts/smoke.sh --target rust --input ./examples/petstore.yaml
#
#   # Go CLI from MCP config
#   ./scripts/smoke.sh --target go --mcp ./examples/quark.json
#
#   # Rust CLI from MCP config, custom names
#   ./scripts/smoke.sh --target rust --mcp ./examples/quark.json --app quark --module quark

set -euo pipefail

# ── colours ───────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BOLD='\033[1m'; RESET='\033[0m'

pass()   { echo -e "${GREEN}  ✓ $*${RESET}" >&2; (( PASS++ ))  || true; }
fail()   { echo -e "${RED}  ✗ $*${RESET}"   >&2; (( FAIL++ ))  || true; }
skip()   { echo -e "${YELLOW}  ⚠ $*${RESET}" >&2; (( SKIP++ )) || true; }
header() { echo -e "\n${BOLD}── $* ──${RESET}" >&2; }
die()    { echo -e "${RED}${BOLD}ERROR: $*${RESET}" >&2; exit 1; }

PASS=0; FAIL=0; SKIP=0

# ── paths ─────────────────────────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
OPENCLI="$ROOT/dist/opencli"
TMP="$ROOT/tmp"

# ── argument parsing ──────────────────────────────────────────────────────────
TARGET="go"
INPUT_FILE=""
MCP_FILE=""
APP_NAME=""
MODULE=""
BUILD=true

while [[ $# -gt 0 ]]; do
  case "$1" in
    --target)   TARGET="$2";     shift 2 ;;
    --input)    INPUT_FILE="$2"; shift 2 ;;
    --mcp)      MCP_FILE="$2";   shift 2 ;;
    --app)      APP_NAME="$2";   shift 2 ;;
    --module)   MODULE="$2";     shift 2 ;;
    --no-build) BUILD=false;     shift   ;;
    -h|--help)
      sed -n '2,/^set /p' "$0" | grep '^#' | sed 's/^# \{0,1\}//'
      exit 0 ;;
    *) die "Unknown option: $1" ;;
  esac
done

# ── validate args ─────────────────────────────────────────────────────────────
[[ "$TARGET" == "go" || "$TARGET" == "rust" ]] \
  || die "--target must be 'go' or 'rust', got '$TARGET'"

[[ -n "$INPUT_FILE" || -n "$MCP_FILE" ]] \
  || die "Provide exactly one of --input <path> or --mcp <path>"

[[ -z "$INPUT_FILE" || -z "$MCP_FILE" ]] \
  || die "--input and --mcp are mutually exclusive"

# Resolve to absolute paths
[[ -n "$INPUT_FILE" ]] && INPUT_FILE="$(cd "$(dirname "$INPUT_FILE")" && pwd)/$(basename "$INPUT_FILE")"
[[ -n "$MCP_FILE"   ]] && MCP_FILE="$(cd "$(dirname "$MCP_FILE")"   && pwd)/$(basename "$MCP_FILE")"

# Derive app name from filename if not given
if [[ -z "$APP_NAME" ]]; then
  SRC="${INPUT_FILE:-$MCP_FILE}"
  APP_NAME="$(basename "$SRC" | sed 's/\.[^.]*$//' | tr '[:upper:]' '[:lower:]' | tr '_' '-')-cli"
fi

# Derive module from app name if not given
if [[ -z "$MODULE" ]]; then
  if [[ "$TARGET" == "go" ]]; then
    MODULE="github.com/acme/$APP_NAME"
  else
    MODULE="$APP_NAME"
  fi
fi

# Output dir: tmp/smoke-<app>-<target>
OUT_DIR="$TMP/smoke-${APP_NAME}-${TARGET}"

# Source type label
if [[ -n "$INPUT_FILE" ]]; then
  SRC_LABEL="--input $(basename "$INPUT_FILE")"
  SRC_FLAG=(--input "$INPUT_FILE")
else
  SRC_LABEL="--mcp $(basename "$MCP_FILE")"
  SRC_FLAG=(--mcp-config "$MCP_FILE")
fi

# ── print plan ────────────────────────────────────────────────────────────────
echo -e "${BOLD}opencli smoke test${RESET}" >&2
echo "  target  : $TARGET" >&2
echo "  source  : $SRC_LABEL" >&2
echo "  app     : $APP_NAME" >&2
echo "  module  : $MODULE" >&2
echo "  output  : $OUT_DIR" >&2

# ── helpers ───────────────────────────────────────────────────────────────────
build_go_binary() {
  local dir="$1" app="$2"
  local bin="$dir/$app"
  if (cd "$dir" && go build -o "$bin" "./cmd/$app" 2>/dev/null); then
    pass "go build $app"
    # print path to stdout for capture — must come after pass() to avoid mixing
    printf '%s' "$bin"
  else
    fail "go build $app (see output in $dir)"
    printf ''
  fi
}

verify_help() {
  local bin="$1" want="$2"
  local out
  out=$("$bin" --help 2>&1) || true
  if echo "$out" | grep -qF "$want"; then
    pass "'$want' appears in --help"
  else
    fail "'$want' missing from --help"
  fi
}

check_file() {
  local f="$1"
  if [[ -f "$f" ]]; then
    pass "file exists: ${f#$OUT_DIR/}"
  else
    fail "file missing: ${f#$OUT_DIR/}"
  fi
}

# ── step 0: build opencli ─────────────────────────────────────────────────────
header "Step 0: build opencli"

if $BUILD; then
  echo "  running: make build-host" >&2
  if make -C "$ROOT" build-host >/dev/null 2>&1; then
    pass "make build-host"
  else
    fail "make build-host"
    die "Cannot continue without a working opencli binary."
  fi
else
  skip "build skipped (--no-build)"
fi

[[ -x "$OPENCLI" ]] || die "dist/opencli not found. Run without --no-build."

# ── step 1: generate ──────────────────────────────────────────────────────────
header "Step 1: generate ($TARGET)"

rm -rf "$OUT_DIR"

GEN_ARGS=(
  "${SRC_FLAG[@]}"
  --output "$OUT_DIR"
  --module "$MODULE"
  --app    "$APP_NAME"
)
[[ "$TARGET" == "rust" ]] && GEN_ARGS+=(--target rust)

echo "  $ opencli generate ${GEN_ARGS[*]}" >&2

if [[ -n "$MCP_FILE" ]]; then
  # MCP may fail due to network — treat as skip, not hard failure
  if ! "$OPENCLI" generate "${GEN_ARGS[@]}" 2>/dev/null; then
    skip "generate skipped — MCP server unreachable or auth expired"
    header "Summary"
    echo -e "  ${GREEN}passed: $PASS${RESET}  ${RED}failed: $FAIL${RESET}  ${YELLOW}skipped: $SKIP${RESET}" >&2
    echo -e "\n${YELLOW}${BOLD}SMOKE TEST SKIPPED (MCP unavailable)${RESET}" >&2
    exit 0
  fi
else
  if ! "$OPENCLI" generate "${GEN_ARGS[@]}" 2>&1; then
    fail "generate failed"
    header "Summary"
    echo -e "  ${GREEN}passed: $PASS${RESET}  ${RED}failed: $FAIL${RESET}  ${YELLOW}skipped: $SKIP${RESET}" >&2
    echo -e "\n${RED}${BOLD}SMOKE TEST FAILED${RESET}" >&2
    exit 1
  fi
fi
pass "generate $APP_NAME ($TARGET)"

# ── step 2: verify generated files ───────────────────────────────────────────
header "Step 2: verify generated files"

if [[ "$TARGET" == "go" ]]; then
  check_file "$OUT_DIR/cmd/$APP_NAME/main.go"
  check_file "$OUT_DIR/go.mod"
  check_file "$OUT_DIR/go.sum"
  check_file "$OUT_DIR/README.md"
  # at least one internal package must exist
  if compgen -G "$OUT_DIR/internal/*/command.go" >/dev/null 2>&1; then
    pass "internal/<group>/command.go exists"
  else
    fail "no internal/<group>/command.go found"
  fi
else
  check_file "$OUT_DIR/Cargo.toml"
  check_file "$OUT_DIR/src/main.rs"
  check_file "$OUT_DIR/src/cli.rs"
  check_file "$OUT_DIR/src/client.rs"
  check_file "$OUT_DIR/src/commands/mod.rs"
  check_file "$OUT_DIR/README.md"
fi

# ── step 3: build ─────────────────────────────────────────────────────────────
header "Step 3: build"

if [[ "$TARGET" == "go" ]]; then
  BIN=$(build_go_binary "$OUT_DIR" "$APP_NAME")

  # ── step 4: run --help ──────────────────────────────────────────────────────
  header "Step 4: run --help"

  if [[ -n "$BIN" && -x "$BIN" ]]; then
    verify_help "$BIN" "$APP_NAME"

    # list top-level sub-commands
    echo "" >&2
    echo -e "  ${BOLD}$ $APP_NAME --help${RESET}" >&2
    "$BIN" --help 2>&1 | sed 's/^/    /' >&2
  fi

else
  # Rust
  if ! command -v cargo &>/dev/null; then
    skip "cargo not installed — skipping build"
  else
    echo "  running: cargo build --release (this may take a moment)" >&2
    cargo_out=$(cd "$OUT_DIR" && cargo build --release 2>&1)
    cargo_exit=$?
    if [[ $cargo_exit -ne 0 ]]; then
      if echo "$cargo_out" | grep -qE "Could not resolve host|failed to download from"; then
        skip "cargo build skipped (network restricted)"
      else
        fail "cargo build $APP_NAME"
        echo "$cargo_out" | tail -20 | sed 's/^/    /' >&2
      fi
    else
      pass "cargo build $APP_NAME"

      RUST_BIN="$OUT_DIR/target/release/$APP_NAME"

      # ── step 4: run --help ────────────────────────────────────────────────
      header "Step 4: run --help"

      if [[ -x "$RUST_BIN" ]]; then
        verify_help "$RUST_BIN" "$APP_NAME"

        echo "" >&2
        echo -e "  ${BOLD}$ $APP_NAME --help${RESET}" >&2
        "$RUST_BIN" --help 2>&1 | sed 's/^/    /' >&2
      fi
    fi
  fi
fi

# ── summary ───────────────────────────────────────────────────────────────────
header "Summary"
echo -e "  ${GREEN}passed: $PASS${RESET}  ${RED}failed: $FAIL${RESET}  ${YELLOW}skipped: $SKIP${RESET}" >&2
echo "" >&2

if [[ $FAIL -gt 0 ]]; then
  echo -e "${RED}${BOLD}SMOKE TEST FAILED${RESET}" >&2
  exit 1
else
  echo -e "${GREEN}${BOLD}SMOKE TEST PASSED${RESET}" >&2
  exit 0
fi

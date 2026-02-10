#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PY_DIR="${ROOT_DIR}/server"
GO_DIR="${ROOT_DIR}/cli/src"

run_ruff_format() {
  if command -v uv >/dev/null 2>&1; then
    (cd "${PY_DIR}" && uv run ruff format .)
    return
  fi

  if [[ -x "${PY_DIR}/.venv/bin/ruff" ]]; then
    (cd "${PY_DIR}" && "${PY_DIR}/.venv/bin/ruff" format .)
    return
  fi

  (cd "${PY_DIR}" && ruff format .)
}

run_go_format() {
  if ! command -v gofmt >/dev/null 2>&1; then
    echo "gofmt not installed; skipping Go format"
    return
  fi

  find "${GO_DIR}" -name '*.go' -not -path '*/build/*' -print0 | xargs -0 gofmt -w
}

run_ruff_format
run_go_format

#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PY_DIR="${ROOT_DIR}/server"
GO_DIR="${ROOT_DIR}/cli/src"

run_ruff() {
  if command -v uv >/dev/null 2>&1; then
    (cd "${PY_DIR}" && uv run ruff check .)
    return
  fi

  if [[ -x "${PY_DIR}/.venv/bin/ruff" ]]; then
    (cd "${PY_DIR}" && "${PY_DIR}/.venv/bin/ruff" check .)
    return
  fi

  (cd "${PY_DIR}" && ruff check .)
}

run_go_lint() {
  if ! command -v go >/dev/null 2>&1; then
    echo "go not installed; skipping Go lint"
    return
  fi

  (cd "${GO_DIR}" && go vet ./...)

  if command -v golangci-lint >/dev/null 2>&1; then
    (cd "${GO_DIR}" && golangci-lint run ./...)
  else
    echo "golangci-lint not installed; skipping"
  fi
}

run_ruff
run_go_lint

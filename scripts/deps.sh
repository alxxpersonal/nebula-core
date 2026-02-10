#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PY_DIR="${ROOT_DIR}/server"
GO_DIR="${ROOT_DIR}/cli/src"

install_python_deps() {
  if command -v uv >/dev/null 2>&1; then
    (cd "${PY_DIR}" && uv sync --all-extras --dev)
    return
  fi

  if [[ -x "${PY_DIR}/.venv/bin/python" ]]; then
    (cd "${PY_DIR}" && "${PY_DIR}/.venv/bin/pip" install -e ".[dev,test]")
    return
  fi

  echo "uv not installed and no .venv found."
  echo "Install uv or create a venv and run: pip install -e '.[dev,test]'"
  exit 1
}

install_go_deps() {
  if ! command -v go >/dev/null 2>&1; then
    echo "go not installed; skipping Go deps"
    return
  fi

  (cd "${GO_DIR}" && go mod download)
}

install_python_deps
install_go_deps

#!/usr/bin/env bash
set -euo pipefail

REPO_URL="${NEBULA_REPO_URL:-https://github.com/alxxpersonal/nebula-core.git}"
INSTALL_DIR="${NEBULA_INSTALL_DIR:-$HOME/.nebula/nebula-core}"

ROOT_DIR=""
COMPOSE_FILE=""
ENV_FILE=""
ENV_EXAMPLE=""
DATA_DIR=""

print_box() {
  local title="$1"
  local body="$2"
  printf "╭─────────────────────────────── [ %s ] ───────────────────────────────╮\n" "$title"
  while IFS= read -r line; do
    printf "│ %s\n" "$line"
  done <<<"$body"
  printf "╰───────────────────────────────────────────────────────────────────────╯\n"
}

info() {
  print_box "$1" "$2"
}

error_box() {
  print_box "Error" "$1"
}

warn_box() {
  print_box "Warning" "$1"
}

set_paths() {
  ROOT_DIR="$1"
  COMPOSE_FILE="$ROOT_DIR/docker-compose.yml"
  ENV_FILE="$ROOT_DIR/.env"
  ENV_EXAMPLE="$ROOT_DIR/.env.example"
  DATA_DIR="$ROOT_DIR/database/data"
}

require_docker() {
  if ! command -v docker >/dev/null 2>&1; then
    error_box "docker is required for nebula install.\ninstall docker desktop and rerun this command."
    exit 1
  fi

  if ! docker info >/dev/null 2>&1; then
    error_box "docker daemon is not running.\nstart docker desktop and rerun this command."
    exit 1
  fi

  if ! docker compose version >/dev/null 2>&1; then
    error_box "docker compose plugin is required.\nupdate docker desktop and rerun this command."
    exit 1
  fi
}

resolve_root_dir() {
  local script_dir
  script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

  if [[ -f "$script_dir/../docker-compose.yml" ]]; then
    set_paths "$(cd "$script_dir/.." && pwd)"
    return
  fi

  if ! command -v git >/dev/null 2>&1; then
    error_box "git is required for curl install mode.\ninstall git and rerun this command."
    exit 1
  fi

  mkdir -p "$(dirname "$INSTALL_DIR")"
  if [[ -d "$INSTALL_DIR/.git" ]]; then
    info "Install" "updating existing nebula checkout in $INSTALL_DIR"
    git -C "$INSTALL_DIR" fetch --all --prune
    git -C "$INSTALL_DIR" reset --hard origin/main
  else
    info "Install" "cloning nebula-core into $INSTALL_DIR"
    rm -rf "$INSTALL_DIR"
    git clone --depth 1 "$REPO_URL" "$INSTALL_DIR"
  fi

  set_paths "$INSTALL_DIR"
}

prepare_env() {
  if [[ ! -f "$ENV_FILE" ]]; then
    if [[ -f "$ENV_EXAMPLE" ]]; then
      cp "$ENV_EXAMPLE" "$ENV_FILE"
      info "Config" "created .env from .env.example"
    else
      error_box "missing .env and .env.example in $ROOT_DIR."
      exit 1
    fi
  fi
}

start_stack() {
  mkdir -p "$DATA_DIR"
  chmod 700 "$DATA_DIR" 2>/dev/null || true

  if [[ "${NEBULA_SKIP_PULL:-0}" == "1" ]]; then
    warn_box "skipping image pull because NEBULA_SKIP_PULL=1"
  else
    info "Install" "pulling container images"
    if ! docker compose -f "$COMPOSE_FILE" pull; then
      warn_box "image pull failed, continuing with local/build cache."
    fi
  fi

  info "Install" "starting nebula services"
  docker compose -f "$COMPOSE_FILE" up -d --build
}

wait_for_postgres() {
  local attempts=60
  local i
  for ((i=1; i<=attempts; i++)); do
    local status
    status="$(docker inspect --format='{{.State.Health.Status}}' nebula-core 2>/dev/null || true)"
    if [[ "$status" == "healthy" ]]; then
      info "Success" $'postgres is healthy on port 6432\nadminer is available on http://localhost:8080\nrepo path: '"$ROOT_DIR"
      return 0
    fi
    sleep 2
  done

  error_box $'postgres did not become healthy in time.\ncheck logs with: docker compose -f '"$COMPOSE_FILE"$' logs postgres'
  exit 1
}

main() {
  require_docker
  resolve_root_dir
  info "Nebula" "starting installer from $ROOT_DIR"
  prepare_env
  start_stack
  wait_for_postgres
}

main "$@"

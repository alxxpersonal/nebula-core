#!/bin/bash
: '
Nebula Database Migration Script

Runs all SQL migrations in order against the running postgres container.

Usage:
    ./scripts/migrate.sh          run all migrations
    ./scripts/migrate.sh 002      run specific migration (002_*.sql)
    ./scripts/migrate.sh --fresh  nuke db and rebuild from scratch
    ./scripts/migrate.sh --help   show this message
'

set -e

if [[ "$1" == "--help" || "$1" == "-h" ]]; then
    sed -n "3,11p" "$0"
    exit 0
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
MIGRATIONS_DIR="$PROJECT_DIR/database/migrations"
CONTAINER="nebula-core"
DB_USER="nebula"
DB_NAME="nebula"

# --- Colors ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # no color

# --- Header ---
echo -e "${YELLOW}Nebula Migration Script${NC}"
echo "================================"

# --- Check if container is running ---
if ! docker ps --format '{{.Names}}' | grep -q "^${CONTAINER}$"; then
    echo -e "${RED}Error: Container '$CONTAINER' is not running${NC}"
    echo "Run: docker compose up -d"
    exit 1
fi

# --- Fresh rebuild ---
if [[ "$1" == "--fresh" ]]; then
    echo -e "${YELLOW}Nuking database and rebuilding...${NC}"
    cd "$PROJECT_DIR"
    docker compose down -v
    docker compose up -d
    echo -e "${GREEN}Done. Fresh db with all migrations applied.${NC}"
    exit 0
fi

# --- Run specific migration ---
if [[ -n "$1" ]]; then
    MIGRATION=$(ls "$MIGRATIONS_DIR"/${1}*.sql 2>/dev/null | head -1)
    if [[ -z "$MIGRATION" ]]; then
        echo -e "${RED}Error: No migration matching '$1'${NC}"
        exit 1
    fi
    echo -e "Running: ${YELLOW}$(basename "$MIGRATION")${NC}"
    docker exec -i "$CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" < "$MIGRATION"
    echo -e "${GREEN}Done.${NC}"
    exit 0
fi

# --- Run all migrations ---
echo "Running all migrations..."
for f in "$MIGRATIONS_DIR"/*.sql; do
    if [[ -f "$f" ]]; then
        echo -e "  -> ${YELLOW}$(basename "$f")${NC}"
        docker exec -i "$CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" < "$f" 2>&1 | grep -v "already exists" || true
    fi
done

echo -e "${GREEN}All migrations complete.${NC}"

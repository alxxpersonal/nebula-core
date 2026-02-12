# 2026-02-12 Semantic Search Implementation

## Scope

Implemented semantic search end to end for entities and knowledge across:

- REST API (`/api/search/semantic`)
- MCP tool (`semantic_search`)
- CLI Search tab (text/semantic mode toggle)

## Server

- Added deterministic semantic ranking utility:
  - `server/src/nebula_mcp/semantic.py`
  - hash-based embedding + cosine similarity
  - stable sorting and score filtering
- Added candidate SQL queries:
  - `server/src/queries/search/entities_semantic_candidates.sql`
  - `server/src/queries/search/knowledge_semantic_candidates.sql`
- Added REST route:
  - `server/src/nebula_api/routes/search.py`
  - `POST /api/search/semantic`
  - supports `kinds`, `limit`, `candidate_limit`
  - scope-aware filtering (user callers constrained to public)
- Wired route into app:
  - `server/src/nebula_api/app.py`

## MCP

- Added model:
  - `SemanticSearchInput` in `server/src/nebula_mcp/models.py`
- Added tool:
  - `semantic_search` in `server/src/nebula_mcp/server.py`
- Reused same candidate queries and ranker as REST path.

## CLI

- Added API client method and type:
  - `cli/src/internal/api/search.go`
  - `SemanticSearchResult`
  - `Client.SemanticSearch(...)`
- Updated Search UI:
  - `cli/src/internal/ui/search.go`
  - Mode toggle via `tab` (`text` / `semantic`)
  - Semantic mode calls `/api/search/semantic`
  - Selection loads full entity/knowledge/job detail on demand
- Updated hint bar:
  - `cli/src/internal/ui/app.go`

## Tests Added

- REST:
  - `server/tests/api/test_semantic_search.py`
    - happy path result retrieval
    - privacy/scope enforcement
    - invalid payload rejection
- MCP:
  - `server/tests/integration/test_semantic_search.py`
    - happy path
    - scope enforcement
    - invalid query validation
- CLI:
  - `cli/src/internal/api/search_test.go`
  - `cli/src/internal/ui/search_test.go` (semantic mode + detail fetch path)

## Validation Commands

- `cd server && uv run pytest tests/api/test_semantic_search.py tests/integration/test_semantic_search.py -q`
- `cd cli/src && go test ./internal/api ./internal/ui -count=1`
- `./scripts/lint.sh`


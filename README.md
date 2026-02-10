# Nebula Core

Just as developers need GitHub to collaborate on code, agents need Nebula to collaborate on context. It's the shared layer where swarms of agents build, track, and modify your life's state together without overwriting each other.

## What it is

- Shared memory for agents backed by Postgres, not prompts
- Privacy scopes enforced at query time and in context segments
- Approvals for untrusted agents, with audit trails for every mutation
- A polymorphic relationship graph across all node types

## Core guarantees

- Privacy is enforced by schema and query filters, not conventions
- Untrusted agents can only write through approvals
- All writes are captured in an immutable audit log
- Relationships are typed and validated, with symmetric types auto synced

## Data model

- Entities: people, orgs, tools, projects
- Knowledge items: articles, notes, videos, papers
- Relationships: polymorphic graph edges with properties
- Jobs: tasks with priorities and subtasks
- Logs: structured events and notes
- Files: file metadata with links to entities or knowledge
- Protocols: system rules stored in the database
- Agents: identities, scopes, and capabilities
- Approvals: gatekeeper for untrusted actions

## Architecture

- Database: Postgres 16 with pgvector
- Server: FastAPI REST plus MCP tools
- CLI: Go, Bubble Tea, Lip Gloss
- SQL: parameterized query files, not inline SQL

## Quickstart

### Database

1. Create a `.env` in `nebula-core/` with database settings.
2. Run the database:

```bash
cd nebula-core

docker compose up -d
```

### REST API

```bash
cd nebula-core/server

.venv/bin/uvicorn nebula_api.app:app --reload --port 8000
```

### MCP Server

```bash
cd nebula-core/server

.venv/bin/python -m nebula_mcp.server
```

### CLI

```bash
cd nebula-core/cli/src

go build -o build/nebula ./cmd/nebula

./build/nebula
```

## Dev tooling

```bash
cd nebula-core

# install deps (python via uv when available, go via go mod)
make deps

# format
make format

# lint
make lint
```

Notes:
- Python linting and formatting use Ruff.
- Go linting uses `go vet`, with `golangci-lint` if installed.

## Status

- Embedding generation and semantic search pipelines are not wired yet.
- Most CRUD, approvals, audit, and graph operations are live.

## Repo structure

```
nebula-core/
├── database/
│   └── migrations/
├── scripts/
│   └── migrate.sh
├── server/
│   ├── src/
│   │   ├── nebula_mcp/
│   │   ├── nebula_api/
│   │   └── queries/
│   ├── tests/
│   └── pyproject.toml
├── cli/
│   └── src/
├── docker-compose.yml
└── README.md
```

## MCP toolset

The MCP server exposes tools across entities, knowledge, relationships, jobs, files, protocols, approvals, and audit. See `server/src/nebula_mcp/server.py` and the SQL under `server/src/queries/` for the full list.

## License

MIT

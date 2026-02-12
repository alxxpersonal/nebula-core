"""Semantic search API routes."""

# Standard Library
import json
from pathlib import Path
from typing import Any

# Third-Party
from fastapi import APIRouter, Depends, Request
from pydantic import BaseModel, Field, field_validator

# Local
from nebula_api.auth import require_auth
from nebula_api.response import success
from nebula_mcp.query_loader import QueryLoader
from nebula_mcp.semantic import rank_semantic_candidates

QUERIES = QueryLoader(Path(__file__).resolve().parents[2] / "queries")

router = APIRouter()
ALLOWED_SEMANTIC_KINDS = {"entity", "knowledge"}
DEFAULT_SEMANTIC_KINDS = ["entity", "knowledge"]


class SemanticSearchBody(BaseModel):
    """Semantic search request payload."""

    query: str = Field(..., min_length=2, max_length=512)
    kinds: list[str] = Field(default_factory=lambda: list(DEFAULT_SEMANTIC_KINDS))
    limit: int = Field(default=20, ge=1, le=100)
    candidate_limit: int = Field(default=250, ge=50, le=2000)

    @field_validator("query", mode="before")
    @classmethod
    def _clean_query(cls, value: str) -> str:
        return str(value or "").strip()

    @field_validator("kinds", mode="before")
    @classmethod
    def _clean_kinds(cls, value: list[str] | None) -> list[str]:
        if not value:
            return list(DEFAULT_SEMANTIC_KINDS)
        out: list[str] = []
        for item in value:
            name = str(item or "").strip().lower()
            if name and name in ALLOWED_SEMANTIC_KINDS and name not in out:
                out.append(name)
        return out or list(DEFAULT_SEMANTIC_KINDS)


def _scope_filter_ids(auth: dict, enums: Any) -> list[str] | None:
    if auth.get("caller_type") == "user":
        public_id = enums.scopes.name_to_id.get("public")
        return [public_id] if public_id else []
    scopes = auth.get("scopes", [])
    return scopes if scopes else []


def _entity_candidate(row: dict[str, Any]) -> dict[str, Any]:
    metadata = row.get("metadata") or {}
    tags = row.get("tags") or []
    text = " ".join(
        [
            str(row.get("name", "")),
            str(row.get("type", "")),
            " ".join(str(t) for t in tags),
            json.dumps(metadata, sort_keys=True),
        ]
    ).strip()
    subtitle = str(row.get("type", "") or "entity")
    snippet_parts = [subtitle]
    if tags:
        snippet_parts.append(", ".join(str(t) for t in tags[:3]))
    return {
        "kind": "entity",
        "id": str(row.get("id", "")),
        "title": str(row.get("name", "")),
        "subtitle": subtitle,
        "snippet": " · ".join(part for part in snippet_parts if part),
        "text": text,
    }


def _knowledge_candidate(row: dict[str, Any]) -> dict[str, Any]:
    metadata = row.get("metadata") or {}
    tags = row.get("tags") or []
    content = str(row.get("content") or "")
    text = " ".join(
        [
            str(row.get("title", "")),
            str(row.get("source_type", "")),
            content,
            " ".join(str(t) for t in tags),
            json.dumps(metadata, sort_keys=True),
        ]
    ).strip()
    subtitle = str(row.get("source_type", "") or "knowledge")
    snippet_base = content.strip().replace("\n", " ")
    if len(snippet_base) > 120:
        snippet_base = snippet_base[:120].rstrip() + "..."
    snippet_parts = [subtitle]
    if snippet_base:
        snippet_parts.append(snippet_base)
    return {
        "kind": "knowledge",
        "id": str(row.get("id", "")),
        "title": str(row.get("title", "")),
        "subtitle": subtitle,
        "snippet": " · ".join(part for part in snippet_parts if part),
        "text": text,
    }


@router.post("/semantic")
async def semantic_search(
    payload: SemanticSearchBody,
    request: Request,
    auth: dict = Depends(require_auth),
) -> dict[str, Any]:
    """Run semantic search across entities and knowledge with scope filtering."""

    pool = request.app.state.pool
    enums = request.app.state.enums
    scope_ids = _scope_filter_ids(auth, enums)
    candidates: list[dict[str, Any]] = []

    if "entity" in payload.kinds:
        rows = await pool.fetch(
            QUERIES["search/entities_semantic_candidates"],
            scope_ids,
            payload.candidate_limit,
        )
        candidates.extend(_entity_candidate(dict(row)) for row in rows)

    if "knowledge" in payload.kinds:
        rows = await pool.fetch(
            QUERIES["search/knowledge_semantic_candidates"],
            scope_ids,
            payload.candidate_limit,
        )
        candidates.extend(_knowledge_candidate(dict(row)) for row in rows)

    ranked = rank_semantic_candidates(payload.query, candidates, limit=payload.limit)
    for item in ranked:
        item.pop("text", None)
    return success(ranked)

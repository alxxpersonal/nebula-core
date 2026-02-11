"""Red team tests for approval queue validation gaps."""

# Standard Library
import json

# Third-Party
import pytest

# Local
from nebula_mcp.models import (
    BulkImportInput,
    CreateEntityInput,
    CreateKnowledgeInput,
    CreateRelationshipInput,
    RevertEntityInput,
)
from nebula_mcp.server import (
    bulk_import_entities,
    create_entity,
    create_knowledge,
    create_relationship,
    revert_entity,
)


@pytest.mark.asyncio
@pytest.mark.xfail(reason="revert should validate audit id before approval")
async def test_revert_entity_rejects_invalid_audit_id(
    test_entity, untrusted_mcp_context
):
    """Invalid audit ids should be rejected before approval queue."""

    payload = RevertEntityInput(
        entity_id=str(test_entity["id"]),
        audit_id="fake-audit-id-12345",
    )

    with pytest.raises(ValueError):
        await revert_entity(payload, untrusted_mcp_context)


@pytest.mark.asyncio
@pytest.mark.xfail(reason="relationship creation should validate node existence")
async def test_create_relationship_rejects_missing_nodes(
    enums, test_entity, untrusted_mcp_context
):
    """Relationships to missing nodes should be rejected before approval."""

    payload = CreateRelationshipInput(
        source_type="entity",
        source_id=str(test_entity["id"]),
        target_type="entity",
        target_id="00000000-0000-0000-0000-000000000001",
        relationship_type="related-to",
        properties={"note": "bad target"},
    )

    with pytest.raises(ValueError):
        await create_relationship(payload, untrusted_mcp_context)


@pytest.mark.asyncio
@pytest.mark.xfail(reason="vault file paths should be sanitized")
async def test_create_entity_rejects_path_traversal(mock_mcp_context):
    """Entities should reject vault file paths outside vault root."""

    payload = CreateEntityInput(
        name="Path Traversal",
        type="person",
        status="active",
        scopes=["public"],
        tags=["test"],
        metadata={},
        vault_file_path="../../../../etc/passwd",
    )

    with pytest.raises(ValueError):
        await create_entity(payload, mock_mcp_context)


@pytest.mark.asyncio
@pytest.mark.xfail(reason="knowledge URLs should reject javascript scheme")
async def test_create_knowledge_rejects_javascript_url(mock_mcp_context):
    """Knowledge URLs should be restricted to http and https."""

    payload = CreateKnowledgeInput(
        title="Bad URL",
        url="javascript:alert('xss')",
        source_type="article",
        content="x",
        status="active",
        scopes=["public"],
        tags=["test"],
        metadata={},
    )

    with pytest.raises(ValueError):
        await create_knowledge(payload, mock_mcp_context)


@pytest.mark.asyncio
@pytest.mark.xfail(reason="metadata should block prototype pollution keys")
async def test_create_entity_rejects_proto_pollution(mock_mcp_context):
    """Entities should reject prototype pollution keys in metadata."""

    payload = CreateEntityInput(
        name="Proto",
        type="person",
        status="active",
        scopes=["public"],
        tags=["test"],
        metadata={"__proto__": {"isAdmin": True}},
    )

    with pytest.raises(ValueError):
        await create_entity(payload, mock_mcp_context)


@pytest.mark.asyncio
@pytest.mark.xfail(reason="bulk import should require per item approval")
async def test_bulk_import_requires_per_item_approval(
    db_pool, untrusted_mcp_context
):
    """Bulk imports should not collapse into a single approval."""

    payload = BulkImportInput(
        items=[
            {
                "name": "Alpha",
                "type": "person",
                "status": "active",
                "scopes": ["public"],
                "tags": ["test"],
                "metadata": {},
            },
            {
                "name": "Beta",
                "type": "person",
                "status": "active",
                "scopes": ["public"],
                "tags": ["test"],
                "metadata": {},
            },
        ]
    )

    await bulk_import_entities(payload, untrusted_mcp_context)

    count = await db_pool.fetchval("SELECT COUNT(*) FROM approval_requests")
    assert count >= 2

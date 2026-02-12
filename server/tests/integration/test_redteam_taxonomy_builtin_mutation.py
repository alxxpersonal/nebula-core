"""Red team tests for taxonomy built-in mutation and reserved name reuse (MCP)."""

# Third-Party
import pytest

from nebula_mcp.models import (
    CreateTaxonomyInput,
    ListTaxonomyInput,
    UpdateTaxonomyInput,
)
from nebula_mcp.server import (
    create_taxonomy,
    list_taxonomy,
    update_taxonomy,
)

pytestmark = pytest.mark.integration


@pytest.mark.xfail(
    reason="built-in scope names are security boundaries and should be immutable/reserved"
)
async def test_mcp_taxonomy_builtin_scope_name_is_immutable(mock_mcp_context, db_pool):
    """Renaming built-in scopes should be rejected (prevents reserved-name reuse)."""

    rows = await list_taxonomy(
        ListTaxonomyInput(kind="scopes", search="admin"),
        mock_mcp_context,
    )
    admin_scope = next((r for r in rows if r["name"] == "admin"), None)
    assert admin_scope is not None
    assert admin_scope["is_builtin"] is True

    builtin_id = str(admin_scope["id"])
    renamed_name = "admin-rt-renamed"
    created_id: str | None = None
    updated: dict | None = None

    try:
        try:
            updated = await update_taxonomy(
                UpdateTaxonomyInput(
                    kind="scopes",
                    item_id=builtin_id,
                    name=renamed_name,
                ),
                mock_mcp_context,
            )
        except Exception:
            updated = None

        # If rename succeeds, reserved name becomes reusable. Show that too.
        if updated is not None:
            try:
                created = await create_taxonomy(
                    CreateTaxonomyInput(
                        kind="scopes",
                        name="admin",
                        description="rt reserved name reuse",
                        metadata={},
                    ),
                    mock_mcp_context,
                )
            except Exception:
                created = None
            if created is not None:
                created_id = str(created["id"])

        assert updated is None
    finally:
        # Taxonomy tables are not truncated by the global test cleanup fixture.
        if created_id is not None:
            await db_pool.execute(
                "DELETE FROM privacy_scopes WHERE id = $1::uuid",
                created_id,
            )
        if updated is not None:
            await db_pool.execute(
                "UPDATE privacy_scopes SET name = $2 WHERE id = $1::uuid",
                builtin_id,
                "admin",
            )

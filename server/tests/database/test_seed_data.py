"""Database tests for seed data verification."""

# Third-Party
import pytest

pytestmark = pytest.mark.database


# --- Statuses ---


@pytest.mark.asyncio
@pytest.mark.parametrize(
    "status_name",
    [
        "active",
        "in-progress",
        "planning",
        "on-hold",
        "completed",
        "abandoned",
        "replaced",
        "deleted",
        "inactive",
    ],
)
async def test_status_exists(db_pool, status_name):
    """Each expected status exists in the statuses table."""

    row = await db_pool.fetchrow("SELECT * FROM statuses WHERE name = $1", status_name)
    assert row is not None, f"Status '{status_name}' not found"


@pytest.mark.asyncio
@pytest.mark.parametrize(
    "status_name",
    ["active", "in-progress", "planning", "on-hold"],
)
async def test_active_statuses_have_active_category(db_pool, status_name):
    """Active statuses have category = 'active'."""

    row = await db_pool.fetchrow(
        "SELECT category FROM statuses WHERE name = $1", status_name
    )
    assert row["category"] == "active"


@pytest.mark.asyncio
@pytest.mark.parametrize(
    "status_name",
    ["completed", "abandoned", "replaced", "deleted", "inactive"],
)
async def test_archived_statuses_have_archived_category(db_pool, status_name):
    """Archived statuses have category = 'archived'."""

    row = await db_pool.fetchrow(
        "SELECT category FROM statuses WHERE name = $1", status_name
    )
    assert row["category"] == "archived"


# --- Privacy Scopes ---


@pytest.mark.asyncio
@pytest.mark.parametrize(
    "scope_name",
    [
        "public",
        "personal",
        "vault-only",
        "uni",
        "code",
        "health",
        "social",
        "sensitive",
        "blacklisted",
    ],
)
async def test_scope_exists(db_pool, scope_name):
    """Each expected privacy scope exists in the privacy_scopes table."""

    row = await db_pool.fetchrow(
        "SELECT * FROM privacy_scopes WHERE name = $1", scope_name
    )
    assert row is not None, f"Scope '{scope_name}' not found"


# --- Entity Types ---


@pytest.mark.asyncio
@pytest.mark.parametrize(
    "type_name",
    [
        "person",
        "project",
        "tool",
        "organization",
        "course",
        "idea",
        "framework",
        "paper",
        "university",
    ],
)
async def test_entity_type_exists(db_pool, type_name):
    """Each expected entity type exists in the entity_types table."""

    row = await db_pool.fetchrow(
        "SELECT * FROM entity_types WHERE name = $1", type_name
    )
    assert row is not None, f"Entity type '{type_name}' not found"


# --- Log Types ---


@pytest.mark.asyncio
@pytest.mark.parametrize(
    "log_type_name",
    [
        "gym-session",
        "weight",
        "mood",
        "sleep",
        "calories",
        "workout",
        "meditation",
        "reading",
        "water-intake",
    ],
)
async def test_log_type_exists(db_pool, log_type_name):
    """Each expected log type exists in the log_types table."""

    row = await db_pool.fetchrow(
        "SELECT * FROM log_types WHERE name = $1", log_type_name
    )
    assert row is not None, f"Log type '{log_type_name}' not found"


# --- Relationship Types ---


@pytest.mark.asyncio
@pytest.mark.parametrize(
    "rel_type_name",
    [
        "friends-with",
        "dating",
        "inner-circle",
        "related-to",
        "classmates-with",
        "roommates-with",
        "colleagues-with",
        "confidant",
        "acquaintance",
        "partners-with",
        "groupmates-with",
        "gym-partner",
        "minecraft-friend",
        "discord-friend",
    ],
)
async def test_symmetric_relationship_type(db_pool, rel_type_name):
    """Each symmetric relationship type has is_symmetric = True."""

    row = await db_pool.fetchrow(
        "SELECT is_symmetric FROM relationship_types WHERE name = $1",
        rel_type_name,
    )
    assert row is not None, f"Relationship type '{rel_type_name}' not found"
    assert row["is_symmetric"] is True


@pytest.mark.asyncio
async def test_works_on_not_symmetric(db_pool):
    """The works-on relationship type has is_symmetric = False."""

    row = await db_pool.fetchrow(
        "SELECT is_symmetric FROM relationship_types WHERE name = 'works-on'"
    )
    assert row is not None
    assert row["is_symmetric"] is False


@pytest.mark.asyncio
async def test_total_relationship_types_at_least_45(db_pool):
    """The relationship_types table contains at least 45 seeded types."""

    count = await db_pool.fetchval("SELECT COUNT(*) FROM relationship_types")
    assert count >= 44

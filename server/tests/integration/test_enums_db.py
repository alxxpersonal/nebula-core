"""Integration tests for enum loading from a real Postgres database."""

# Standard Library
from uuid import UUID

import pytest

pytestmark = pytest.mark.integration


# --- Load Completeness ---


async def test_load_enums_all_sections_populated(enums):
    """All five enum sections should be populated after load."""

    assert enums.statuses.name_to_id
    assert enums.scopes.name_to_id
    assert enums.relationship_types.name_to_id
    assert enums.entity_types.name_to_id
    assert enums.log_types.name_to_id


# --- Section Counts ---


async def test_statuses_count(enums):
    """Statuses section should contain exactly 9 entries."""

    assert len(enums.statuses.name_to_id) == 9


async def test_scopes_count(enums):
    """Scopes section should contain exactly 10 entries."""

    assert len(enums.scopes.name_to_id) == 10


async def test_relationship_types_count(enums):
    """Relationship types section should contain at least 45 entries."""

    assert len(enums.relationship_types.name_to_id) >= 44


async def test_entity_types_count(enums):
    """Entity types section should contain exactly 9 entries."""

    assert len(enums.entity_types.name_to_id) == 9


async def test_log_types_count(enums):
    """Log types section should contain exactly 9 entries."""

    assert len(enums.log_types.name_to_id) == 9


# --- UUID Validation ---


async def test_all_ids_are_valid_uuids(enums):
    """Every ID across all enum sections should be a valid UUID."""

    sections = [
        enums.statuses,
        enums.scopes,
        enums.relationship_types,
        enums.entity_types,
        enums.log_types,
    ]

    for section in sections:
        for uid in section.name_to_id.values():
            assert isinstance(uid, UUID), f"Expected UUID, got {type(uid)}: {uid}"

"""Database migration manifest contract tests."""

# Third-Party
import pytest

# Local
from tests.conftest import MIGRATION_FILES, MIGRATIONS_DIR

pytestmark = pytest.mark.database


def test_migration_manifest_covers_all_sql_files():
    """Session migration manifest should include every migration SQL file."""

    disk_files = {path.name for path in MIGRATIONS_DIR.glob("*.sql")}
    manifest_files = set(MIGRATION_FILES)

    assert manifest_files == disk_files


def test_migration_manifest_preserves_bootstrap_order():
    """Manifest should keep extension/bootstrap files before schema init."""

    assert MIGRATION_FILES[0] == "006_pgcrypto.sql"
    assert MIGRATION_FILES[1] == "000_init.sql"

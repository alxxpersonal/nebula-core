"""Database test fixtures."""

# Standard Library
# Third-Party
import sys
from pathlib import Path

SRC_DIR = Path(__file__).resolve().parents[2] / "src"
sys.path.insert(0, str(SRC_DIR))

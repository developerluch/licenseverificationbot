# Base scraper interface for state DOI license lookups
from __future__ import annotations

import logging
from dataclasses import dataclass, field
from typing import Optional

logger = logging.getLogger(__name__)


@dataclass
class LicenseResult:
    """Standardized result from any state DOI lookup."""

    found: bool = False
    active: bool = False
    full_name: str = ""
    license_number: str = ""
    npn: str = ""
    state: str = ""
    license_type: str = ""          # e.g. "Life & Annuity", "Health"
    status: str = ""                # e.g. "Valid", "Invalid", "Expired"
    expiration_date: str = ""       # ISO date or state format
    issue_date: str = ""
    resident: bool = False
    appointments: list[str] = field(default_factory=list)
    raw_data: dict = field(default_factory=dict)
    error: Optional[str] = None

    @property
    def is_life_licensed(self) -> bool:
        """Check if this license covers life insurance."""
        life_keywords = ["life", "life & annuity", "life and annuity", "life/annuity"]
        lic_type = self.license_type.lower()
        return any(kw in lic_type for kw in life_keywords) and self.active


class StateScraper:
    """Base class for state DOI scrapers. Subclass per state."""

    STATE_CODE: str = ""
    STATE_NAME: str = ""
    LOOKUP_URL: str = ""

    async def lookup_by_name(
        self, first_name: str, last_name: str
    ) -> list[LicenseResult]:
        """Search by first + last name. Returns list of matches."""
        raise NotImplementedError

    async def lookup_by_npn(self, npn: str) -> list[LicenseResult]:
        """Search by NPN. Returns list of matches (usually 1)."""
        raise NotImplementedError

    async def lookup_by_license_number(
        self, license_number: str
    ) -> list[LicenseResult]:
        """Search by state license number."""
        raise NotImplementedError

    async def close(self) -> None:
        """Clean up resources (browser, session, etc.)."""
        pass

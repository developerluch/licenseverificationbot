# NAIC SOLAR — Fallback scraper for states without dedicated scrapers
# Source: https://sbs.naic.org/solar-external-lookup/
#
# NAIC SOLAR is a national database that covers ALL states.
# Slower and less detailed than state-specific lookups, but works everywhere.
# Supports: first name + last name + optional state filter.

import logging
from typing import Optional

import aiohttp
from bs4 import BeautifulSoup

from scrapers.base import LicenseResult, StateScraper

logger = logging.getLogger(__name__)

SOLAR_URL = "https://sbs.naic.org/solar-external-lookup/"


class NAICScraper(StateScraper):
    """Fallback scraper using NAIC SOLAR for any state."""

    STATE_CODE = "ALL"
    STATE_NAME = "NAIC (National)"
    LOOKUP_URL = SOLAR_URL

    def __init__(self, state_code: str = ""):
        self._state_code = state_code
        self._session: Optional[aiohttp.ClientSession] = None

    async def _get_session(self) -> aiohttp.ClientSession:
        if self._session is None or self._session.closed:
            self._session = aiohttp.ClientSession(
                headers={
                    "User-Agent": (
                        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "
                        "AppleWebKit/537.36 (KHTML, like Gecko) "
                        "Chrome/120.0.0.0 Safari/537.36"
                    ),
                }
            )
        return self._session

    async def lookup_by_name(
        self, first_name: str, last_name: str
    ) -> list[LicenseResult]:
        """Search NAIC SOLAR by name."""
        try:
            session = await self._get_session()

            # First GET to establish session/cookies
            async with session.get(SOLAR_URL) as resp:
                if resp.status != 200:
                    return [LicenseResult(error=f"HTTP {resp.status}")]

            payload = {
                "firstName": first_name.strip(),
                "lastName": last_name.strip(),
            }
            if self._state_code:
                payload["state"] = self._state_code.upper()

            async with session.post(SOLAR_URL, data=payload) as resp:
                if resp.status != 200:
                    return [LicenseResult(error=f"HTTP {resp.status}")]
                html = await resp.text()

            return self._parse_results(html)

        except Exception as e:
            logger.error(f"NAIC SOLAR lookup error: {e}")
            return [LicenseResult(error=str(e))]

    async def lookup_by_npn(self, npn: str) -> list[LicenseResult]:
        """NAIC SOLAR doesn't support direct NPN search via the web form."""
        logger.warning("NAIC SOLAR web form doesn't support NPN search")
        return [LicenseResult(found=False, error="Use name search for NAIC")]

    async def lookup_by_license_number(
        self, license_number: str
    ) -> list[LicenseResult]:
        """NAIC SOLAR doesn't support license number search."""
        return [LicenseResult(found=False, error="Use name search for NAIC")]

    def _parse_results(self, html: str) -> list[LicenseResult]:
        """Parse NAIC SOLAR results."""
        soup = BeautifulSoup(html, "html.parser")
        results = []

        table = soup.find("table", class_="results") or soup.find("table")
        if not table:
            if "no results" in html.lower() or "not found" in html.lower():
                return [LicenseResult(found=False)]
            return [LicenseResult(found=False)]

        rows = table.find_all("tr")
        if len(rows) < 2:
            return [LicenseResult(found=False)]

        for row in rows[1:]:  # Skip header
            cells = row.find_all("td")
            if len(cells) < 3:
                continue

            result = LicenseResult(
                found=True,
                state=self._state_code or "",
            )

            try:
                result.npn = cells[0].get_text(strip=True)
                result.full_name = cells[1].get_text(strip=True)
                if len(cells) > 2:
                    result.state = cells[2].get_text(strip=True)
                if len(cells) > 3:
                    result.license_number = cells[3].get_text(strip=True)

                # NAIC doesn't always show status in search results
                # Mark as active if found (needs detail check for confirmation)
                result.active = True
                result.status = "Found in NAIC"

            except (IndexError, AttributeError) as e:
                logger.warning(f"NAIC parse error: {e}")
                continue

            results.append(result)

        return results if results else [LicenseResult(found=False)]

    async def close(self) -> None:
        if self._session and not self._session.closed:
            await self._session.close()

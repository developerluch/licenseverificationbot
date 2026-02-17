# California Department of Insurance — License Lookup Scraper
# Source: https://cdicloud.insurance.ca.gov/cal/
#
# CA CDI supports: name search (min 2 chars last name) or license number.
# Returns: license status, discipline history, lines of authority.
# JavaScript-heavy — uses aiohttp for the backend API calls.

import logging
from typing import Optional

import aiohttp
from bs4 import BeautifulSoup

from scrapers.base import LicenseResult, StateScraper

logger = logging.getLogger(__name__)

BASE_URL = "https://cdicloud.insurance.ca.gov/cal"
NAME_SEARCH_URL = f"{BASE_URL}/IndividualNameSearch"
LICENSE_SEARCH_URL = f"{BASE_URL}/LicenseNumberSearch"


class CaliforniaScraper(StateScraper):
    STATE_CODE = "CA"
    STATE_NAME = "California"
    LOOKUP_URL = BASE_URL

    def __init__(self):
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
        """Search CA CDI by name."""
        try:
            session = await self._get_session()

            # CA CDI uses form POST for name search
            data = {
                "LastName": last_name.strip(),
                "FirstName": first_name.strip(),
            }

            async with session.post(NAME_SEARCH_URL, data=data) as resp:
                if resp.status != 200:
                    logger.error(f"CA lookup failed: HTTP {resp.status}")
                    return [LicenseResult(error=f"HTTP {resp.status}")]
                html = await resp.text()

            return self._parse_results(html)

        except Exception as e:
            logger.error(f"CA name lookup error: {e}")
            return [LicenseResult(error=str(e))]

    async def lookup_by_license_number(
        self, license_number: str
    ) -> list[LicenseResult]:
        """Search CA CDI by license number."""
        try:
            session = await self._get_session()
            data = {"LicenseNumber": license_number.strip()}
            async with session.post(LICENSE_SEARCH_URL, data=data) as resp:
                if resp.status != 200:
                    return [LicenseResult(error=f"HTTP {resp.status}")]
                html = await resp.text()
            return self._parse_results(html)
        except Exception as e:
            logger.error(f"CA license# lookup error: {e}")
            return [LicenseResult(error=str(e))]

    async def lookup_by_npn(self, npn: str) -> list[LicenseResult]:
        """CA CDI doesn't directly support NPN search — fall back to empty."""
        logger.warning("CA CDI does not support NPN lookup — use name or license #")
        return [LicenseResult(found=False, error="CA does not support NPN lookup")]

    def _parse_results(self, html: str) -> list[LicenseResult]:
        """Parse CA CDI search results."""
        soup = BeautifulSoup(html, "html.parser")
        results = []

        # Look for results table
        table = soup.find("table", {"id": "gridResult"}) or soup.find("table", class_="table")
        if not table:
            if "no data" in html.lower() or "no results" in html.lower():
                return [LicenseResult(found=False)]
            # Try alternate table structures
            tables = soup.find_all("table")
            for t in tables:
                if t.find("tr"):
                    table = t
                    break

        if not table:
            return [LicenseResult(found=False)]

        rows = table.find_all("tr")[1:]  # Skip header
        for row in rows:
            cells = row.find_all("td")
            if len(cells) < 2:
                continue

            result = LicenseResult(found=True, state="CA")

            try:
                # CA typically shows: License # | Name | Status | Type
                result.license_number = cells[0].get_text(strip=True) if cells else ""
                result.full_name = cells[1].get_text(strip=True) if len(cells) > 1 else ""
                status_text = cells[2].get_text(strip=True).lower() if len(cells) > 2 else ""
                result.status = status_text.title()
                result.active = "active" in status_text or "valid" in status_text

                if len(cells) > 3:
                    result.license_type = cells[3].get_text(strip=True)

            except (IndexError, AttributeError) as e:
                logger.warning(f"CA parse error: {e}")
                continue

            results.append(result)

        return results if results else [LicenseResult(found=False)]

    async def close(self) -> None:
        if self._session and not self._session.closed:
            await self._session.close()

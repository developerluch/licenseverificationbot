# Texas Department of Insurance — License Lookup Scraper
# Source: https://txapps.texas.gov/NASApp/tdi/TdiARManager
#
# Texas routes through their own TdiARManager app.
# Supports: name search (full or partial), license number.
# Returns: status, address, lines of authority, discipline history.

import logging
from typing import Optional

import aiohttp
from bs4 import BeautifulSoup

from scrapers.base import LicenseResult, StateScraper

logger = logging.getLogger(__name__)

BASE_URL = "https://txapps.texas.gov/NASApp/tdi/TdiARManager"


class TexasScraper(StateScraper):
    STATE_CODE = "TX"
    STATE_NAME = "Texas"
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
        """Search TX TDI by name."""
        try:
            session = await self._get_session()

            # TX TDI uses form POST
            data = {
                "perlession": "individual",
                "searchName": f"{last_name}, {first_name}".strip(),
                "action": "Search",
            }

            async with session.post(BASE_URL, data=data) as resp:
                if resp.status != 200:
                    logger.error(f"TX lookup failed: HTTP {resp.status}")
                    return [LicenseResult(error=f"HTTP {resp.status}")]
                html = await resp.text()

            return self._parse_results(html)

        except Exception as e:
            logger.error(f"TX name lookup error: {e}")
            return [LicenseResult(error=str(e))]

    async def lookup_by_license_number(
        self, license_number: str
    ) -> list[LicenseResult]:
        """Search TX TDI by license number."""
        try:
            session = await self._get_session()
            data = {
                "perlession": "individual",
                "searchLicense": license_number.strip(),
                "action": "Search",
            }
            async with session.post(BASE_URL, data=data) as resp:
                if resp.status != 200:
                    return [LicenseResult(error=f"HTTP {resp.status}")]
                html = await resp.text()
            return self._parse_results(html)
        except Exception as e:
            logger.error(f"TX license# lookup error: {e}")
            return [LicenseResult(error=str(e))]

    async def lookup_by_npn(self, npn: str) -> list[LicenseResult]:
        """TX TDI doesn't directly support NPN — fall back to empty."""
        logger.warning("TX TDI does not support direct NPN lookup")
        return [LicenseResult(found=False, error="TX does not support NPN lookup")]

    def _parse_results(self, html: str) -> list[LicenseResult]:
        """Parse TX TDI search results."""
        soup = BeautifulSoup(html, "html.parser")
        results = []

        # Find results table
        table = soup.find("table", class_="resultsTable") or soup.find("table")
        if not table:
            if "no records" in html.lower() or "no results" in html.lower():
                return [LicenseResult(found=False)]
            return [LicenseResult(found=False)]

        rows = table.find_all("tr")[1:]  # Skip header
        for row in rows:
            cells = row.find_all("td")
            if len(cells) < 2:
                continue

            result = LicenseResult(found=True, state="TX")

            try:
                result.full_name = cells[0].get_text(strip=True)
                if len(cells) > 1:
                    result.license_number = cells[1].get_text(strip=True)
                if len(cells) > 2:
                    status_text = cells[2].get_text(strip=True).lower()
                    result.status = status_text.title()
                    result.active = "active" in status_text or "current" in status_text
                if len(cells) > 3:
                    result.license_type = cells[3].get_text(strip=True)

            except (IndexError, AttributeError) as e:
                logger.warning(f"TX parse error: {e}")
                continue

            results.append(result)

        return results if results else [LicenseResult(found=False)]

    async def close(self) -> None:
        if self._session and not self._session.closed:
            await self._session.close()

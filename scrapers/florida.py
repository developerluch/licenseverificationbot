# Florida Department of Financial Services — License Lookup Scraper
# Source: https://licenseesearch.fldfs.com
#
# Florida's lookup supports: name, NPN, FL license number, email.
# Returns: license status, category, issue/expiration dates, appointments.
# Uses aiohttp + BeautifulSoup (no browser needed — form is simple POST).

import logging
from typing import Optional

import aiohttp
from bs4 import BeautifulSoup

from scrapers.base import LicenseResult, StateScraper

logger = logging.getLogger(__name__)

BASE_URL = "https://licenseesearch.fldfs.com"
SEARCH_URL = f"{BASE_URL}/Licensee"


class FloridaScraper(StateScraper):
    STATE_CODE = "FL"
    STATE_NAME = "Florida"
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
        """Search FL DOI by first + last name."""
        try:
            session = await self._get_session()

            # Florida's search form uses GET parameters
            params = {
                "SearchFirstName": first_name.strip(),
                "SearchLastName": last_name.strip(),
                "SearchLicenseStatus": "Both",  # Valid + Invalid
            }

            async with session.get(SEARCH_URL, params=params) as resp:
                if resp.status != 200:
                    logger.error(f"FL lookup failed: HTTP {resp.status}")
                    return [LicenseResult(error=f"HTTP {resp.status}")]

                html = await resp.text()

            return self._parse_search_results(html)

        except Exception as e:
            logger.error(f"FL name lookup error: {e}")
            return [LicenseResult(error=str(e))]

    async def lookup_by_npn(self, npn: str) -> list[LicenseResult]:
        """Search FL DOI by NPN."""
        try:
            session = await self._get_session()
            params = {
                "SearchNPN": npn.strip(),
                "SearchLicenseStatus": "Both",
            }
            async with session.get(SEARCH_URL, params=params) as resp:
                if resp.status != 200:
                    return [LicenseResult(error=f"HTTP {resp.status}")]
                html = await resp.text()
            return self._parse_search_results(html)
        except Exception as e:
            logger.error(f"FL NPN lookup error: {e}")
            return [LicenseResult(error=str(e))]

    async def lookup_by_license_number(
        self, license_number: str
    ) -> list[LicenseResult]:
        """Search FL DOI by FL license number."""
        try:
            session = await self._get_session()
            params = {
                "SearchLicenseNumber": license_number.strip(),
                "SearchLicenseStatus": "Both",
            }
            async with session.get(SEARCH_URL, params=params) as resp:
                if resp.status != 200:
                    return [LicenseResult(error=f"HTTP {resp.status}")]
                html = await resp.text()
            return self._parse_search_results(html)
        except Exception as e:
            logger.error(f"FL license# lookup error: {e}")
            return [LicenseResult(error=str(e))]

    def _parse_search_results(self, html: str) -> list[LicenseResult]:
        """Parse the FL DOI search results page."""
        soup = BeautifulSoup(html, "html.parser")
        results = []

        # Look for result rows in the search results table
        table = soup.find("table", class_="table") or soup.find("table")
        if not table:
            # Check for "no results" message
            if "no licensee" in html.lower() or "no results" in html.lower():
                return [LicenseResult(found=False)]
            # Try to find individual result cards/divs
            cards = soup.find_all("div", class_="licensee-card") or []
            if not cards:
                logger.warning("FL: Could not find results table or cards")
                return [LicenseResult(found=False)]

        rows = table.find_all("tr")[1:] if table else []  # Skip header

        for row in rows:
            cells = row.find_all("td")
            if len(cells) < 3:
                continue

            result = LicenseResult(
                found=True,
                state="FL",
            )

            # Parse cells — FL typically shows:
            # Name | License # | Status | Category | Expiration
            try:
                result.full_name = cells[0].get_text(strip=True)
                result.license_number = cells[1].get_text(strip=True) if len(cells) > 1 else ""
                status_text = cells[2].get_text(strip=True).lower() if len(cells) > 2 else ""
                result.status = status_text.title()
                result.active = "valid" in status_text

                if len(cells) > 3:
                    result.license_type = cells[3].get_text(strip=True)
                if len(cells) > 4:
                    result.expiration_date = cells[4].get_text(strip=True)

                # Try to get detail link for more info
                link = row.find("a", href=True)
                if link and "/Licensee/" in link["href"]:
                    result.raw_data["detail_url"] = BASE_URL + link["href"]

            except (IndexError, AttributeError) as e:
                logger.warning(f"FL parse error on row: {e}")
                continue

            results.append(result)

        if not results:
            return [LicenseResult(found=False)]

        return results

    async def get_detail(self, detail_url: str) -> dict:
        """Fetch the full detail page for a specific licensee."""
        try:
            session = await self._get_session()
            async with session.get(detail_url) as resp:
                if resp.status != 200:
                    return {}
                html = await resp.text()

            soup = BeautifulSoup(html, "html.parser")
            details = {}

            # Parse detail fields
            for row in soup.find_all("div", class_="row"):
                label = row.find("label") or row.find("strong")
                value = row.find("span") or row.find("div", class_="col")
                if label and value:
                    key = label.get_text(strip=True).rstrip(":").lower().replace(" ", "_")
                    details[key] = value.get_text(strip=True)

            return details

        except Exception as e:
            logger.error(f"FL detail fetch error: {e}")
            return {}

    async def close(self) -> None:
        if self._session and not self._session.closed:
            await self._session.close()

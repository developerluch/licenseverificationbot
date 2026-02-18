# Florida Department of Financial Services — License Lookup Scraper
# Source: https://licenseesearch.fldfs.com
#
# Flow:
#   1. GET / to establish session cookies
#   2. POST / with full form payload (name fields filled, rest empty)
#   3. Parse the response HTML for search results table
#   4. Fetch /Licensee/{id} detail page for status (Valid/Invalid)

import logging
import re
from typing import Optional

import aiohttp
from bs4 import BeautifulSoup

from scrapers.base import LicenseResult, StateScraper

logger = logging.getLogger(__name__)

BASE_URL = "https://licenseesearch.fldfs.com"


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

    def _build_form_data(
        self,
        first_name: str = "",
        last_name: str = "",
        license_number: str = "",
        npn: str = "",
    ) -> dict:
        """Build the full POST payload matching the FL DOI form exactly."""
        return {
            "IndividualFNameFilter": first_name.strip(),
            "IndividualLNameFilter": last_name.strip(),
            "IndividualMNameFilter": "",
            "EmailAddressBeginContainFilter": "1",
            "EmailFilter": "",
            "FirmNameBeginContainFilter": "1",
            "FirmNameFilter": "",
            "ResidentStatusFilter": "",
            "FLLicenseNoFilter": license_number.strip(),
            "NPNNoFilter": npn.strip(),
            "LicenseStatusFilter": "1",
            "LicenseCategoryFilter": "",
            "LicenseIssueDateFromFilter": "",
            "LicenseIssueDateToFilter": "",
            "OnlyLicWithNoQuApptFilter": "false",
            "BusinessStateFilter": "",
            "BusinessCityFilter": "",
            "BusinessCountyFilter": "",
            "BusinessZipFilter": "",
            "CEDueDtFromFilter": "",
            "CEDueDtToFilter": "",
            "CEHrsNotMetFilter": "false",
            "AppointingEntityTYCLFilter": "",
            "AppointingEntityStatusFilter": "",
            "AppointingEntityStatusDateFromFilter": "",
            "AppointingEntityStatusDateToFilter": "",
            "LicenseeSearchInfo.PagingInfo.SortBy": "Name",
            "LicenseeSearchInfo.PagingInfo.SortDesc": "False",
            "LicenseeSearchInfo.PagingInfo.CurrentPage": "1",
            "AppointingEntityIdFilter": "",
            "AppointingEntityDisplayName": "",
            "TabLLValue": "0",
            "TabCEValue": "0",
            "TabAppValue": "",
            "hdnLApptEntitySearchListUrl": "/Home/GetAppointingEntityListForSearch",
            "hdnLicenseeSearchListUrl": "/Home/GetLicenseeSearchListPartialView",
        }

    async def _search(self, form_data: dict) -> list[LicenseResult]:
        """Execute the two-step search: POST form to /, then parse results."""
        try:
            session = await self._get_session()

            # Step 1: GET / to establish session cookies
            async with session.get(BASE_URL) as resp:
                if resp.status != 200:
                    logger.error(f"FL GET failed: HTTP {resp.status}")
                    return [LicenseResult(error=f"HTTP {resp.status} on GET")]

            # Step 2: POST / with full form payload
            async with session.post(
                BASE_URL + "/",
                data=form_data,
                headers={
                    "Content-Type": "application/x-www-form-urlencoded",
                    "Referer": BASE_URL,
                },
            ) as resp:
                if resp.status != 200:
                    logger.error(f"FL POST failed: HTTP {resp.status}")
                    return [LicenseResult(error=f"HTTP {resp.status} on POST")]
                html = await resp.text()

            return await self._parse_search_results(html)

        except Exception as e:
            logger.error(f"FL search error: {e}")
            return [LicenseResult(error=str(e))]

    async def lookup_by_name(
        self, first_name: str, last_name: str
    ) -> list[LicenseResult]:
        """Search FL DOI by first + last name."""
        logger.info(f"FL lookup: {first_name} {last_name}")
        form_data = self._build_form_data(first_name=first_name, last_name=last_name)
        return await self._search(form_data)

    async def lookup_by_npn(self, npn: str) -> list[LicenseResult]:
        """Search FL DOI by NPN."""
        logger.info(f"FL NPN lookup: {npn}")
        form_data = self._build_form_data(npn=npn)
        return await self._search(form_data)

    async def lookup_by_license_number(
        self, license_number: str
    ) -> list[LicenseResult]:
        """Search FL DOI by FL license number."""
        logger.info(f"FL license# lookup: {license_number}")
        form_data = self._build_form_data(license_number=license_number)
        return await self._search(form_data)

    async def _parse_search_results(self, html: str) -> list[LicenseResult]:
        """Parse the FL DOI search results page and fetch detail pages."""
        soup = BeautifulSoup(html, "html.parser")
        results = []

        # Check for "No Licensee Detail Found"
        if "no licensee" in html.lower() or "no results" in html.lower():
            logger.info("FL: No results found")
            return [LicenseResult(found=False)]

        # Find the results table
        table = soup.find("table", class_="table")
        if not table:
            logger.warning("FL: Could not find results table")
            return [LicenseResult(found=False)]

        rows = table.find("tbody")
        if not rows:
            return [LicenseResult(found=False)]

        for row in rows.find_all("tr"):
            cells = row.find_all("td")
            if len(cells) < 2:
                continue

            # Column order: Name | License Number | Business Address | City/State/Zip | Email
            name_cell = cells[0]
            license_cell = cells[1] if len(cells) > 1 else None

            # Get the detail page link and name
            link = name_cell.find("a", href=True)
            if not link:
                continue

            full_name = link.get_text(strip=True)
            detail_path = link["href"]  # e.g., /Licensee/1432239
            license_number = license_cell.get_text(strip=True) if license_cell else ""

            # Fetch the detail page to get status, type, expiration
            detail = await self._fetch_detail(detail_path)

            result = LicenseResult(
                found=True,
                state="FL",
                full_name=full_name,
                license_number=license_number,
                status=detail.get("status", "Unknown"),
                active=detail.get("status", "").upper() == "VALID",
                license_type=detail.get("type", ""),
                npn=detail.get("npn", ""),
                expiration_date=detail.get("expiration", ""),
                issue_date=detail.get("issue_date", ""),
                raw_data=detail,
            )
            results.append(result)

            # Only fetch details for first 5 results to avoid rate limiting
            if len(results) >= 5:
                break

        if not results:
            return [LicenseResult(found=False)]

        logger.info(f"FL: Found {len(results)} results")
        return results

    async def _fetch_detail(self, detail_path: str) -> dict:
        """Fetch the licensee detail page and extract status, type, NPN, etc.

        The FL DOI detail page layout:
          - Top section: form-group divs with label + sibling div for values
            (License #, Full Name, Business Address, Email, Phone, County, NPN #)
          - "Valid Licenses" panel: table with Type | Issue Date | Qualifying Appointment
          - "Invalid Licenses" panel: same table structure (or "No invalid licenses found")
          - "Active Appointments" / "Inactive Appointments" panels
        Status is determined by WHICH panel the license appears in (Valid vs Invalid).
        """
        try:
            session = await self._get_session()
            url = BASE_URL + detail_path

            async with session.get(url) as resp:
                if resp.status != 200:
                    return {"error": f"HTTP {resp.status}"}
                html = await resp.text()

            soup = BeautifulSoup(html, "html.parser")

            # ── Extract label: value pairs from form-group divs ──
            # Structure: <div class="form-group ...">
            #              <label class="... control-label">Label:</label>
            #              <div class="col-md-8">Value</div>
            #            </div>
            detail = {}
            for fg in soup.find_all("div", class_="form-group"):
                label_tag = fg.find("label")
                if not label_tag:
                    continue
                label_text = label_tag.get_text(strip=True).rstrip(":")
                # Value div is the sibling of the label inside the same form-group
                value_div = label_tag.find_next_sibling("div")
                if value_div:
                    value = value_div.get_text(" ", strip=True)
                    if value:
                        key = label_text.lower().replace(" ", "_").replace("#", "num")
                        detail[key] = value

            result = {
                "full_name": detail.get("full_name", ""),
                "license_number": detail.get("license_num", detail.get("license", "")),
                "npn": detail.get("npn_num", ""),
                "email": detail.get("email", ""),
                "phone": detail.get("phone", ""),
                "business_address": detail.get("business_address", ""),
                "mailing_address": detail.get("mailing_address", ""),
                "county": detail.get("county", ""),
            }

            # ── Extract license types from Valid / Invalid panels ──
            # Each panel has a panel-heading ("Valid Licenses" / "Invalid Licenses")
            # followed by a table with columns: Type | Issue Date | Qualifying Appointment
            valid_licenses = []
            invalid_licenses = []

            for panel in soup.find_all("div", class_="panel"):
                heading = panel.find("div", class_="panel-heading")
                if not heading:
                    continue
                heading_text = heading.get_text(strip=True).lower()

                table = panel.find("table")
                if not table:
                    continue
                tbody = table.find("tbody")
                if not tbody:
                    continue

                for row in tbody.find_all("tr"):
                    cells = row.find_all("td")
                    if len(cells) >= 2:
                        lic_type = cells[0].get_text(strip=True)
                        issue_date = cells[1].get_text(strip=True)
                        entry = {"type": lic_type, "issue_date": issue_date}

                        if "valid license" in heading_text:
                            entry["status"] = "VALID"
                            valid_licenses.append(entry)
                        elif "invalid license" in heading_text:
                            entry["status"] = "INVALID"
                            invalid_licenses.append(entry)

            # Pick the best license to report:
            # Prefer life/health insurance from valid licenses, then any valid, then invalid
            all_licenses = valid_licenses + invalid_licenses
            chosen = None
            for lic in valid_licenses:
                if "life" in lic["type"].lower() or "health" in lic["type"].lower():
                    chosen = lic
                    break
            if not chosen and valid_licenses:
                chosen = valid_licenses[0]
            if not chosen and invalid_licenses:
                chosen = invalid_licenses[0]

            if chosen:
                result["type"] = chosen["type"]
                result["issue_date"] = chosen.get("issue_date", "")
                result["status"] = chosen["status"]

            # ── Extract expiration from Active/Inactive Appointments ──
            for panel in soup.find_all("div", class_="panel"):
                heading = panel.find("div", class_="panel-heading")
                if not heading:
                    continue
                heading_text = heading.get_text(strip=True).lower()
                if "appointment" not in heading_text:
                    continue

                table = panel.find("table")
                if not table:
                    continue
                tbody = table.find("tbody")
                if not tbody:
                    continue

                for row in tbody.find_all("tr"):
                    cells = row.find_all("td")
                    # Active Appointments: Company Name | Issue Date | Exp Date | Status Date
                    if len(cells) >= 3:
                        exp_date = cells[2].get_text(strip=True)
                        if exp_date and not result.get("expiration"):
                            result["expiration"] = exp_date

            return result

        except Exception as e:
            logger.error(f"FL detail fetch error: {e}")
            return {"error": str(e)}

    async def close(self) -> None:
        if self._session and not self._session.closed:
            await self._session.close()

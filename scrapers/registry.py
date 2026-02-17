# Scraper registry — maps state codes to their scrapers
# Falls back to NAIC SOLAR for unsupported states.

import logging

from scrapers.base import StateScraper
from scrapers.florida import FloridaScraper
from scrapers.california import CaliforniaScraper
from scrapers.texas import TexasScraper
from scrapers.naic import NAICScraper

logger = logging.getLogger(__name__)

# States with dedicated scrapers
_SCRAPER_MAP: dict[str, type[StateScraper]] = {
    "FL": FloridaScraper,
    "CA": CaliforniaScraper,
    "TX": TexasScraper,
}

SUPPORTED_STATES = set(_SCRAPER_MAP.keys())


def get_scraper(state_code: str) -> StateScraper:
    """Get the appropriate scraper for a state.

    Returns a state-specific scraper if available, otherwise
    falls back to NAIC SOLAR with the state filter applied.
    """
    state_code = state_code.upper().strip()

    if state_code in _SCRAPER_MAP:
        logger.info(f"Using dedicated scraper for {state_code}")
        return _SCRAPER_MAP[state_code]()

    # Fallback to NAIC SOLAR for any other state
    logger.info(f"No dedicated scraper for {state_code} — using NAIC SOLAR fallback")
    return NAICScraper(state_code=state_code)

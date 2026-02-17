# VIPA License Bot — Database layer
# Connects to the SAME database as onboarding-bot (shared schema).
# Only adds the license_checks table if it doesn't exist.
import logging
from datetime import datetime, timezone

import aiosqlite

from config import settings

logger = logging.getLogger(__name__)

# Additional tables this bot needs (onboarding-bot owns the main schema)
LICENSE_BOT_SCHEMA = """
CREATE TABLE IF NOT EXISTS license_checks (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    agent_discord_id    INTEGER NOT NULL,
    check_date          TEXT NOT NULL DEFAULT (datetime('now')),
    state               TEXT,
    status              TEXT,
    details             TEXT,
    notified            INTEGER DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_license_checks_agent
    ON license_checks(agent_discord_id);
CREATE INDEX IF NOT EXISTS idx_license_checks_date
    ON license_checks(check_date);
"""


class LicenseDB:
    """Database operations for the license verification bot."""

    def __init__(self):
        self._db_path = settings.DATABASE_URL.replace("sqlite:///", "")

    def _path(self) -> str:
        return self._db_path

    async def init(self) -> None:
        """Create license-specific tables + run migrations on shared tables."""
        async with aiosqlite.connect(self._path()) as db:
            await db.executescript(LICENSE_BOT_SCHEMA)

            # Ensure onboarding_agents has the columns we need
            migrations = [
                "ALTER TABLE onboarding_agents ADD COLUMN phone_number TEXT",
                "ALTER TABLE onboarding_agents ADD COLUMN license_verified INTEGER DEFAULT 0",
                "ALTER TABLE onboarding_agents ADD COLUMN license_expiry TEXT",
                "ALTER TABLE onboarding_agents ADD COLUMN last_license_check TEXT",
            ]
            for sql in migrations:
                try:
                    await db.execute(sql)
                except Exception:
                    pass  # Column already exists

            await db.commit()
        logger.info(f"License DB initialized (shared: {self._db_path})")

    # ------------------------------------------------------------------ #
    #  Read from shared onboarding_agents table                            #
    # ------------------------------------------------------------------ #

    async def get_licensed_agents(self) -> list[dict]:
        """Get all licensed agents who need monitoring."""
        async with aiosqlite.connect(self._path()) as db:
            db.row_factory = aiosqlite.Row
            async with db.execute(
                """SELECT * FROM onboarding_agents
                   WHERE license_status = 'licensed'
                   AND kicked_at IS NULL
                   AND home_state IS NOT NULL
                   AND home_state != ''
                   AND full_name IS NOT NULL"""
            ) as cursor:
                return [dict(r) for r in await cursor.fetchall()]

    async def get_agent(self, discord_id: int) -> dict | None:
        """Get a single agent by Discord ID."""
        async with aiosqlite.connect(self._path()) as db:
            db.row_factory = aiosqlite.Row
            async with db.execute(
                "SELECT * FROM onboarding_agents WHERE discord_id = ?",
                (discord_id,),
            ) as cursor:
                row = await cursor.fetchone()
                return dict(row) if row else None

    # ------------------------------------------------------------------ #
    #  Write to shared onboarding_agents table                             #
    # ------------------------------------------------------------------ #

    async def update_agent(self, discord_id: int, **kwargs) -> None:
        """Update agent fields in the shared table."""
        if not kwargs:
            return
        set_parts = [f"{k} = ?" for k in kwargs]
        vals = list(kwargs.values()) + [discord_id]
        async with aiosqlite.connect(self._path()) as db:
            await db.execute(
                f"UPDATE onboarding_agents SET {', '.join(set_parts)} WHERE discord_id = ?",
                vals,
            )
            await db.commit()

    # ------------------------------------------------------------------ #
    #  License check log (owned by this bot)                               #
    # ------------------------------------------------------------------ #

    async def log_check(
        self, discord_id: int, state: str, status: str, details: str = ""
    ) -> None:
        """Log a license verification check."""
        async with aiosqlite.connect(self._path()) as db:
            await db.execute(
                """INSERT INTO license_checks
                   (agent_discord_id, state, status, details)
                   VALUES (?, ?, ?, ?)""",
                (discord_id, state, status, details),
            )
            await db.commit()

    async def get_check_history(
        self, discord_id: int, limit: int = 10
    ) -> list[dict]:
        """Get recent license check history for an agent."""
        async with aiosqlite.connect(self._path()) as db:
            db.row_factory = aiosqlite.Row
            async with db.execute(
                """SELECT * FROM license_checks
                   WHERE agent_discord_id = ?
                   ORDER BY check_date DESC LIMIT ?""",
                (discord_id, limit),
            ) as cursor:
                return [dict(r) for r in await cursor.fetchall()]

    # ------------------------------------------------------------------ #
    #  Activity log (write to shared table)                                #
    # ------------------------------------------------------------------ #

    async def log_activity(
        self, discord_id: int, event_type: str, details: str = ""
    ) -> None:
        """Log to the shared activity log."""
        async with aiosqlite.connect(self._path()) as db:
            await db.execute(
                """INSERT INTO agent_activity_log
                   (agent_discord_id, event_type, details)
                   VALUES (?, ?, ?)""",
                (discord_id, event_type, details),
            )
            await db.commit()

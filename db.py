# VIPA License Bot — Database layer
# Creates its own copy of the onboarding_agents table if it doesn't exist.
# When deployed alongside the onboarding bot with a shared DB, both use the same table.
# When deployed separately, the license bot creates the table itself.
import logging
from datetime import datetime, timezone

import aiosqlite

from config import settings

logger = logging.getLogger(__name__)

# Full schema — license bot creates these if they don't exist
LICENSE_BOT_SCHEMA = """
CREATE TABLE IF NOT EXISTS onboarding_agents (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    discord_id          INTEGER UNIQUE NOT NULL,
    guild_id            INTEGER NOT NULL,
    full_name           TEXT,
    agency              TEXT,
    upline_manager      TEXT,
    experience_level    TEXT,
    license_status      TEXT DEFAULT 'none',
    production_written  TEXT,
    lead_source         TEXT,
    vision_goals        TEXT,
    comp_pct            TEXT,
    show_comp           INTEGER DEFAULT 0,
    npn                 TEXT,
    license_number      TEXT,
    home_state          TEXT,
    resident_state      TEXT,
    verified_at         TEXT,
    current_stage       INTEGER DEFAULT 1,
    notification_pref   TEXT DEFAULT 'discord',
    contracting_booked  INTEGER DEFAULT 0,
    contracting_completed INTEGER DEFAULT 0,
    setup_completed     INTEGER DEFAULT 0,
    joined_at           TEXT NOT NULL DEFAULT (datetime('now')),
    form_completed_at   TEXT,
    sorted_at           TEXT,
    activated_at        TEXT,
    kicked_at           TEXT,
    kick_reason         TEXT,
    phone_number        TEXT,
    license_verified    INTEGER DEFAULT 0,
    license_expiry      TEXT,
    last_license_check  TEXT,
    last_active         TEXT DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS license_checks (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    agent_discord_id    INTEGER NOT NULL,
    check_date          TEXT NOT NULL DEFAULT (datetime('now')),
    state               TEXT,
    status              TEXT,
    details             TEXT,
    notified            INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS agent_activity_log (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    agent_discord_id    INTEGER NOT NULL,
    event_type          TEXT NOT NULL,
    details             TEXT,
    created_at          TEXT NOT NULL DEFAULT (datetime('now'))
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
        """Create all tables if they don't exist."""
        async with aiosqlite.connect(self._path()) as db:
            await db.executescript(LICENSE_BOT_SCHEMA)
            await db.commit()
        logger.info(f"License DB initialized at {self._db_path}")

    # ------------------------------------------------------------------ #
    #  Read from onboarding_agents table                                   #
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
    #  Write to onboarding_agents table                                    #
    # ------------------------------------------------------------------ #

    async def upsert_agent(self, discord_id: int, guild_id: int, **kwargs) -> None:
        """Create agent if not exists, then update fields."""
        async with aiosqlite.connect(self._path()) as db:
            # Create if not exists
            await db.execute(
                "INSERT OR IGNORE INTO onboarding_agents (discord_id, guild_id) VALUES (?, ?)",
                (discord_id, guild_id),
            )
            # Update fields
            if kwargs:
                set_parts = [f"{k} = ?" for k in kwargs]
                vals = list(kwargs.values()) + [discord_id]
                await db.execute(
                    f"UPDATE onboarding_agents SET {', '.join(set_parts)} WHERE discord_id = ?",
                    vals,
                )
            await db.commit()

    async def update_agent(self, discord_id: int, **kwargs) -> None:
        """Update agent fields."""
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
    #  License check log                                                   #
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
    #  Activity log                                                        #
    # ------------------------------------------------------------------ #

    async def log_activity(
        self, discord_id: int, event_type: str, details: str = ""
    ) -> None:
        """Log to the activity log."""
        async with aiosqlite.connect(self._path()) as db:
            await db.execute(
                """INSERT INTO agent_activity_log
                   (agent_discord_id, event_type, details)
                   VALUES (?, ?, ?)""",
                (discord_id, event_type, details),
            )
            await db.commit()

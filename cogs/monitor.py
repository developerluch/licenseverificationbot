# Scheduled license monitoring — checks all verified agents on a loop
import logging
from datetime import datetime, timezone

import discord
from discord.ext import commands, tasks

from config import settings
from db import LicenseDB
from scrapers import get_scraper
from sms import send_license_alert

logger = logging.getLogger(__name__)


class MonitorCog(commands.Cog, name="Monitor"):
    """Scheduled background license status checks + DM/SMS alerts."""

    def __init__(self, bot: commands.Bot, db: LicenseDB) -> None:
        self.bot = bot
        self.db = db

    async def cog_load(self) -> None:
        if settings.LICENSE_CHECK_ENABLED and settings.LICENSE_CHECK_INTERVAL_HOURS > 0:
            self.check_loop.change_interval(hours=settings.LICENSE_CHECK_INTERVAL_HOURS)
            self.check_loop.start()
            logger.info(
                f"License monitor started — every {settings.LICENSE_CHECK_INTERVAL_HOURS}h"
            )

    async def cog_unload(self) -> None:
        self.check_loop.cancel()

    # ================================================================== #
    #  Scheduled Loop                                                      #
    # ================================================================== #

    @tasks.loop(hours=168)  # Overridden in cog_load
    async def check_loop(self):
        """Run license checks for all monitored agents."""
        logger.info("=== Scheduled license check starting ===")

        agents = await self.db.get_licensed_agents()
        if not agents:
            logger.info("No agents to check")
            return

        checked = 0
        alerts = 0
        errors = 0

        for agent in agents:
            try:
                result = await self._check_one(agent)
                checked += 1
                if result == "alert":
                    alerts += 1
            except Exception as e:
                errors += 1
                logger.error(f"Check failed for {agent.get('discord_id')}: {e}")

        logger.info(f"License check done: {checked} ok, {alerts} alerts, {errors} errors")

        # Post summary
        await self._post_summary(checked, alerts, errors)

    @check_loop.before_loop
    async def before_check(self):
        await self.bot.wait_until_ready()

    # ================================================================== #
    #  Single agent check                                                  #
    # ================================================================== #

    async def _check_one(self, agent: dict) -> str:
        """Check a single agent's license. Returns 'ok', 'alert', or 'error'."""
        discord_id = agent["discord_id"]
        full_name = agent.get("full_name", "")
        state = agent.get("home_state", "")
        phone = agent.get("phone_number")

        if not full_name or not state:
            return "error"

        # Split name into first/last
        parts = full_name.strip().split()
        first = parts[0] if parts else ""
        last = parts[-1] if len(parts) > 1 else parts[0]

        # Scrape
        scraper = get_scraper(state)
        try:
            results = await scraper.lookup_by_name(first, last)
        finally:
            await scraper.close()

        # Check if any result shows an active license
        active = any(r.found and r.active for r in results)

        # Update check timestamp
        await self.db.update_agent(
            discord_id,
            last_license_check=datetime.now(timezone.utc).isoformat(),
        )

        if active:
            await self.db.log_check(discord_id, state, "active", "License active")
            return "ok"

        # ── LICENSE PROBLEM ──
        logger.warning(f"LICENSE ALERT: {full_name} ({discord_id}) — {state}")
        await self.db.log_check(discord_id, state, "inactive", "License not found or inactive")

        # DM the agent
        await self._dm_alert(discord_id, full_name, state)

        # SMS the agent
        if phone:
            await send_license_alert(
                phone, full_name, state, "expired",
                "Run /verify in VIPA Discord to update your info."
            )

        return "alert"

    # ================================================================== #
    #  Notifications                                                       #
    # ================================================================== #

    async def _dm_alert(self, discord_id: int, name: str, state: str) -> None:
        """DM an agent about a license status issue."""
        try:
            guild = self.bot.get_guild(settings.GUILD_ID)
            if not guild:
                return
            member = guild.get_member(discord_id)
            if not member:
                try:
                    member = await guild.fetch_member(discord_id)
                except Exception:
                    return

            embed = discord.Embed(
                title="\u26a0\ufe0f  License Status Alert",
                description=(
                    f"Hey {name}, we ran a routine license check and couldn't "
                    f"confirm an active license for you in **{state}**.\n\n"
                    f"**This could mean:**\n"
                    f"\u2022 Your license has expired or lapsed\n"
                    f"\u2022 Your name on file is different\n"
                    f"\u2022 There's a temporary lookup issue\n\n"
                    f"**What to do:**\n"
                    f"\u2022 Run `/verify` to update your info\n"
                    f"\u2022 Contact your upline if you need help renewing\n"
                    f"\u2022 Check your state DOI website directly"
                ),
                color=0xE74C3C,
                timestamp=datetime.now(timezone.utc),
            )
            embed.set_footer(text="VIPA License Monitor")
            await member.send(embed=embed)
            logger.info(f"Sent DM alert to {name} ({discord_id})")

        except discord.Forbidden:
            logger.warning(f"Cannot DM {discord_id} — DMs disabled")
        except Exception as e:
            logger.error(f"DM alert failed for {discord_id}: {e}")

    async def _post_summary(self, checked: int, alerts: int, errors: int) -> None:
        """Post check summary to the license check channel."""
        if not settings.LICENSE_CHECK_CHANNEL_ID:
            return
        channel = self.bot.get_channel(settings.LICENSE_CHECK_CHANNEL_ID)
        if not channel:
            return

        color = 0x2ECC71 if alerts == 0 else 0xE74C3C
        embed = discord.Embed(
            title="\U0001f50d  License Check Complete",
            description=(
                f"**Agents checked:** {checked}\n"
                f"**Alerts sent:** {alerts}\n"
                f"**Errors:** {errors}\n"
                f"**Time:** {datetime.now(timezone.utc).strftime('%Y-%m-%d %H:%M UTC')}"
            ),
            color=color,
            timestamp=datetime.now(timezone.utc),
        )
        embed.set_footer(text="VIPA License Monitor")
        await channel.send(embed=embed)


async def setup(bot: commands.Bot) -> None:
    await bot.add_cog(MonitorCog(bot, bot._db))

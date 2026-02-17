# /verify command — manual license verification via state DOI scrapers
import logging
from datetime import datetime, timezone

import discord
from discord import app_commands
from discord.ext import commands

from config import settings
from db import LicenseDB
from roles import assign_role, remove_role
from scrapers import get_scraper
from sms import send_license_alert

logger = logging.getLogger(__name__)


class VerifyCog(commands.Cog, name="Verify"):
    """/verify slash command for agents to verify their license."""

    def __init__(self, bot: commands.Bot, db: LicenseDB) -> None:
        self.bot = bot
        self.db = db

    @app_commands.command(
        name="verify",
        description="Verify your insurance license and get promoted to Licensed Agent",
    )
    @app_commands.describe(
        first_name="Your legal first name (as on your license)",
        last_name="Your legal last name (as on your license)",
        state="Your home state (2-letter code, e.g. FL, TX, CA)",
        phone="Your phone number for license update texts (optional)",
    )
    async def verify_license(
        self,
        interaction: discord.Interaction,
        first_name: str,
        last_name: str,
        state: str = None,
        phone: str = None,
    ) -> None:
        await interaction.response.defer(ephemeral=True)
        member = interaction.user
        logger.info(f"License verify: {first_name} {last_name} ({state}) by {member}")

        # Pull state from DB if not provided
        if not state:
            agent = await self.db.get_agent(member.id)
            state = agent.get("home_state", "") if agent else ""

        if not state or len(state) != 2:
            await interaction.followup.send(
                "\u274c Please provide your 2-letter state code.\n"
                "Example: `/verify first_name:John last_name:Doe state:FL`",
                ephemeral=True,
            )
            return

        state = state.upper().strip()

        # Save phone number if provided
        if phone:
            clean_phone = "".join(c for c in phone if c.isdigit())
            if len(clean_phone) == 10:
                clean_phone = "+1" + clean_phone
            elif len(clean_phone) == 11 and clean_phone.startswith("1"):
                clean_phone = "+" + clean_phone
            if clean_phone:
                await self.db.update_agent(member.id, phone_number=clean_phone)

        # Run the scraper for this state
        scraper = get_scraper(state)
        try:
            results = await scraper.lookup_by_name(first_name, last_name)
        finally:
            await scraper.close()

        # Find best match — prefer life insurance licenses
        match = None
        for r in results:
            if r.found and r.active:
                if r.is_life_licensed or not r.license_type:
                    match = r
                    break
        if not match:
            for r in results:
                if r.found and r.active:
                    match = r
                    break

        if match and match.found and match.active:
            # ── SUCCESS ──
            await self.db.update_agent(
                member.id,
                npn=match.npn or None,
                license_number=match.license_number or None,
                resident_state=state,
                license_status="licensed",
                license_verified=1,
                license_expiry=match.expiration_date or None,
                verified_at=datetime.now(timezone.utc).isoformat(),
                last_license_check=datetime.now(timezone.utc).isoformat(),
                current_stage=5,
            )
            await self.db.log_activity(
                member.id, "license_verified",
                f"Verified via {state} DOI: {match.full_name} | "
                f"License: {match.license_number} | Status: {match.status}"
            )
            await self.db.log_check(member.id, state, "verified", match.status)

            # Role changes
            try:
                guild = self.bot.get_guild(settings.GUILD_ID)
                guild_member = guild.get_member(member.id) if guild else None
                if guild_member:
                    if settings.LICENSED_AGENT_ROLE_ID:
                        await assign_role(guild_member, settings.LICENSED_AGENT_ROLE_ID, "License verified")
                    if settings.STUDENT_ROLE_ID:
                        await remove_role(guild_member, settings.STUDENT_ROLE_ID, "Promoted")
            except Exception as e:
                logger.warning(f"Role change failed: {e}")

            # Post to channel
            await self._post_verification(member, match, state)

            # SMS confirmation
            agent = await self.db.get_agent(member.id)
            if agent and agent.get("phone_number"):
                await send_license_alert(
                    agent["phone_number"],
                    f"{first_name} {last_name}",
                    state, "verified",
                )

            # DM next steps
            await self._dm_next_steps(member)

            await interaction.followup.send(
                f"\u2705 **License Verified!**\n\n"
                f"**Name:** {match.full_name}\n"
                f"**License #:** {match.license_number or 'N/A'}\n"
                f"**NPN:** {match.npn or 'N/A'}\n"
                f"**State:** {state}\n"
                f"**Status:** {match.status}\n"
                f"**Type:** {match.license_type or 'N/A'}\n\n"
                f"You've been promoted to **Licensed Agent**!",
                ephemeral=True,
            )

        elif results and results[0].error:
            await interaction.followup.send(
                f"\u26a0\ufe0f **Lookup Error:** {results[0].error}\n\n"
                f"The {state} lookup may be temporarily unavailable. Try again later.",
                ephemeral=True,
            )
        else:
            await interaction.followup.send(
                "\u274c **Could not verify your license.**\n\n"
                "\u2022 Your name may not match what's on file\n"
                "\u2022 Your license may not be processed yet\n"
                "\u2022 You may be licensed in a different state\n\n"
                f"**Searched:** {first_name} {last_name} in {state}\n\n"
                "Contact your upline for manual verification.",
                ephemeral=True,
            )

    # ── /license-history — check your own verification history ──

    @app_commands.command(
        name="license-history",
        description="View your license check history",
    )
    async def license_history(self, interaction: discord.Interaction) -> None:
        await interaction.response.defer(ephemeral=True)

        checks = await self.db.get_check_history(interaction.user.id, limit=5)
        if not checks:
            await interaction.followup.send(
                "No license checks on file yet. Run `/verify` to get started!",
                ephemeral=True,
            )
            return

        lines = []
        for c in checks:
            emoji = "\u2705" if c["status"] == "verified" or c["status"] == "active" else "\u274c"
            lines.append(
                f"{emoji} **{c['state']}** — {c['status'].title()} "
                f"({c['check_date'][:10]})"
            )

        embed = discord.Embed(
            title="\U0001f4cb  License Check History",
            description="\n".join(lines),
            color=0x3498DB,
        )
        await interaction.followup.send(embed=embed, ephemeral=True)

    # ── Helpers ──

    async def _post_verification(self, user, match, state: str) -> None:
        channel_id = settings.LICENSE_CHECK_CHANNEL_ID or settings.HIRING_LOG_CHANNEL_ID
        channel = self.bot.get_channel(channel_id) if channel_id else None
        if not channel:
            return
        embed = discord.Embed(
            title="\U0001f393  License Verified!",
            description=(
                f"<@{user.id}> verified as a licensed agent.\n\n"
                f"**Name:** {match.full_name}\n"
                f"**License #:** {match.license_number or 'N/A'}\n"
                f"**State:** {state} | **Status:** {match.status}"
            ),
            color=0x2ECC71,
            timestamp=datetime.now(timezone.utc),
        )
        await channel.send(embed=embed)

    async def _dm_next_steps(self, member) -> None:
        try:
            embed = discord.Embed(
                title="\U0001f389  License Verified! Next Step: Contracting",
                description=(
                    f"Welcome **{member.display_name}**! Your license has been "
                    f"confirmed. Use `/contract` in the server to book your "
                    f"contracting appointment."
                ),
                color=0x9B59B6,
            )
            embed.add_field(
                name="\U0001f4dd  What to Prepare",
                value=(
                    "\u2022 Government-issued photo ID\n"
                    "\u2022 Social Security number\n"
                    "\u2022 E&O insurance info\n"
                    "\u2022 Bank info for direct deposit\n"
                    "\u2022 Resident state license number"
                ),
                inline=False,
            )
            await member.send(embed=embed)
        except discord.Forbidden:
            logger.warning(f"Cannot DM {member}")


async def setup(bot: commands.Bot) -> None:
    await bot.add_cog(VerifyCog(bot, bot._db))

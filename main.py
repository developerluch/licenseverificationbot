# VIPA License Bot — Entry point
# Separate Discord bot for license verification + monitoring.
# Shares a database with the onboarding bot.
import asyncio
import logging
import sys

import discord
from discord.ext import commands

from config import settings
from db import LicenseDB

# ── Logging ──
logging.basicConfig(
    level=getattr(logging, settings.LOG_LEVEL.upper(), logging.INFO),
    format="%(asctime)s %(levelname)-8s %(name)s: %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
    handlers=[logging.StreamHandler(sys.stdout)],
)
logger = logging.getLogger("license-bot")

# ── Bot setup ──
intents = discord.Intents.default()
intents.members = True       # Need member info for DMs and role changes
intents.message_content = False  # Don't need message content

bot = commands.Bot(
    command_prefix="!lic-",   # Prefix commands (rarely used, mostly slash)
    intents=intents,
    help_command=None,
)

# Shared DB reference
db = LicenseDB()
bot._db = db  # Accessible by cogs


@bot.event
async def on_ready():
    logger.info(f"License Bot online as {bot.user} (ID: {bot.user.id})")
    logger.info(f"Connected to {len(bot.guilds)} guild(s)")
    logger.info(f"Shared DB: {db._path()}")


async def setup_hook():
    """Load cogs and sync slash commands."""
    await db.init()

    # Load cogs
    cog_modules = [
        "cogs.verify",    # /verify command
        "cogs.monitor",   # Scheduled license checks
    ]
    for cog in cog_modules:
        try:
            await bot.load_extension(cog)
            logger.info(f"Loaded cog: {cog}")
        except Exception as e:
            logger.error(f"Failed to load {cog}: {e}")

    # Sync slash commands
    if settings.GUILD_ID:
        guild = discord.Object(id=settings.GUILD_ID)
        bot.tree.copy_global_to(guild=guild)
        await bot.tree.sync(guild=guild)
        logger.info(f"Synced commands to guild {settings.GUILD_ID}")
    else:
        await bot.tree.sync()
        logger.info("Synced commands globally")


bot.setup_hook = setup_hook


def main():
    logger.info("Starting VIPA License Bot...")
    bot.run(settings.DISCORD_TOKEN, log_handler=None)


if __name__ == "__main__":
    main()

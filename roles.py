# Role management helpers (shared pattern with onboarding-bot)
import logging

import discord

logger = logging.getLogger(__name__)


async def assign_role(
    member: discord.Member, role_id: int, reason: str = "License Bot"
) -> bool:
    """Assign a role to a member. Returns True on success."""
    if not role_id:
        return False
    role = member.guild.get_role(role_id)
    if not role:
        logger.warning(f"Role {role_id} not found in guild")
        return False
    if role in member.roles:
        return True  # Already has it
    try:
        await member.add_roles(role, reason=reason)
        logger.info(f"Assigned {role.name} to {member}")
        return True
    except discord.Forbidden:
        logger.error(f"No permission to assign {role.name} to {member}")
        return False


async def remove_role(
    member: discord.Member, role_id: int, reason: str = "License Bot"
) -> bool:
    """Remove a role from a member. Returns True on success."""
    if not role_id:
        return False
    role = member.guild.get_role(role_id)
    if not role:
        return False
    if role not in member.roles:
        return True  # Already doesn't have it
    try:
        await member.remove_roles(role, reason=reason)
        logger.info(f"Removed {role.name} from {member}")
        return True
    except discord.Forbidden:
        logger.error(f"No permission to remove {role.name} from {member}")
        return False

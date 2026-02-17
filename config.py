# VIPA License Bot — Configuration
import logging
from pydantic_settings import BaseSettings, SettingsConfigDict

logger = logging.getLogger(__name__)


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        case_sensitive=False,
    )

    # Discord
    DISCORD_TOKEN: str
    GUILD_ID: int

    # Channels
    LICENSE_CHECK_CHANNEL_ID: int = 0  # Where license check summaries go
    HIRING_LOG_CHANNEL_ID: int = 0     # Cross-post verifications here too

    # Roles (must match onboarding bot's role IDs)
    STUDENT_ROLE_ID: int = 0
    LICENSED_AGENT_ROLE_ID: int = 0

    # Twilio SMS
    TWILIO_ACCOUNT_SID: str = ""
    TWILIO_AUTH_TOKEN: str = ""
    TWILIO_PHONE_NUMBER: str = ""  # E.164 format: +1XXXXXXXXXX

    # License monitoring schedule
    LICENSE_CHECK_INTERVAL_HOURS: int = 168  # Default: weekly
    LICENSE_CHECK_ENABLED: bool = True

    # Shared database (MUST be the same DB as onboarding-bot)
    DATABASE_URL: str = "sqlite:///onboarding.db"

    # Logging
    LOG_LEVEL: str = "INFO"


settings = Settings()

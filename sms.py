# Twilio SMS helper for license monitoring notifications
import logging
from typing import Optional

logger = logging.getLogger(__name__)

# Twilio is optional — import lazily
_twilio_client = None


def _get_client():
    """Lazy-load Twilio client."""
    global _twilio_client
    if _twilio_client is not None:
        return _twilio_client

    try:
        from twilio.rest import Client
        from config import settings

        if not settings.TWILIO_ACCOUNT_SID or not settings.TWILIO_AUTH_TOKEN:
            logger.warning("Twilio credentials not configured — SMS disabled")
            return None

        _twilio_client = Client(settings.TWILIO_ACCOUNT_SID, settings.TWILIO_AUTH_TOKEN)
        logger.info("Twilio client initialized")
        return _twilio_client

    except ImportError:
        logger.warning("twilio package not installed — run: pip install twilio")
        return None
    except Exception as e:
        logger.error(f"Twilio init error: {e}")
        return None


async def send_sms(to: str, body: str) -> Optional[str]:
    """Send an SMS via Twilio.

    Args:
        to: Phone number in E.164 format (e.g. +15551234567)
        body: Message text (max 1600 chars, will be split into segments)

    Returns:
        Message SID on success, None on failure.
    """
    from config import settings

    client = _get_client()
    if not client:
        logger.warning(f"SMS not sent (no client): {to} — {body[:50]}...")
        return None

    if not settings.TWILIO_PHONE_NUMBER:
        logger.warning("TWILIO_PHONE_NUMBER not configured")
        return None

    try:
        # Twilio is synchronous — run in executor to not block the event loop
        import asyncio
        loop = asyncio.get_event_loop()
        message = await loop.run_in_executor(
            None,
            lambda: client.messages.create(
                body=body,
                from_=settings.TWILIO_PHONE_NUMBER,
                to=to,
            ),
        )
        logger.info(f"SMS sent to {to}: SID={message.sid}")
        return message.sid

    except Exception as e:
        logger.error(f"SMS send failed to {to}: {e}")
        return None


async def send_license_alert(
    phone: str,
    agent_name: str,
    state: str,
    status: str,
    details: str = "",
) -> Optional[str]:
    """Send a license status alert SMS.

    Templates for different alert types.
    """
    if status == "expired":
        body = (
            f"VIPA License Alert: {agent_name}, your {state} insurance license "
            f"appears to have EXPIRED. Please renew ASAP to stay compliant. "
            f"Contact your upline if you need help. {details}"
        )
    elif status == "not_found":
        body = (
            f"VIPA License Check: {agent_name}, we could not verify your "
            f"{state} insurance license. Please confirm your info is correct "
            f"by running /verify in the VIPA Discord. {details}"
        )
    elif status == "expiring_soon":
        body = (
            f"VIPA Reminder: {agent_name}, your {state} insurance license is "
            f"expiring soon. Make sure to renew before the expiration date "
            f"to avoid any gaps in coverage. {details}"
        )
    elif status == "verified":
        body = (
            f"VIPA: {agent_name}, your {state} insurance license has been "
            f"verified! You're all set. Welcome to the team!"
        )
    else:
        body = (
            f"VIPA License Update: {agent_name}, your {state} license status "
            f"is: {status}. {details}"
        )

    return await send_sms(phone, body.strip())

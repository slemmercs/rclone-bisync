import pytest
from pydantic import BaseModel
from typing import List, Dict, Optional
from rclone_bisync.core.security import (
    redact_secrets,
    is_sensitive_key_name,
    is_sensitive_value,
    REDACTED_STRING,
)


def test_is_sensitive_key_name():
    assert is_sensitive_key_name("RC_PASS") is True
    assert is_sensitive_key_name("CLIENT_SECRET") is True
    assert is_sensitive_key_name("auth_token") is True
    assert is_sensitive_key_name("DISCORD_WEBHOOK_URL") is True
    assert is_sensitive_key_name("PRIVATE_KEY") is True
    assert is_sensitive_key_name("api_key") is True
    assert is_sensitive_key_name("user_credentials") is True

    assert is_sensitive_key_name("LOCAL_PATH") is False
    assert is_sensitive_key_name("REMOTE_NAME") is False
    assert is_sensitive_key_name("status") is False
    assert is_sensitive_key_name("speed_str") is False


def test_is_sensitive_value():
    assert is_sensitive_value("ya29.a0AfH6SMD-example-token-123") is True
    assert is_sensitive_value("ghp_123456789012345678901234567890123456") is True
    assert is_sensitive_value("xoxb-1234567890-123456789012-abcdefghijklmnop") is True
    assert is_sensitive_value("regular_text_value") is False
    assert is_sensitive_value("https://127.0.0.1:5572") is False


def test_redact_secrets_nested_dicts_and_lists():
    sample_payload = {
        "status": "healthy",
        "remotes": [
            {
                "name": "gdrive",
                "type": "drive",
                "client_id": "public_client_id_123",
                "client_secret": "super_secret_client_pass",
                "nested_token": {
                    "access_token": "ya29.sample_oauth_token",
                    "expiry": 3600,
                },
            }
        ],
        "system": {
            "password": "system_admin_pass",
            "safe_counter": 42,
            "sub_array": ["item1", "ya29.leaked_token_in_list", "item3"],
        },
        "empty_pass": "",
        "none_token": None,
    }

    sanitized = redact_secrets(sample_payload)

    # Validate that original payload was not modified in-place or mutated destructively
    assert sample_payload["system"]["password"] == "system_admin_pass"

    # Validate recursive redaction
    assert sanitized["status"] == "healthy"
    assert sanitized["remotes"][0]["client_id"] == "public_client_id_123"
    assert sanitized["remotes"][0]["client_secret"] == REDACTED_STRING
    assert sanitized["remotes"][0]["nested_token"]["access_token"] == REDACTED_STRING
    assert sanitized["remotes"][0]["nested_token"]["expiry"] == 3600

    assert sanitized["system"]["password"] == REDACTED_STRING
    assert sanitized["system"]["safe_counter"] == 42
    assert sanitized["system"]["sub_array"][0] == "item1"
    assert sanitized["system"]["sub_array"][1] == REDACTED_STRING
    assert sanitized["system"]["sub_array"][2] == "item3"

    assert sanitized["empty_pass"] == ""
    assert sanitized["none_token"] is None


def test_redact_secrets_pydantic_model():
    class SubConfig(BaseModel):
        token: str
        enabled: bool

    class ConfigPayload(BaseModel):
        user: str
        sub: SubConfig
        tags: List[str]

    model_instance = ConfigPayload(
        user="alice",
        sub=SubConfig(token="secret_auth_token_value", enabled=True),
        tags=["production", "backup"],
    )

    sanitized = redact_secrets(model_instance)
    assert sanitized["user"] == "alice"
    assert sanitized["sub"]["token"] == REDACTED_STRING
    assert sanitized["sub"]["enabled"] is True
    assert sanitized["tags"] == ["production", "backup"]


def test_redact_secrets_primitives():
    assert redact_secrets("normal_str") == "normal_str"
    assert redact_secrets("ya29.token") == REDACTED_STRING
    assert redact_secrets(12345) == 12345
    assert redact_secrets(True) is True
    assert redact_secrets(None) is None

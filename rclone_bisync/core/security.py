"""
Core security utilities: recursive credential/secret sanitization and protection.
"""

from typing import Any, Set
import re

# Standard sensitive substrings across configuration and environment vectors
SENSITIVE_KEYWORDS: Set[str] = {
    "PASS",
    "SECRET",
    "TOKEN",
    "KEY",
    "CREDENTIAL",
    "AUTH",
    "WEBHOOK",
    "PRIVATE",
}

# Regex matching patterns that appear to be secrets even if the key name is obscure
SENSITIVE_VALUE_REGEXES = [
    re.compile(r"ghp_[A-Za-z0-9_]{36,}"),                 # GitHub Personal Access Token
    re.compile(r"xox[baprs]-[0-9a-zA-Z-]{10,}"),         # Slack Token
    re.compile(r"ya29\.[0-9A-Za-z\-_]+"),                 # Google OAuth Access Token
]

REDACTED_STRING = "********"


def is_sensitive_key_name(key: str) -> bool:
    """Checks whether a key identifier hints at sensitive data."""
    if not isinstance(key, str):
        return False
    key_upper = key.upper()
    return any(keyword in key_upper for keyword in SENSITIVE_KEYWORDS)


def is_sensitive_value(value: Any) -> bool:
    """Checks whether a string value matches known high-risk token signatures."""
    if not isinstance(value, str):
        return False
    return any(pattern.search(value) for pattern in SENSITIVE_VALUE_REGEXES)


def redact_secrets(data: Any) -> Any:
    """
    Recursively inspects data structures (dicts, lists, tuples, sets, Pydantic models)
    and sanitizes values associated with sensitive keys or matching sensitive token formats.
    """
    # Handle Pydantic models
    if hasattr(data, "model_dump") and callable(data.model_dump):
        # Pydantic v2
        dumped = data.model_dump()
        return redact_secrets(dumped)
    elif hasattr(data, "dict") and callable(data.dict):
        # Pydantic v1
        dumped = data.dict()
        return redact_secrets(dumped)

    if isinstance(data, dict):
        sanitized = {}
        for k, v in data.items():
            if isinstance(v, (dict, list, tuple, set)):
                # Traverse nested containers even if the parent key has "token" or "credentials" in its name
                sanitized[k] = redact_secrets(v)
            elif is_sensitive_key_name(str(k)):
                if v is not None and v != "":
                    sanitized[k] = REDACTED_STRING
                else:
                    sanitized[k] = v
            else:
                sanitized[k] = redact_secrets(v)
        return sanitized

    elif isinstance(data, list):
        return [redact_secrets(item) for item in data]

    elif isinstance(data, tuple):
        return tuple(redact_secrets(item) for item in data)

    elif isinstance(data, set):
        return {redact_secrets(item) for item in data}

    elif isinstance(data, str):
        if is_sensitive_value(data):
            return REDACTED_STRING
        return data

    return data

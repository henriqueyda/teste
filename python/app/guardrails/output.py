from __future__ import annotations

import re

# CPF: 000.000.000-00 or 00000000000 (11 digits, optionally formatted)
_CPF_FORMATTED = re.compile(r"\b\d{3}\.\d{3}\.\d{3}-\d{2}\b")
_CPF_RAW = re.compile(r"\b\d{11}\b")

# Credit card: 16 digits optionally separated by spaces or hyphens
_CARD = re.compile(r"\b(\d{4})[\s\-]?(\d{4})[\s\-]?(\d{4})[\s\-]?(\d{4})\b")


def _mask_card(m: re.Match[str]) -> str:
    return f"****-****-****-{m.group(4)}"


def redact_pii(text: str) -> str:
    """Redact Brazilian PII (CPF, credit card numbers) from LLM output."""
    text = _CPF_FORMATTED.sub("***.***.***-**", text)
    text = _CPF_RAW.sub("***.***.***-**", text)
    text = _CARD.sub(_mask_card, text)
    return text

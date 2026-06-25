from __future__ import annotations

import re

_INJECTION_PATTERNS: list[re.Pattern[str]] = [
    re.compile(p, re.IGNORECASE)
    for p in [
        # EN: classic instruction override
        r"ignore\s+(previous|prior|above|all)\s+instructions",
        # PT: ignore / desconsidere as instruções
        r"(ignore|desconsidere|esqueça)\s+(as\s+)?(anteriores?|todas(\s+as)?|acima\s+das?)\s+instru[çc][oõ]es",
        r"(ignore|desconsidere)\s+(todas\s+)?(suas\s+)?instru[çc][oõ]es",

        # EN: memory wipe
        r"forget\s+(everything|what\s+you\s+know|all\s+previous)",
        # PT: esqueça tudo / o que foi dito
        r"esque[çc]a\s+(tudo|o\s+que\s+(voc[eê]\s+)?sabe|tudo\s+que\s+foi\s+dito)",

        # EN: persona hijack
        r"\byou\s+are\s+now\b|\bact\s+as\b|\bpretend\s+to\s+be\b|\broleplay\s+as\b",
        # PT: você é agora / aja como / finja ser / se comporte como
        r"\bvoc[eê]\s+[eé]\s+agora\b|\baja\s+como\b|\bfinja\s+(ser|que)\b|\bse\s+comporte\s+como\b|\binterprete\s+o\s+papel\b",

        # EN: instruction extraction
        r"reveal\s+.{0,30}instructions|what\s+are\s+your\s+.{0,20}instructions",
        # PT: revele suas instruções / quais são suas instruções
        r"revele\s+(suas\s+)?instru[çc][oõ]es|quais\s+s[aã]o\s+(suas\s+)?instru[çc][oõ]es",
        r"mostre\s+(seu\s+)?(prompt|instru[çc][oõ]es|sistema)",

        # Token injection (language-agnostic)
        r"<\|im_start\||<\|endoftext\|>|<\|INST\|>|\[INST\]",

        # JSON role injection (language-agnostic)
        r'\{"role"\s*:\s*"(system|assistant)"',

        # XML/MCP tag injection (language-agnostic)
        r"</?(?:tool_call|tool_result|function)[>\s]",

        # EN: DAN jailbreak
        r"\bDAN\b.{0,20}do\s+anything|do\s+anything\s+now",

        # EN: disregard rules
        r"disregard\s+(all|any|your|previous)\s+(instructions|rules|constraints)",
        # PT: desconsidere as regras / restrições
        r"desconsidere\s+(todas\s+)?(as\s+)?(suas\s+)?(instru[çc][oõ]es|regras|restri[çc][oõ]es)",

        # EN: new directive injection
        r"new\s+(instruction|directive|rule|task)s?\s*:",
        # PT: nova instrução / diretiva
        r"nova\s+(instru[çc][aã]o|diretiva|regra|tarefa)\s*:",
    ]
]

_REJECTION = "Não consigo processar essa solicitação."


def check_prompt_injection(text: str) -> str | None:
    """Return a rejection message if the text looks like a prompt injection attempt, else None."""
    for pattern in _INJECTION_PATTERNS:
        if pattern.search(text):
            return _REJECTION
    return None

"""SOFT guardrails (non-authoritative; UX/first-pass only).

Input: prompt-injection heuristics. Output: PII redaction, grounding/citation check.
These improve UX and catch obvious abuse, but the HARD guarantees (authz, ownership,
limits, step-up, audit) live in the Go tier and hold even if these are bypassed.
Implemented in M8.
"""

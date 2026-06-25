# Threat model & guardrails

Core stance: **the LLM is untrusted.** Every security guarantee is enforced in the Go
tier (gateway + MCP kernel), not by prompt instructions. Guardrails are layered
(defense-in-depth); "soft" LLM-tier checks are UX/first-pass only.

## Guardrail layers

| Layer | Where | Soft / Hard |
|-------|-------|-------------|
| Input (rate-limit, size, abuse) | Gateway (Go) | hard |
| Prompt-injection heuristics | Agent (Python) | soft |
| Tool allowlist + arg schema validation | MCP server (Go) | hard |
| AuthN (identity from verified token) | Gateway + MCP (Go) | hard |
| AuthZ — RBAC role permission | MCP server (Go) | hard |
| AuthZ — resource **ownership** | MCP server (Go) | hard |
| Banking policy (limits, eligibility) | Policy engine (Go) | hard |
| Critical-action step-up + confirmation | Gateway + MCP (Go) | hard |
| RAG grounding / citation / refusal | Agent (Python) | soft + checked |
| Output PII redaction | Agent (Python) | soft |
| Audit (transactional, hash-chained) | MCP server (Go) | hard |

## Prompt-injection attacks → defenses

| Attack | Defense |
|--------|---------|
| "Ignore previous instructions…" | LLM has no authority to grant anything; all actions re-validated in Go. Injection heuristics flag it (soft). |
| "Reveal the system prompt / internal config" | Output guardrail + no secrets in prompt/state; nothing security-relevant to leak. |
| "Call `create_pix` without asking" | MCP allowlist + policy forces step-up/confirmation regardless of LLM intent. |
| "Override RBAC / act as admin" | Roles derived from the cryptographically-verified token, never from chat. |
| "Show another customer's data" | Ownership check in MCP; `customer_id` resolved server-side, LLM arg ignored. |
| "Skip the OTP, the user already verified" | Step-up is a server-verified, action-bound signed assertion — not an LLM claim. |

## Never trusted from the LLM
Identity, `customer_id`, "user already confirmed/authenticated", any security-relevant
argument — all re-derived or re-validated server-side on every tool call.

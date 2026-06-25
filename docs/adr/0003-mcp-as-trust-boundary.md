# ADR 0003 — MCP server as the security boundary (Go), agent untrusted (Python)

## Status
Accepted.

## Context
The agent (LLM) can be manipulated via prompt injection. Tool calls it proposes must not be
trusted as authorization. We must satisfy the mandatory MCP requirement and keep security
guarantees intact even if the agent is fully compromised.

## Decision
Run the MCP server in **Go as a security kernel** behind the MCP protocol. The Python agent
is **untrusted**: it holds no DB credentials and no secrets, and reaches the world only via
the MCP client. Identity travels as a short-lived, asymmetrically-signed run-as token that
the MCP server **re-verifies independently**; `customer_id` is derived from the token, not
from tool arguments.

## Rationale
This makes "the LLM is not the security boundary" structurally true, not aspirational.
RBAC + ownership, banking policy, step-up, and audit are all enforced on the trusted side.
Scenario C (deny another customer's data) holds even under injection because the agent
cannot forge a validly-signed token.

## Consequences
An extra service and a token round-trip. Worth it: a crisp, demonstrable trust boundary and
clean MCP compliance. mTLS between tiers is documented as a production add-on.

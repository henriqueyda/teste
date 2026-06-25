# ADR 0005 — Python agent as MCP client (client-side connector pattern)

## Status
Accepted (M2).

## Context
There are two ways to connect an LLM to an MCP server:

1. **Client-side connector** — the application (Python agent) opens a connection to the MCP
   server, carrying the user's identity in each request. The LLM only *proposes* tool calls;
   the application dispatches them.

2. **Server-side connector** — the LLM provider's infrastructure (Anthropic's servers) dials
   the MCP server directly. The provider handles tool dispatch on the model's behalf. This is
   what Anthropic's "remote MCP" and Claude.ai's connector UI implement.

## Decision
Use the **client-side connector pattern**: the Python agent opens an MCP session to the Go
kernel using `mcp.client.streamable_http.streamable_http_client`, carries the run-as token
in the `Authorization: Bearer` header, and dispatches tool calls synchronously within the
LangGraph loop. The LLM never sees the token and cannot influence it.

## Rationale

### Why not the server-side (Anthropic-hosted) connector?

1. **Identity cannot flow**: The server-side connector authenticates with a *service* credential
   that Anthropic controls — there is no mechanism to carry a *per-user* run-as token. The Go
   kernel enforces access per customer identity; a single service credential would collapse all
   users to the same identity (or require the kernel to trust the LLM's claim about identity,
   which violates our core security principle).

2. **The kernel must not be internet-reachable**: Our MCP server is an internal security
   boundary. Exposing it to Anthropic's infrastructure would require making port 7070 publicly
   reachable and trusting Anthropic's network as a perimeter — unacceptable for a banking
   system.

3. **Audit and control**: Every tool call must be audited inside the same database transaction
   as the action (the "transactional audit" requirement). With a server-side connector, the
   provider dispatches calls asynchronously and we cannot guarantee they arrive inside an open
   transaction with the right identity. With a client-side connector, the Python agent drives
   the loop and the Go kernel owns the transaction.

4. **Vendor lock-in**: The client-side pattern works with any model that supports tool-use
   (including open-weight models). The server-side pattern is Anthropic-specific.

### Why not expose tools as plain REST endpoints (skip MCP entirely)?
MCP is a hard challenge requirement. Beyond compliance, it gives us a standard protocol that
any client (Inspector, VS Code extension, future agents) can use without knowing our internal
API shape.

## Consequences
- The Python agent is the only MCP *client*; it has no DB credentials, no secrets, and no
  authority beyond the opaque run-as token it received from the gateway.
- The LLM sees tool schemas (name + description + input_schema) but never sees the token, the
  customer_id, or any value it could inject into the tool call to influence authorization.
- Adding a new LLM provider (e.g. Gemini) requires only swapping the `LLMInterface`
  implementation; the MCP client, graph, and security kernel are unchanged.
- `mcp>=1.0` is the only MCP dependency on the Python side (no Anthropic-specific MCP SDK).

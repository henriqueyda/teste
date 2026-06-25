# ADR 0004 — MCP transport: official Streamable HTTP (revised M2)

## Status
Supersedes: M1 decision (minimal hand-rolled JSON-RPC over HTTP).  
Accepted (M2).

## Context
The MCP server must expose tools to the Python agent. MCP is JSON-RPC 2.0 with methods
`tools/list` and `tools/call` over a transport (stdio or Streamable HTTP/SSE). In M1 we
shipped a hand-rolled implementation that was MCP-compatible at the method level but lacked
the mandatory `initialize` handshake and Streamable HTTP framing — meaning no real MCP
client could connect.

When the Python agent arrived in M2 (using the official `mcp` Python SDK), it required a
spec-compliant server to connect to.

## Decision
In M2, rewrite the Go MCP server using the **official Go SDK**
(`github.com/modelcontextprotocol/go-sdk v1.6.1`). Expose tools via
`mcp.NewStreamableHTTPHandler` mounted at `/mcp`. Tool handlers receive the incoming HTTP
request headers via `req.Extra.Header`, so the run-as token (`Authorization: Bearer`) still
flows out-of-band from the LLM context.

The Python agent uses `mcp.client.streamable_http.streamable_http_client` with a custom
`httpx.AsyncClient` that injects `Authorization: Bearer <run-as-token>` on every request.

## Rationale
- The challenge lists "MCP protocol" as a hard requirement; spec compliance is the minimum
  bar. A real MCP Inspector or client SDK must be able to connect and call `initialize`.
- The Go SDK handles all protocol framing; we write only tool-handler logic, not JSON-RPC
  plumbing. This is strictly less code.
- `req.Extra.Header` gives tool handlers direct access to the HTTP request headers, so the
  identity flow is unchanged: token → header → handler → `VerifyInternalToken` → Subject.
  No fragile context threading required.
- The Python SDK's `streamable_http_client` accepts a custom `httpx.AsyncClient`, so
  injecting the per-user run-as token in headers is a one-liner.

## Consequences
- The hand-rolled `internal/mcp/jsonrpc.go` and custom registry/dispatch are deleted.
- All M1 security guarantees (RBAC, transactional audit, ownership-ready design) are
  preserved behind the new transport — the `runToolPipeline` helper is unchanged.
- Any MCP-compliant client (Inspector, Python SDK, future mobile app) can now connect
  without needing to understand our internal HTTP/JSON-RPC contract.
- The run-as token remains in the `Authorization` header, never in the tool arguments or
  LLM context. The kernel never trusts identity from the LLM.

"""MCP client: discovers tools and dispatches calls to the Go MCP server.

The run-as token travels as the Authorization Bearer header — out-of-band from the
LLM context — and is re-verified by the Go kernel on every call. This tier has no
DB credentials; its only authority is the opaque token it received from the gateway.

One ClientSession is opened per /invoke call (acceptable for M2; pooling is M3+).
"""
from __future__ import annotations
import json
from typing import Any

import httpx
from mcp import ClientSession
from mcp.client.streamable_http import streamable_http_client
from opentelemetry.propagate import inject


class MCPClient:
    def __init__(self, mcp_url: str, run_as_token: str) -> None:
        self._url = mcp_url
        self._token = run_as_token

    def _http(self) -> httpx.AsyncClient:
        headers: dict[str, str] = {"Authorization": f"Bearer {self._token}"}
        inject(headers)  # adds traceparent (W3C TraceContext) for Go MCP child spans
        return httpx.AsyncClient(headers=headers, timeout=30.0)

    async def call_tool(self, name: str, arguments: dict[str, Any]) -> tuple[str, bool]:
        """Call a tool. Returns (result_text, is_error)."""
        async with streamable_http_client(self._url, http_client=self._http()) as (r, w, _):
            async with ClientSession(r, w) as session:
                await session.initialize()
                result = await session.call_tool(name, arguments)
                parts: list[str] = []
                for block in result.content or []:
                    if hasattr(block, "text"):
                        parts.append(block.text)
                    else:
                        parts.append(json.dumps(block, default=str))
                return "\n".join(parts), bool(result.isError)

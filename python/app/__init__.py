"""Banking agent — untrusted reasoning tier.

Holds NO database credentials and NO banking secrets. Its only path to the world is
the MCP client. Tokens are opaque here and are never shown to the LLM. All security
guarantees live in the Go tier; anything in this package is non-authoritative.
"""

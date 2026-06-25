"""Graph nodes (small, testable functions):

- router:     classify intent (RAG question vs banking operation).
- retrieve:   RAG retrieval + grounding/citation/refusal (M5).
- plan_tool:  let the LLM propose a tool call (M2).
- guardrails: soft input/output checks; non-authoritative (M8).
- confirm:    raise interrupt() for confirmation / step-up (M6/M7).
- call_tool:  dispatch the proposed call via the MCP client (M2).
- respond:    final grounded answer.
"""

"""Agent state schema.

IMPORTANT: The run_as_token is NEVER stored here. It lives in
config["configurable"]["run_as_token"] — the LangGraph administrative side-channel
that is not persisted to the checkpointer and is never visible to the LLM.
"""
from __future__ import annotations
from typing import Annotated, Sequence, TypedDict

from langchain_core.messages import BaseMessage
from langgraph.graph.message import add_messages


class AgentState(TypedDict):
    messages: Annotated[Sequence[BaseMessage], add_messages]

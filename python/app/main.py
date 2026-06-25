"""FastAPI entrypoint for the agent service.

POST /invoke — main chat endpoint.
  - Sends user message to the LangGraph agent.
  - When the graph is paused (awaiting confirmation or transaction PIN session token),
    the next message is treated as the resume value via Command(resume=...).
  - Returns awaiting_confirmation=True for yes/no confirmations.
  - Returns awaiting_pin=True when step_up_node is waiting for the opaque session
    token that the Go gateway /step-up endpoint issues after PIN verification.

The raw PIN never reaches this service or the LangGraph checkpoint.
The run-as token travels as X-Run-As-Token header — never in the LLM context.
Conversation history is persisted in Postgres via the LangGraph checkpointer.
"""
from __future__ import annotations

from contextlib import asynccontextmanager

from fastapi import FastAPI, Header, HTTPException
from psycopg_pool import AsyncConnectionPool
from pydantic import BaseModel
from langgraph.checkpoint.postgres.aio import AsyncPostgresSaver
from langgraph.types import Command

from opentelemetry import trace as otel_trace

from .config import settings
from .graph.build import build_graph
from .guardrails.input import check_prompt_injection
from .guardrails.output import redact_pii
from .rag.retriever import init_retriever
from .telemetry import setup_telemetry

_tracer = otel_trace.get_tracer("banking-agent")

_graph = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    global _graph
    setup_telemetry(settings.otel_exporter_otlp_endpoint)
    pool = AsyncConnectionPool(
        conninfo=settings.checkpoint_db_url,
        max_size=10,
        open=False,
        kwargs={"autocommit": True},
    )
    await pool.open()
    checkpointer = AsyncPostgresSaver(pool)
    await checkpointer.setup()
    init_retriever(settings.faiss_index_path, settings.google_api_key)
    _graph = build_graph(checkpointer)
    yield
    await pool.close()


app = FastAPI(title="Banking Agent", version="0.5.0", lifespan=lifespan)


class InvokeRequest(BaseModel):
    message: str
    thread_id: str = ""


class InvokeResponse(BaseModel):
    reply: str
    awaiting_confirmation: bool = False
    awaiting_pin: bool = False


@app.get("/healthz")
async def healthz() -> dict:
    return {"status": "ok", "model": settings.google_model}


@app.post("/invoke", response_model=InvokeResponse)
async def invoke(
    req: InvokeRequest,
    x_run_as_token: str = Header(alias="X-Run-As-Token"),
    x_correlation_id: str = Header(default="", alias="X-Correlation-Id"),
) -> InvokeResponse:
    if not x_run_as_token:
        raise HTTPException(status_code=401, detail="missing run-as token")

    thread_id = req.thread_id or x_correlation_id or "default"
    config = {"configurable": {"thread_id": thread_id, "run_as_token": x_run_as_token}}

    snap = await _graph.aget_state(config)
    is_paused = any(t.interrupts for t in (snap.tasks or []))

    if not is_paused:
        if rejection := check_prompt_injection(req.message):
            return InvokeResponse(reply=rejection)

    inp = Command(resume=req.message) if is_paused else {"messages": [{"role": "user", "content": req.message}]}
    with _tracer.start_as_current_span("agent.invoke"):
        final = await _graph.ainvoke(inp, config=config)

    if interrupts := final.get("__interrupt__"):
        iv = interrupts[0].value
        if isinstance(iv, dict) and iv.get("type") == "pin_required":
            return InvokeResponse(reply=iv.get("message", "Autenticação necessária"), awaiting_pin=True)
        return InvokeResponse(reply=redact_pii(str(iv)), awaiting_confirmation=True)

    content = final["messages"][-1].content
    if isinstance(content, list):
        reply = "\n".join(b["text"] for b in content if isinstance(b, dict) and b.get("type") == "text")
    else:
        reply = str(content)
    return InvokeResponse(reply=redact_pii(reply))

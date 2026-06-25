"""LangGraph ReAct agent — M7: PIX step-up (transaction PIN) on top of M6 HIL confirmation.

Flow:
  llm ──(read tools)──────────────► tools ──────────────────────► llm
      ├──(write tools)──► confirm ──► tools ──(pin_required?)──► step_up ──► llm
      │                          └──(denied)──► llm             └──(normal)──► llm
      └──(no tools)───────────────────────────────────────────────────────── END

Write tools (update_card_limit, create_pix) are intercepted before ToolNode.
confirm_node interrupts to ask "confirm? (sim/não)".
For create_pix, the MCP server returns pin_required when no PIN is supplied.
step_up_node then interrupts to collect the 4-digit transaction PIN and retries
create_pix with the PIN — PIN verification stays server-side (Go), never in the LLM.

Requires Python 3.11+ (interrupt() uses ContextVar propagation unavailable on 3.10).
"""
from __future__ import annotations

import json

from langchain_core.messages import SystemMessage, ToolMessage
from langchain_core.runnables import RunnableConfig
from langchain_core.tools import tool
from langgraph.graph import END, StateGraph
from langgraph.prebuilt import ToolNode
from langgraph.types import interrupt
from opentelemetry import trace as otel_trace

from ..config import settings
from ..llm import get_model
from ..mcp.client import MCPClient
from ..rag.retriever import get_retriever
from .state import AgentState

_tracer = otel_trace.get_tracer("banking-agent")

_SYSTEM = SystemMessage(content=(
    "Você é um assistente bancário do Itaú. Responda sempre em português do Brasil. "
    "Use as ferramentas disponíveis para obter dados do cliente — nunca invente ou suponha "
    "valores como saldo, nome ou documento. "
    "Para perguntas sobre políticas, taxas ou regras do banco, use search_knowledge_base. "
    "Se a informação não estiver na base de conhecimento, diga que não pode responder. "
    "Ao citar informações da base de conhecimento, mencione a fonte. "
    "Seja direto e objetivo."
))

# Tools that mutate state and require explicit user confirmation before ToolNode runs them.
_WRITE_TOOLS = {"update_card_limit", "create_pix"}


@tool
async def get_account_balance(config: RunnableConfig) -> str:
    """Consulta o saldo da conta corrente do cliente autenticado."""
    token = config["configurable"]["run_as_token"]
    result, _ = await MCPClient(settings.mcp_server_url, token).call_tool("get_account_balance", {})
    return result


@tool
async def get_customer_profile(config: RunnableConfig) -> str:
    """Consulta o perfil e dados cadastrais do cliente autenticado."""
    token = config["configurable"]["run_as_token"]
    result, _ = await MCPClient(settings.mcp_server_url, token).call_tool("get_customer_profile", {})
    return result


@tool
async def get_card_limit(card_id: str, config: RunnableConfig) -> str:
    """Consulta o limite do cartão de crédito pelo ID do cartão."""
    token = config["configurable"]["run_as_token"]
    result, _ = await MCPClient(settings.mcp_server_url, token).call_tool(
        "get_card_limit", {"card_id": card_id}
    )
    return result


@tool
async def update_card_limit(card_id: str, new_limit_cents: int, config: RunnableConfig) -> str:
    """Atualiza o limite do cartão de crédito. new_limit_cents em centavos (ex: 100000 = R$1.000,00)."""
    token = config["configurable"]["run_as_token"]
    result, _ = await MCPClient(settings.mcp_server_url, token).call_tool(
        "update_card_limit", {"card_id": card_id, "new_limit_cents": new_limit_cents}
    )
    return result


@tool
async def create_pix(recipient_key: str, amount_cents: int, config: RunnableConfig) -> str:
    """Envia uma transferência PIX. recipient_key é a chave PIX (CPF, e-mail, telefone ou chave aleatória). amount_cents em centavos (ex: 20000 = R$200,00)."""
    token = config["configurable"]["run_as_token"]
    result, _ = await MCPClient(settings.mcp_server_url, token).call_tool(
        "create_pix", {"recipient_key": recipient_key, "amount_cents": amount_cents}
    )
    return result


@tool
async def search_knowledge_base(query: str) -> str:
    """Busca políticas bancárias, taxas de juros e regras na base de conhecimento interna do Itaú."""
    r = get_retriever()
    if r is None:
        return "Base de conhecimento não disponível no momento."
    return r.search(query)


TOOLS = [get_account_balance, get_customer_profile, get_card_limit, update_card_limit, create_pix, search_knowledge_base]


def _is_pin_required(content: str) -> bool:
    try:
        return json.loads(content).get("error") == "pin_required"
    except (json.JSONDecodeError, TypeError, AttributeError):
        return False


def _fmt_brl(cents: int) -> str:
    return f"R$ {cents / 100:,.2f}".replace(",", "X").replace(".", ",").replace("X", ".")


def build_graph(checkpointer):
    model = get_model().bind_tools(TOOLS)

    def _route_llm(state: AgentState) -> str:
        tool_calls = getattr(state["messages"][-1], "tool_calls", None) or []
        if not tool_calls:
            return END
        if any(tc["name"] in _WRITE_TOOLS for tc in tool_calls):
            return "confirm"
        return "tools"

    def _route_confirm(state: AgentState) -> str:
        return "llm" if isinstance(state["messages"][-1], ToolMessage) else "tools"

    def _route_tools(state: AgentState) -> str:
        for msg in reversed(state["messages"]):
            if isinstance(msg, ToolMessage):
                if _is_pin_required(msg.content):
                    return "step_up"
            elif getattr(msg, "tool_calls", None):
                break
        return "llm"

    async def llm_node(state: AgentState, config: RunnableConfig) -> dict:
        with _tracer.start_as_current_span("llm_node") as span:
            messages = [_SYSTEM] + list(state["messages"])
            result = await model.ainvoke(messages, config=config)
            usage = getattr(result, "usage_metadata", None) or {}
            if usage:
                input_tok = usage.get("input_tokens", 0)
                output_tok = usage.get("output_tokens", 0)
                span.set_attribute("llm.input_tokens", input_tok)
                span.set_attribute("llm.output_tokens", output_tok)
            return {"messages": [result]}

    async def confirm_node(state: AgentState) -> dict:
        with _tracer.start_as_current_span("confirm_node"):
            last = state["messages"][-1]
            write_calls = [tc for tc in last.tool_calls if tc["name"] in _WRITE_TOOLS]
            tc = write_calls[0]
            if tc["name"] == "update_card_limit":
                formatted = _fmt_brl(tc["args"]["new_limit_cents"])
                question = f"Confirma a alteração do limite do cartão {tc['args']['card_id']} para {formatted}? (sim/não)"
            elif tc["name"] == "create_pix":
                formatted = _fmt_brl(tc["args"]["amount_cents"])
                question = f"Confirma o envio de PIX de {formatted} para {tc['args']['recipient_key']}? (sim/não)"
            else:
                question = f"Confirma a execução de '{tc['name']}'? (sim/não)"
            answer: str = interrupt(question)
            if answer.strip().lower() in ("sim", "s", "yes", "confirmo", "ok"):
                return {}
            return {
                "messages": [
                    ToolMessage(content="Operação cancelada pelo usuário.", tool_call_id=c["id"])
                    for c in write_calls
                ]
            }

    async def step_up_node(state: AgentState, config: RunnableConfig) -> dict:
        with _tracer.start_as_current_span("step_up_node"):
            tm = next(
                (m for m in reversed(state["messages"]) if isinstance(m, ToolMessage) and _is_pin_required(m.content)),
                None,
            )
            if tm is None:
                return {}

            token: str = interrupt({"type": "pin_required", "message": "Autenticação de senha de transação necessária"})

            ai_msg = next(
                (m for m in reversed(state["messages"])
                 if getattr(m, "tool_calls", None) and any(tc["name"] == "create_pix" for tc in m.tool_calls)),
                None,
            )
            if ai_msg is None:
                return {}
            pix_call = next(tc for tc in ai_msg.tool_calls if tc["name"] == "create_pix")

            run_as_token = config["configurable"]["run_as_token"]
            pix_result, _ = await MCPClient(settings.mcp_server_url, run_as_token).call_tool(
                "create_pix", {**pix_call["args"], "pin_session_token": token.strip()}
            )

            return {"messages": [ToolMessage(content=pix_result, tool_call_id=tm.tool_call_id, id=tm.id)]}

    builder = StateGraph(AgentState)
    builder.add_node("llm", llm_node)
    builder.add_node("tools", ToolNode(TOOLS))
    builder.add_node("confirm", confirm_node)
    builder.add_node("step_up", step_up_node)
    builder.set_entry_point("llm")
    builder.add_conditional_edges("llm", _route_llm, {"confirm": "confirm", "tools": "tools", END: END})
    builder.add_conditional_edges("confirm", _route_confirm)
    builder.add_conditional_edges("tools", _route_tools, {"step_up": "step_up", "llm": "llm"})
    builder.add_edge("step_up", "llm")
    return builder.compile(checkpointer=checkpointer)

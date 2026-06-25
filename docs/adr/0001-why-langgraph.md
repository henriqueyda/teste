# ADR 0001 — LangGraph for orchestration, LangChain only for RAG

## Status
Accepted.

## Context
The mandatory critical-operation flows (limit-increase confirmation, PIX step-up) require
pausing a workflow and resuming it on a *later* HTTP request. We need conversation memory
across turns and an inspectable, auditable decision path.

## Decision
Use **LangGraph** for the orchestration graph and **LangChain** only for RAG plumbing
(loaders, splitters, retriever). Use plain LCEL for the linear RAG path.

## Rationale
LangGraph's `interrupt()` + Postgres checkpointer give human-in-the-loop pause/resume and
durable per-thread memory as first-class primitives. Doing this with LangChain's
`AgentExecutor` would mean hand-rolling state persistence and resumption — i.e.
reimplementing LangGraph, worse and harder to audit. We considered (a) AgentExecutor and
(b) rolling our own; both rejected for a weekend budget.

## Consequences
One framework dependency; the graph is explicit and testable. Security decisions stay in
the Go tier regardless — LangGraph orchestrates, it does not authorize.

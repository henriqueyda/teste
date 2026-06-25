# ADR 0002 — FAISS for the RAG vector index

## Status
Accepted.

## Context
RAG needs vector storage/search over a small, read-mostly knowledge base of public policy
documents. The challenge suggests three options:

- **FAISS** — an in-process *library* (not a server). Lowest ops; the index is a file.
- **Chroma** — an embedded *vector DB* with native metadata filtering; the lightweight
  middle ground, more "database-like" than FAISS with near-zero ops.
- **OpenSearch** — a distributed *search engine*; its key draw is **hybrid keyword+vector
  search** (valuable for exact banking terms like "PIX"/"consignado"), plus multi-node
  scale and built-in auth/RBAC. Heaviest to run.

## Decision
Use **FAISS** as an on-disk index owned by the Python (agent) tier, built offline by
`scripts/ingest_kb.py` and loaded by `app/rag/retriever.py`.

## Rationale
- Matches the suggested stack.
- **Sharpens the trust boundary:** the KB index is a local file in the untrusted Python tier
  and contains only public policy text. The banking Postgres (system-of-record + audit) stays
  Go-only and holds no RAG data.
- Zero extra infrastructure; ideal for a small, read-mostly KB.

## Metadata filtering (deferred)
Not needed here. The KB is **public** policy text, and authorization in this system applies
to the banking **tools/data**, not to the documents — so there is nothing to filter by
audience. We attach **source metadata only for citation** (required by scenario A), not for
predicate filtering. If the KB ever held role/segment-restricted documents, document-level
filtering would become a **RAG guardrail** (don't retrieve what the caller can't see) and we
would move to a store with first-class filtering (Chroma or OpenSearch).

## Consequences
FAISS is an in-process library, not a datastore: no transactions, no concurrent multi-writer
access, persistence is a saved index file loaded at startup. Acceptable here. If the KB grew
large, needed hybrid search, live multi-writer updates, or document-level access control, we'd
migrate to OpenSearch (hybrid/scale/auth) or Chroma (lighter, native filtering) — isolated
behind `app/rag/retriever.py`. (Loading a saved LangChain FAISS index requires
`allow_dangerous_deserialization=True`; safe because we build and own the file.)

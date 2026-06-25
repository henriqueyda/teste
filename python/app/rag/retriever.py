"""FAISS retriever for the banking knowledge base.

Loaded once at startup (init_retriever). The search_knowledge_base tool in
graph/build.py calls search() on the singleton. Returns top-K chunks with source
citation, or a "not found" signal when the best match is below the similarity
threshold — so the agent refuses instead of hallucinating.

The index holds only public policy text; no authz is applied (see ADR 0002).
"""
from __future__ import annotations

from pathlib import Path

from langchain_community.vectorstores import FAISS
from langchain_google_genai import GoogleGenerativeAIEmbeddings

# L2 distance above this → not relevant enough to cite.
# text-embedding-004 outputs L2-normalised vectors, so L2 ∈ [0, 2].
# L2 < 1.0 ≈ cosine > 0.5; anything above is likely off-topic.
_THRESHOLD = 0.7
_NOT_FOUND = (
    "Informação não encontrada na base de conhecimento interna. "
    "Não posso responder sobre esse assunto."
)


class KBRetriever:
    def __init__(self, index_path: str, api_key: str) -> None:
        embeddings = GoogleGenerativeAIEmbeddings(
            model="models/gemini-embedding-001",
            google_api_key=api_key,
        )
        self._db = FAISS.load_local(
            index_path,
            embeddings,
            allow_dangerous_deserialization=True,
        )

    def search(self, query: str, k: int = 3) -> str:
        results = self._db.similarity_search_with_score(query, k=k)
        if not results or results[0][1] > _THRESHOLD:
            return _NOT_FOUND
        parts = []
        for doc, _score in results:
            source = doc.metadata.get("source", "política interna")
            parts.append(f"[Fonte: {source}]\n{doc.page_content}")
        return "\n\n---\n\n".join(parts)


_retriever: KBRetriever | None = None


def init_retriever(index_path: str, api_key: str) -> None:
    global _retriever
    if not Path(index_path).exists():
        print(f"[rag] index not found at {index_path} — search_knowledge_base unavailable. Run `make ingest`.")
        return
    _retriever = KBRetriever(index_path, api_key)
    print(f"[rag] loaded FAISS index from {index_path}")


def get_retriever() -> KBRetriever | None:
    return _retriever

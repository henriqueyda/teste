"""One-shot KB ingestion: load kb/policies/*.pdf -> chunk -> embed -> save FAISS index.

Run via: make ingest   (which runs: cd python && python -m scripts.ingest_kb)

PDFs are generated from kb/sources/*.md by `make kb` (see scripts/build_kb_pdfs.py).
The FAISS index is saved to disk and loaded at agent startup by app/rag/retriever.py.
"""
from __future__ import annotations

import sys
from pathlib import Path

# Allow `python -m scripts.ingest_kb` from the python/ dir
sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

from langchain_community.document_loaders import PyPDFLoader
from langchain_community.vectorstores import FAISS
from langchain_google_genai import GoogleGenerativeAIEmbeddings
from langchain_text_splitters import RecursiveCharacterTextSplitter

from app.config import settings

ROOT = Path(__file__).resolve().parents[2]
KB_DIR = ROOT / "kb" / "policies"
INDEX_PATH = ROOT / "python" / ".faiss_index"


def main() -> None:
    pdfs = sorted(KB_DIR.glob("*.pdf"))
    if not pdfs:
        raise SystemExit(f"No PDFs in {KB_DIR}. Run `make kb` first.")

    docs = []
    for pdf in pdfs:
        pages = PyPDFLoader(str(pdf)).load()
        for page in pages:
            page.metadata["source"] = pdf.stem
        docs.extend(pages)
        print(f"  loaded {pdf.name} ({len(pages)} page(s))")

    chunks = RecursiveCharacterTextSplitter(
        chunk_size=800, chunk_overlap=100
    ).split_documents(docs)
    print(f"Split into {len(chunks)} chunks.")

    embeddings = GoogleGenerativeAIEmbeddings(
        model="models/gemini-embedding-001",
        google_api_key=settings.google_api_key,
    )

    print("Embedding and building FAISS index ...")
    db = FAISS.from_documents(chunks, embeddings)

    INDEX_PATH.mkdir(parents=True, exist_ok=True)
    db.save_local(str(INDEX_PATH))
    print(f"Saved index to {INDEX_PATH}")


if __name__ == "__main__":
    main()

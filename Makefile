.DEFAULT_GOAL := help
SHELL := /bin/bash

POSTGRES_USER ?= bank
POSTGRES_DB   ?= bank

.PHONY: help up down reset logs psql kb ingest demo test verify-audit gateway mcp agent hash

up: ## Start infra + agent (Postgres, Jaeger, Python agent) — no key setup needed
	docker compose up -d
	@echo "Postgres :5432 | Jaeger UI http://localhost:16686 | Agent http://localhost:8000"

down: ## Stop all services
	docker compose down

reset: ## Wipe DB volume and re-seed from scratch
	docker compose down -v && docker compose up -d

logs: ## Tail service logs
	docker compose logs -f

gateway: ## Run the gateway on the host (needs `make up`)
	cd go && go run ./cmd/gateway

mcp: ## Run the MCP server on the host (needs `make up`)
	cd go && go run ./cmd/mcp-server

agent: ## Build and restart the Python agent container
	cd python && .venv/bin/uvicorn app.main:app --reload

kb: ## Build KB PDFs from kb/sources/*.md into kb/policies/
	cd python && python3 scripts/build_kb_pdfs.py

ingest: ## (M5) Ingest kb/policies/*.pdf into the FAISS index
	cd python && python3 -m scripts.ingest_kb

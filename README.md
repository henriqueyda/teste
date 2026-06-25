# Agente de Atendimento Bancário Inteligente

Um agente de IA conversacional para autoatendimento bancário.

---

## Como Executar

### Pré-requisitos

- Docker e Docker Compose
- Go 1.22+
- Python 3.12+
- Chave de API do Google AI Studio (`GOOGLE_API_KEY`)

### 1. Configurar variáveis de ambiente

```bash
# Edite o .env existente e preencha GOOGLE_API_KEY
```

### 2. Subir a infraestrutura

```bash
make up       # inicia PostgreSQL (:5432) e Jaeger (:16686)
make reset    # alternativa: derruba, limpa o volume e reinicia do zero
```

### 3. Construir a base de conhecimento

```bash
make kb       # converte kb/sources/*.md em PDFs em kb/policies/
make ingest   # embute os PDFs no índice FAISS (.faiss_index)
```

> Necessário apenas na primeira execução ou ao alterar as políticas.

### 4. Iniciar os serviços Go (dois terminais separados)

```bash
make gateway  # Gateway em :8080
make mcp      # MCP Server (Security Kernel) em :7070
```

### 5. Iniciar o agente Python

```bash
cd python && python -m venv .venv && .venv/bin/pip install -e .
make agent    # uvicorn em :8000 com --reload
```

### 6. Usar a API

**Login:**
```bash
curl -s -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"username":"maria","password":"demo123"}' | jq .
```

**Chat:**
```bash
curl -s -X POST http://localhost:8080/chat \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"message":"Qual é o meu saldo?"}' | jq .
```

**Usuários de demonstração:**

| Usuário | Senha   | PIN  | Papel    |
|---------|---------|------|----------|
| maria   | demo123 | 1234 | customer |
| ana     | demo123 | —    | manager  |


## Estrutura do Projeto

```
.
├── go/                         # Tier Go — gateway e kernel de segurança
│   ├── cmd/
│   │   ├── gateway/            # API HTTP pública (:8080)
│   │   ├── mcp-server/         # MCP Server (:7070)
│   │   └── verify-audit/       # CLI de verificação do audit log
│   └── internal/
│       ├── gateway/            # Handlers HTTP, auth middleware, /step-up
│       ├── mcp/                # Ferramentas: balance, profile, card, pix
│       ├── auth/               # JWT sign/verify, bcrypt
│       ├── authz/              # RBAC + ownership check
│       ├── audit/              # Log com hash encadeado
│       ├── store/              # Queries SQL (pgx)
│       ├── domain/             # Tipos compartilhados
│       └── telemetry/          # OTel (TracerProvider, OTLP HTTP)
│
├── python/                     # Tier Python — agente LangGraph
│   └── app/
│       ├── main.py             # FastAPI, endpoint /invoke
│       ├── graph/
│       │   ├── build.py        # Grafo: llm_node, confirm_node, step_up_node
│       │   └── state.py        # AgentState
│       ├── mcp/client.py       # Cliente MCP com injeção de traceparent
│       ├── rag/retriever.py    # FAISS + limiar de similaridade L2
│       ├── guardrails/
│       │   ├── input.py        # Detecção de prompt injection (PT-BR + EN)
│       │   └── output.py       # Redação de PII (CPF, cartões)
│       └── telemetry.py        # OTel Python (tracer + propagador W3C)
│
├── db/
│   ├── 01_schema.sql           # users, accounts, cards, audit_log, pin_sessions
│   ├── 02_seed.sql             # Dados de demo (maria, ana, contas, cartões)
│   ├── 03_audit_lockdown.sql   # app_user: somente INSERT em audit_log
│   └── 04_checkpoint.sql       # Tabelas do LangGraph checkpointer
│
├── kb/
│   ├── sources/                # Políticas em Markdown (pix, card-limit, loan-rates)
│   └── policies/               # PDFs gerados por `make kb`
│
├── scripts/demo.sh             # Script dos cenários A → D
├── docker-compose.yml          # PostgreSQL + Jaeger
├── Makefile                    # Ponto de entrada de todos os comandos
└── architecture.excalidraw     # Diagrama de arquitetura
```

---

## Decisões Arquiteturais e Trade-offs

### 1. O LLM não é a fronteira de segurança

A premissa central do projeto: decisões de segurança (autenticação, autorização, validação de PIN) são sempre tomadas no backend Go, nunca delegadas ao modelo.

O agente Python recebe um `run-as token` opaco via header `X-Run-As-Token` — fora do contexto do LLM. O MCP Server revalida esse token em *cada* chamada de ferramenta, independentemente do que o modelo propuser.

---

### 2. MCP como protocolo entre agente e backend

O Model Context Protocol cria uma fronteira explícita entre o que o LLM pode *propor* e o que o backend pode *executar*. O allowlist de ferramentas está hardcoded no servidor Go — o agente não consegue invocar funções arbitrárias.

**Trade-off:** overhead de rede versus chamada direta. A vantagem é o desacoplamento total: o backend pode ser auditado, substituído ou escalonado sem tocar no código do agente.

---

### 3. LangGraph com interrupt para operações sensíveis

Operações de escrita interrompem o grafo (`interrupt()`) e aguardam confirmação explícita antes de executar. O PIX requer um segundo fator: o gateway Go verifica o PIN com bcrypt e emite um token de sessão de uso único (TTL de 60s) — que nunca passa pelo contexto do LLM.

**Trade-off:** o checkpoint do LangGraph no PostgreSQL persiste estado entre requisições, adicionando dependência de banco no tier Python. Simplifica a lógica de sessão ao custo de mais acoplamento à infraestrutura.

---

### 4. RBAC + verificação de ownership no kernel MCP

Cada chamada passa pelo pipeline: `JWT verify → RBAC → ownership check → execute → audit`. O ownership check garante que um `customer` só acessa seus próprios recursos — mesmo que o LLM tente passar um `card_id` de outro cliente.

**Trade-off:** a verificação de ownership exige uma leitura prévia do banco antes da transação principal. Aceitável dado o ganho de segurança e a natureza de autoatendimento (baixo volume concorrente).

---

### 5. FAISS local em vez de banco vetorial externo

O índice de embeddings fica em disco (`.faiss_index`) no tier Python, sem dependência de serviços externos como Pinecone ou pgvector.

**Trade-off:** sem suporte a atualizações incrementais — qualquer mudança nas políticas requer `make ingest` completo. Adequado para a escala do desafio; em produção seria substituído por um banco vetorial gerenciado com indexação incremental.

---

## Limitações e Próximos Passos

### Limitações atuais

- **Sem frontend:** a integração com o fluxo de PIN (`/step-up`) requer que o cliente detecte `awaiting_pin: true` e apresente uma tela separada. Hoje só é testável via `curl` ou o script de demo.
- **FAISS sem hot-reload:** atualizar as políticas exige reiniciar o agente após `make ingest`.
- **Uma sessão MCP por requisição:** o cliente abre e fecha uma conexão HTTP para cada tool call. Em produção seria necessário pooling de conexões.
- **Histórico de conversa sem truncamento:** o LangGraph envia todas as mensagens ao LLM. Em conversas longas, custo e latência crescem linearmente.

### Próximos passos

- **Frontend web** com estados `awaiting_confirmation` e `awaiting_pin` mapeados para componentes visuais dedicados.
- **Testes de integração** cobrindo os cenários de segurança (A–D) de forma automatizada.
- **Truncamento de janela de contexto** para controlar custo de tokens em conversas longas.
- **Refresh de JWT** para evitar relogin após expiração sem comprometer a sessão do usuário.
- **Pool de conexões MCP** para reutilizar sessões HTTP entre tool calls da mesma requisição.

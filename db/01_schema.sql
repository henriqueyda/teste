-- Banking Agent schema. Auto-applied on first DB boot (docker-entrypoint-initdb.d).
-- Trust model: this DB is reachable ONLY from the Go tier. The Python agent has no creds.
-- RAG vectors do NOT live here: the KB index is FAISS, owned by the Python tier (see ADR 0002).
-- This keeps the banking system-of-record free of KB data and sharpens the trust boundary.

-- ============================ IDENTITY ============================
CREATE TABLE customers (
    id          TEXT PRIMARY KEY,                 -- 'cust_maria'
    full_name   TEXT NOT NULL,
    document    TEXT NOT NULL UNIQUE,             -- CPF
    is_retiree  BOOLEAN NOT NULL DEFAULT FALSE,   -- used by loan-rate policy / RAG
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE users (
    id                   TEXT PRIMARY KEY,        -- 'usr_maria' == JWT sub
    username             TEXT NOT NULL UNIQUE,
    password_hash        TEXT NOT NULL,           -- bcrypt; NEVER plaintext
    transaction_pin_hash TEXT,                    -- bcrypt; NULL for staff who cannot transact
    customer_id          TEXT REFERENCES customers(id),  -- NULL for staff (teller/admin)
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================== RBAC ==============================
CREATE TABLE roles (
    id             TEXT PRIMARY KEY,              -- 'customer' | 'teller' | 'admin'
    description    TEXT NOT NULL,
    any_customer BOOLEAN NOT NULL DEFAULT FALSE -- may act on OTHER customers' resources?
);

CREATE TABLE permissions (
    id          TEXT PRIMARY KEY,                 -- 'card_limit.update', 'pix.create', ...
    description TEXT NOT NULL
);

CREATE TABLE role_permissions (
    role_id       TEXT NOT NULL REFERENCES roles(id),
    permission_id TEXT NOT NULL REFERENCES permissions(id),
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE user_roles (
    user_id TEXT NOT NULL REFERENCES users(id),
    role_id TEXT NOT NULL REFERENCES roles(id),
    PRIMARY KEY (user_id, role_id)
);

-- ====================== BANKING RESOURCES =========================
-- customer_id is the OWNER column ownership checks compare against.
CREATE TABLE accounts (
    id                    TEXT PRIMARY KEY,
    customer_id           TEXT NOT NULL REFERENCES customers(id),
    balance_cents         BIGINT NOT NULL DEFAULT 0,
    pix_daily_limit_cents BIGINT NOT NULL DEFAULT 500000
);

CREATE TABLE cards (
    id              TEXT PRIMARY KEY,
    customer_id     TEXT NOT NULL REFERENCES customers(id),
    limit_cents     BIGINT NOT NULL,
    max_limit_cents BIGINT NOT NULL                -- eligibility ceiling for increases
);

-- RAG (KB) vectors are intentionally NOT stored here — see FAISS in the Python tier.

-- ==================== STEP-UP AUTH (PIX PIN sessions) ====================
-- Short-lived one-use tokens issued by /step-up after bcrypt PIN verification.
-- The LangGraph checkpoint stores the token, never the raw PIN.
CREATE TABLE pin_sessions (
    id         TEXT PRIMARY KEY,           -- opaque random hex token
    user_id    TEXT NOT NULL REFERENCES users(id),
    expires_at TIMESTAMPTZ NOT NULL,       -- issued with 60s TTL
    used_at    TIMESTAMPTZ                 -- NULL until consumed; enforces one-use
);

-- ===================== AUDIT (append-only) ========================
-- Written by the MCP server IN THE SAME TX as the action.
CREATE TABLE audit_log (
    seq            BIGSERIAL PRIMARY KEY,
    correlation_id TEXT NOT NULL,     -- == OTel trace id
    user_id        TEXT,
    action         TEXT NOT NULL,
    tool           TEXT,
    arguments      JSONB,
    decision       TEXT NOT NULL,     -- 'allow' | 'deny'
    reason         TEXT,              -- 'ownership_violation', 'step_up_required', ...
    result         JSONB,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

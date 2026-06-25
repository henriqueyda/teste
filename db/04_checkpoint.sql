-- agent_user: the Python agent's DB identity. Access is strictly limited to the
-- LangGraph checkpoint tables — no access to banking tables (users, accounts, cards, etc.).
-- LangGraph's AsyncPostgresSaver.setup() creates the checkpoint tables on first boot.

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'agent_user') THEN
    CREATE ROLE agent_user LOGIN PASSWORD 'agent_user';
  END IF;
END $$;

GRANT CONNECT ON DATABASE bank TO agent_user;
GRANT USAGE ON SCHEMA public TO agent_user;
-- CREATE is required so setup() can create checkpoint tables on first boot.
-- In production: run setup() once as a migration step, then revoke CREATE.
GRANT CREATE ON SCHEMA public TO agent_user;

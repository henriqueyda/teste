-- Make audit_log append-only at the DB level: the application role may INSERT and SELECT,
-- but never UPDATE or DELETE.
--
-- In M1 the app will connect as a least-privilege role 'app_user' (created here).
-- For now (infra-only boot) we just create the role and lock down audit_log.

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_user') THEN
    CREATE ROLE app_user LOGIN PASSWORD 'app_user';
  END IF;
END $$;

GRANT USAGE ON SCHEMA public TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO app_user;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO app_user;

-- audit_log is append-only for the app role.
REVOKE UPDATE, DELETE ON audit_log FROM app_user;

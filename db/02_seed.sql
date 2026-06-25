-- Seed data that makes the four scenarios concrete.
-- password_hash values are bcrypt hashes of the dev passwords below.
-- Regenerate with: make hash PW=demo123  (then paste the output here and `make reset`).

-- roles: customers act only on themselves; staff may cross customers
INSERT INTO roles (id, description, any_customer) VALUES
  ('customer', 'Bank client',      FALSE),
  ('manager',  'Branch manager',   TRUE),
  ('admin',    'Administrator',    TRUE);

INSERT INTO permissions (id, description) VALUES
  ('customer.read',    'Read customer profile'),
  ('account.read',     'Read account balance'),
  ('card_limit.read',  'Read card limit'),
  ('card_limit.update','Increase/decrease card limit'),
  ('pix.create',       'Create a PIX transfer');

-- customer: full self-service. manager: read across customers, but cannot move money. admin: all.
INSERT INTO role_permissions (role_id, permission_id) VALUES
  ('customer','customer.read'),('customer','account.read'),('customer','card_limit.read'),
  ('customer','card_limit.update'),('customer','pix.create'),
  ('manager','customer.read'),('manager','account.read'),('manager','card_limit.read'),
  ('admin','customer.read'),('admin','account.read'),('admin','card_limit.read'),
  ('admin','card_limit.update'),('admin','pix.create');

-- people
INSERT INTO customers (id, full_name, document, is_retiree) VALUES
  ('cust_maria', 'Maria Souza', '111.111.111-11', TRUE),
  ('cust_joao',  'João Silva',  '222.222.222-22', FALSE);  -- scenario C target

-- Maria = logged-in customer. Ana = manager (staff, no customer link). João has no login.
-- maria/demo123 PIN:1234, ana/demo123 (no PIN — staff cannot create PIX)
INSERT INTO users (id, username, password_hash, transaction_pin_hash, customer_id) VALUES
  ('usr_maria', 'maria', '$2a$10$UJ58qORfmEqgkNpcVp0SouVJDPHCxmxIIHUBG9qfHanL/onH.Nwae', '$2a$10$wcn/BD1ftY6dtG/LdoayreNrF6pJL.UtRXJYCJTk8EdCg2lJrmzny', 'cust_maria'),
  ('usr_ana',   'ana',   '$2a$10$UJ58qORfmEqgkNpcVp0SouVJDPHCxmxIIHUBG9qfHanL/onH.Nwae', NULL, NULL);
INSERT INTO user_roles (user_id, role_id) VALUES
  ('usr_maria','customer'), ('usr_ana','manager');

-- resources (note the OWNER column = customer_id)
INSERT INTO accounts (id, customer_id, balance_cents, pix_daily_limit_cents) VALUES
  ('acc_maria','cust_maria', 1000000, 5000000),   -- Maria, R$10k balance, R$50k daily PIX limit
  ('acc_joao', 'cust_joao',  9999999,  500000);   -- João (scenario C target)
INSERT INTO cards (id, customer_id, limit_cents, max_limit_cents) VALUES
  ('card_maria','cust_maria', 800000, 2000000),   -- Maria's card: R$8k now, up to R$20k eligible
  ('card_joao', 'cust_joao',  500000, 1000000);   -- Scenario C target: Maria must NOT touch this

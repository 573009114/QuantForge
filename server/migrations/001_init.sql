CREATE TABLE IF NOT EXISTS strategy (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  name TEXT NOT NULL,
  description TEXT NOT NULL,
  version TEXT NOT NULL,
  status TEXT NOT NULL,
  parameters JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_strategy_tenant ON strategy(tenant_id);

CREATE TABLE IF NOT EXISTS sim_account (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  balance DOUBLE PRECISION NOT NULL,
  equity DOUBLE PRECISION NOT NULL,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sim_account_tenant ON sim_account(tenant_id);

CREATE TABLE IF NOT EXISTS sim_order (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  account_id TEXT NOT NULL,
  symbol TEXT NOT NULL,
  side TEXT NOT NULL,
  qty DOUBLE PRECISION NOT NULL,
  price DOUBLE PRECISION NOT NULL,
  status TEXT NOT NULL,
  submitted_at TIMESTAMP NULL,
  filled_at TIMESTAMP NULL,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sim_order_tenant ON sim_order(tenant_id);

CREATE TABLE IF NOT EXISTS risk_rule (
  tenant_id TEXT PRIMARY KEY,
  max_order_notional DOUBLE PRECISION NOT NULL,
  max_daily_loss DOUBLE PRECISION NOT NULL,
  max_position_percent DOUBLE PRECISION NOT NULL,
  updated_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS audit_log (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  user_id TEXT NOT NULL,
  action TEXT NOT NULL,
  resource TEXT NOT NULL,
  detail TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_audit_tenant_created ON audit_log(tenant_id, created_at DESC);

CREATE TABLE IF NOT EXISTS dead_letter_job (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  job_type TEXT NOT NULL,
  job_id TEXT NOT NULL,
  reason TEXT NOT NULL,
  payload TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_dead_letter_tenant_created ON dead_letter_job(tenant_id, created_at DESC);

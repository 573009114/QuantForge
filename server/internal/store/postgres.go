package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"quantforge/server/internal/model"
)

type PostgresStore struct{ db *sql.DB }

func NewPostgresStoreFromEnv() (*PostgresStore, error) {
	dsn := os.Getenv("QF_PG_DSN")
	if dsn == "" {
		return nil, errors.New("QF_PG_DSN is empty")
	}
	driver := os.Getenv("QF_PG_DRIVER")
	if driver == "" {
		driver = "postgres"
	}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		return nil, err
	}
	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

func (s *PostgresStore) ApplyMigrationsFromDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	files := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		files = append(files, filepath.Join(dir, e.Name()))
	}
	sort.Strings(files)
	for _, file := range files {
		buf, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		parts := strings.Split(string(buf), ";")
		for _, stmt := range parts {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err = s.db.Exec(stmt); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *PostgresStore) SaveStrategy(item model.Strategy) error {
	params, _ := json.Marshal(item.Parameters)
	_, err := s.db.Exec(`INSERT INTO strategy (id,tenant_id,name,description,version,status,parameters,created_at,updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
ON CONFLICT (id) DO UPDATE SET description=EXCLUDED.description,version=EXCLUDED.version,status=EXCLUDED.status,parameters=EXCLUDED.parameters,updated_at=EXCLUDED.updated_at`,
		item.ID, item.TenantID, item.Name, item.Description, item.Version, string(item.Status), string(params), item.CreatedAt, item.UpdatedAt)
	return err
}

func (s *PostgresStore) LoadStrategies(tenantID string) ([]model.Strategy, error) {
	rows, err := s.db.Query(`SELECT id,tenant_id,name,description,version,status,parameters,created_at,updated_at FROM strategy WHERE tenant_id=$1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Strategy{}
	for rows.Next() {
		var it model.Strategy
		var status string
		var params []byte
		if err = rows.Scan(&it.ID, &it.TenantID, &it.Name, &it.Description, &it.Version, &status, &params, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, err
		}
		it.Status = model.StrategyStatus(status)
		_ = json.Unmarshal(params, &it.Parameters)
		out = append(out, it)
	}
	return out, nil
}

func (s *PostgresStore) DeleteStrategy(tenantID, id string) error {
	_, err := s.db.Exec(`DELETE FROM strategy WHERE tenant_id=$1 AND id=$2`, tenantID, id)
	return err
}

func (s *PostgresStore) SaveAccount(it model.SimAccount) error {
	_, err := s.db.Exec(`INSERT INTO sim_account (id,tenant_id,balance,equity,created_at,updated_at)
VALUES ($1,$2,$3,$4,$5,$6)
ON CONFLICT (id) DO UPDATE SET balance=EXCLUDED.balance,equity=EXCLUDED.equity,updated_at=EXCLUDED.updated_at`, it.ID, it.TenantID, it.Balance, it.Equity, it.CreatedAt, it.UpdatedAt)
	return err
}

func (s *PostgresStore) LoadAccounts(tenantID string) ([]model.SimAccount, error) {
	rows, err := s.db.Query(`SELECT id,tenant_id,balance,equity,created_at,updated_at FROM sim_account WHERE tenant_id=$1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.SimAccount{}
	for rows.Next() {
		var it model.SimAccount
		if err = rows.Scan(&it.ID, &it.TenantID, &it.Balance, &it.Equity, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, nil
}

func (s *PostgresStore) SaveOrder(it model.SimOrder) error {
	_, err := s.db.Exec(`INSERT INTO sim_order (id,tenant_id,account_id,symbol,side,qty,price,status,submitted_at,filled_at,created_at,updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
ON CONFLICT (id) DO UPDATE SET status=EXCLUDED.status,submitted_at=EXCLUDED.submitted_at,filled_at=EXCLUDED.filled_at,updated_at=EXCLUDED.updated_at`,
		it.ID, it.TenantID, it.AccountID, it.Symbol, it.Side, it.Qty, it.Price, string(it.Status), it.SubmittedAt, it.FilledAt, it.CreatedAt, it.UpdatedAt)
	return err
}

func (s *PostgresStore) LoadOrders(tenantID string) ([]model.SimOrder, error) {
	rows, err := s.db.Query(`SELECT id,tenant_id,account_id,symbol,side,qty,price,status,submitted_at,filled_at,created_at,updated_at FROM sim_order WHERE tenant_id=$1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.SimOrder{}
	for rows.Next() {
		var it model.SimOrder
		var status string
		if err = rows.Scan(&it.ID, &it.TenantID, &it.AccountID, &it.Symbol, &it.Side, &it.Qty, &it.Price, &status, &it.SubmittedAt, &it.FilledAt, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, err
		}
		it.Status = model.OrderStatus(status)
		out = append(out, it)
	}
	return out, nil
}

func (s *PostgresStore) SaveRiskRule(it model.RiskRule) error {
	_, err := s.db.Exec(`INSERT INTO risk_rule (tenant_id,max_order_notional,max_daily_loss,max_position_percent,updated_at)
VALUES ($1,$2,$3,$4,$5)
ON CONFLICT (tenant_id) DO UPDATE SET max_order_notional=EXCLUDED.max_order_notional,max_daily_loss=EXCLUDED.max_daily_loss,max_position_percent=EXCLUDED.max_position_percent,updated_at=EXCLUDED.updated_at`, it.TenantID, it.MaxOrderNotional, it.MaxDailyLoss, it.MaxPositionPercent, it.UpdatedAt)
	return err
}

func (s *PostgresStore) LoadRiskRule(tenantID string) (model.RiskRule, bool, error) {
	var it model.RiskRule
	err := s.db.QueryRow(`SELECT tenant_id,max_order_notional,max_daily_loss,max_position_percent,updated_at FROM risk_rule WHERE tenant_id=$1`, tenantID).
		Scan(&it.TenantID, &it.MaxOrderNotional, &it.MaxDailyLoss, &it.MaxPositionPercent, &it.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return model.RiskRule{}, false, nil
	}
	if err != nil {
		return model.RiskRule{}, false, err
	}
	return it, true, nil
}

func (s *PostgresStore) AppendAudit(it model.AuditLog) error {
	_, err := s.db.Exec(`INSERT INTO audit_log (id,tenant_id,user_id,action,resource,detail,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`, it.ID, it.TenantID, it.UserID, it.Action, it.Resource, it.Detail, it.CreatedAt)
	return err
}

func (s *PostgresStore) LoadAudits(tenantID string, limit int) ([]model.AuditLog, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.db.Query(`SELECT id,tenant_id,user_id,action,resource,detail,created_at FROM audit_log WHERE tenant_id=$1 ORDER BY created_at DESC LIMIT $2`, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.AuditLog{}
	for rows.Next() {
		var it model.AuditLog
		if err = rows.Scan(&it.ID, &it.TenantID, &it.UserID, &it.Action, &it.Resource, &it.Detail, &it.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, nil
}

func (s *PostgresStore) AppendDeadLetter(it model.DeadLetterJob) error {
	_, err := s.db.Exec(`INSERT INTO dead_letter_job (id,tenant_id,job_type,job_id,reason,payload,created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7)`, it.ID, it.TenantID, it.JobType, it.JobID, it.Reason, it.Payload, it.CreatedAt)
	return err
}

func (s *PostgresStore) LoadDeadLetters(tenantID string, limit int) ([]model.DeadLetterJob, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.Query(`SELECT id,tenant_id,job_type,job_id,reason,payload,created_at FROM dead_letter_job WHERE tenant_id=$1 ORDER BY created_at DESC LIMIT $2`, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.DeadLetterJob{}
	for rows.Next() {
		var it model.DeadLetterJob
		if err = rows.Scan(&it.ID, &it.TenantID, &it.JobType, &it.JobID, &it.Reason, &it.Payload, &it.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, nil
}

func EnvAuditLimit() int {
	n, _ := strconv.Atoi(os.Getenv("QF_AUDIT_LIMIT"))
	if n <= 0 {
		return 200
	}
	return n
}

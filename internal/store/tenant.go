package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/fengheasia/frp-warden/internal/model"
)

// tenantColumns 是读取 tenant 时统一使用的列。
const tenantColumns = "id, code, name, token_hash, status, description"

// TenantInput 是创建租户的输入。token 明文不在此处保存,只接收其 hash。
type TenantInput struct {
	Code        string
	Name        string
	TokenHash   string
	Description string
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTenant(s rowScanner) (model.Tenant, error) {
	var t model.Tenant
	err := s.Scan(&t.ID, &t.Code, &t.Name, &t.TokenHash, &t.Status, &t.Description)
	return t, err
}

// CreateTenant 创建一个租户。code/name/token_hash 必填;status 默认 enabled。
// 若 code 重复,返回 ErrDuplicateCode。
func (s *Store) CreateTenant(ctx context.Context, in TenantInput) (model.Tenant, error) {
	if strings.TrimSpace(in.Code) == "" || strings.TrimSpace(in.Name) == "" || in.TokenHash == "" {
		return model.Tenant{}, ErrEmptyField
	}

	res, err := s.db.ExecContext(ctx,
		`INSERT INTO tenants (code, name, token_hash, status, description) VALUES (?, ?, ?, ?, ?)`,
		in.Code, in.Name, in.TokenHash, model.StatusEnabled, in.Description)
	if err != nil {
		if isUniqueViolation(err) {
			return model.Tenant{}, ErrDuplicateCode
		}
		return model.Tenant{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return model.Tenant{}, err
	}
	return s.GetTenantByID(ctx, id)
}

// GetTenantByID 按 id 查询租户,不存在返回 ErrTenantNotFound。
func (s *Store) GetTenantByID(ctx context.Context, id int64) (model.Tenant, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+tenantColumns+` FROM tenants WHERE id = ?`, id)
	t, err := scanTenant(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Tenant{}, ErrTenantNotFound
	}
	return t, err
}

// GetTenantByCode 按 code 查询租户,不存在返回 ErrTenantNotFound。
func (s *Store) GetTenantByCode(ctx context.Context, code string) (model.Tenant, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+tenantColumns+` FROM tenants WHERE code = ?`, code)
	t, err := scanTenant(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Tenant{}, ErrTenantNotFound
	}
	return t, err
}

// ListTenants 返回全部租户,按 id 升序。
func (s *Store) ListTenants(ctx context.Context) ([]model.Tenant, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+tenantColumns+` FROM tenants ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.Tenant
	for rows.Next() {
		t, err := scanTenant(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

// UpdateTenantStatus 更新租户状态(enabled/disabled)。租户不存在返回 ErrTenantNotFound。
func (s *Store) UpdateTenantStatus(ctx context.Context, id int64, status string) error {
	return s.updateTenantField(ctx, id, "status", status)
}

// UpdateTenantTokenHash 更新租户的 token_hash(用于重置 token,明文只在上层一次性展示)。
func (s *Store) UpdateTenantTokenHash(ctx context.Context, id int64, tokenHash string) error {
	if tokenHash == "" {
		return ErrEmptyField
	}
	return s.updateTenantField(ctx, id, "token_hash", tokenHash)
}

// UpdateTenantLastSeen 把租户的 last_seen_at 更新为当前时间(供 frps plugin 的 Ping 使用)。
// 租户不存在返回 ErrTenantNotFound。
func (s *Store) UpdateTenantLastSeen(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE tenants SET last_seen_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrTenantNotFound
	}
	return nil
}

func (s *Store) updateTenantField(ctx context.Context, id int64, column, value string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE tenants SET `+column+` = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, value, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrTenantNotFound
	}
	return nil
}

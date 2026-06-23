package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fengheasia/frp-warden/internal/model"
)

const grantColumns = "id, tenant_id, resource_id, status"

func scanGrant(s rowScanner) (model.ResourceGrant, error) {
	var g model.ResourceGrant
	err := s.Scan(&g.ID, &g.TenantID, &g.ResourceID, &g.Status)
	return g, err
}

// GrantResourceToTenant 将一个资源授权给一个租户。
//
// 校验:租户必须存在且 enabled;资源必须存在且 available;该资源尚未被授权给任何租户。
// 核心约束:一个资源只能授权给一个租户——若已被授权,返回 ErrResourceAlreadyGranted
// (数据库对 resource_id 的唯一约束作为最终兜底)。
func (s *Store) GrantResourceToTenant(ctx context.Context, tenantID, resourceID int64) (model.ResourceGrant, error) {
	tenant, err := s.GetTenantByID(ctx, tenantID)
	if err != nil {
		return model.ResourceGrant{}, err // 可能为 ErrTenantNotFound
	}
	if tenant.Status != model.StatusEnabled {
		return model.ResourceGrant{}, ErrTenantDisabled
	}

	resource, err := s.GetResourceByID(ctx, resourceID)
	if err != nil {
		return model.ResourceGrant{}, err // 可能为 ErrResourceNotFound
	}
	if resource.Status != model.ResourceStatusAvailable {
		return model.ResourceGrant{}, ErrResourceDisabled
	}

	// 资源是否已被授权(给任意租户)。
	if _, err := s.GetGrantByResource(ctx, resourceID); err == nil {
		return model.ResourceGrant{}, ErrResourceAlreadyGranted
	} else if !errors.Is(err, ErrGrantNotFound) {
		return model.ResourceGrant{}, err
	}

	res, err := s.db.ExecContext(ctx,
		`INSERT INTO resource_grants (tenant_id, resource_id, status) VALUES (?, ?, ?)`,
		tenantID, resourceID, model.StatusEnabled)
	if err != nil {
		if isUniqueViolation(err) {
			return model.ResourceGrant{}, ErrResourceAlreadyGranted
		}
		return model.ResourceGrant{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return model.ResourceGrant{}, err
	}
	return s.getGrantByID(ctx, id)
}

func (s *Store) getGrantByID(ctx context.Context, id int64) (model.ResourceGrant, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+grantColumns+` FROM resource_grants WHERE id = ?`, id)
	g, err := scanGrant(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.ResourceGrant{}, ErrGrantNotFound
	}
	return g, err
}

// GetGrantByResource 按 resource_id 查询授权,不存在返回 ErrGrantNotFound。
func (s *Store) GetGrantByResource(ctx context.Context, resourceID int64) (model.ResourceGrant, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+grantColumns+` FROM resource_grants WHERE resource_id = ?`, resourceID)
	g, err := scanGrant(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.ResourceGrant{}, ErrGrantNotFound
	}
	return g, err
}

// ListGrantsByTenant 返回某租户的全部授权,按 id 升序。
func (s *Store) ListGrantsByTenant(ctx context.Context, tenantID int64) ([]model.ResourceGrant, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+grantColumns+` FROM resource_grants WHERE tenant_id = ? ORDER BY id`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.ResourceGrant
	for rows.Next() {
		g, err := scanGrant(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, g)
	}
	return list, rows.Err()
}

// UpdateGrantStatus 更新授权状态(enabled/disabled)。不存在返回 ErrGrantNotFound。
func (s *Store) UpdateGrantStatus(ctx context.Context, id int64, status string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE resource_grants SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, status, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrGrantNotFound
	}
	return nil
}

// RevokeGrant 撤销授权(置为 disabled)。
func (s *Store) RevokeGrant(ctx context.Context, id int64) error {
	return s.UpdateGrantStatus(ctx, id, model.StatusDisabled)
}

// UpdateGrantStatusByID 按授权 id 更新状态(enabled/disabled)。不存在返回 ErrGrantNotFound。
func (s *Store) UpdateGrantStatusByID(ctx context.Context, id int64, status string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE resource_grants SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, status, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrGrantNotFound
	}
	return nil
}

// ListGrants 返回全部授权,按 id 升序。
func (s *Store) ListGrants(ctx context.Context) ([]model.ResourceGrant, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+grantColumns+` FROM resource_grants ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []model.ResourceGrant
	for rows.Next() {
		g, err := scanGrant(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, g)
	}
	return list, rows.Err()
}

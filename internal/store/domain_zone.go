package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/fengheasia/frp-warden/internal/model"
)

const domainZoneColumns = "id, name, zone, status"

func scanDomainZone(s rowScanner) (model.DomainZone, error) {
	var z model.DomainZone
	err := s.Scan(&z.ID, &z.Name, &z.Zone, &z.Status)
	return z, err
}

// CreateDomainZone 创建一个顶级域名区域。name/zone 必填;status 默认 enabled。
// zone 重复返回 ErrDuplicateZone。
func (s *Store) CreateDomainZone(ctx context.Context, name, zone string) (model.DomainZone, error) {
	if strings.TrimSpace(name) == "" || strings.TrimSpace(zone) == "" {
		return model.DomainZone{}, ErrEmptyField
	}

	res, err := s.db.ExecContext(ctx,
		`INSERT INTO domain_zones (name, zone, status) VALUES (?, ?, ?)`,
		name, zone, model.StatusEnabled)
	if err != nil {
		if isUniqueViolation(err) {
			return model.DomainZone{}, ErrDuplicateZone
		}
		return model.DomainZone{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return model.DomainZone{}, err
	}
	return s.GetDomainZoneByID(ctx, id)
}

// GetDomainZoneByID 按 id 查询顶级域名区域,不存在返回 ErrDomainZoneNotFound。
func (s *Store) GetDomainZoneByID(ctx context.Context, id int64) (model.DomainZone, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+domainZoneColumns+` FROM domain_zones WHERE id = ?`, id)
	z, err := scanDomainZone(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.DomainZone{}, ErrDomainZoneNotFound
	}
	return z, err
}

// ListDomainZones 返回全部顶级域名区域,按 id 升序。
func (s *Store) ListDomainZones(ctx context.Context) ([]model.DomainZone, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+domainZoneColumns+` FROM domain_zones ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.DomainZone
	for rows.Next() {
		z, err := scanDomainZone(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, z)
	}
	return list, rows.Err()
}

// HasEnabledDomainZone 判断是否存在至少一个 enabled 的顶级域名区域。
func (s *Store) HasEnabledDomainZone(ctx context.Context) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM domain_zones WHERE status = ?`, model.StatusEnabled).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// UpdateDomainZoneStatus 更新顶级域名区域状态。不存在返回 ErrDomainZoneNotFound。
func (s *Store) UpdateDomainZoneStatus(ctx context.Context, id int64, status string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE domain_zones SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, status, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrDomainZoneNotFound
	}
	return nil
}

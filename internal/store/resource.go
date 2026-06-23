package store

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"

	"github.com/fengheasia/frp-warden/internal/model"
)

const resourceColumns = "id, type, value, domain_zone_id, status"

func scanResource(s rowScanner) (model.Resource, error) {
	var (
		r      model.Resource
		zoneID sql.NullInt64
	)
	if err := s.Scan(&r.ID, &r.Type, &r.Value, &zoneID, &r.Status); err != nil {
		return model.Resource{}, err
	}
	if zoneID.Valid {
		v := zoneID.Int64
		r.DomainZoneID = &v
	}
	return r, nil
}

// CreateSubdomainResource 创建一个 subdomain 资源。
//
// 前置条件:系统中必须存在至少一个 enabled 的顶级域名区域,且传入的 domainZoneID
// 必须指向一个 enabled 的区域。value 仅保存子域名前缀(如 ufi001),不含完整域名。
func (s *Store) CreateSubdomainResource(ctx context.Context, value string, domainZoneID int64) (model.Resource, error) {
	if strings.TrimSpace(value) == "" {
		return model.Resource{}, ErrEmptyField
	}

	has, err := s.HasEnabledDomainZone(ctx)
	if err != nil {
		return model.Resource{}, err
	}
	if !has {
		return model.Resource{}, ErrNoEnabledDomainZone
	}

	zone, err := s.GetDomainZoneByID(ctx, domainZoneID)
	if errors.Is(err, ErrDomainZoneNotFound) {
		return model.Resource{}, ErrSubdomainNeedsZone
	}
	if err != nil {
		return model.Resource{}, err
	}
	if zone.Status != model.StatusEnabled {
		return model.Resource{}, ErrDomainZoneDisabled
	}

	id := domainZoneID
	return s.insertResource(ctx, model.ResourceTypeSubdomain, value, &id)
}

// CreateTCPPortResource 创建一个 TCP 公网端口资源。端口须在 1-65535。
func (s *Store) CreateTCPPortResource(ctx context.Context, port int) (model.Resource, error) {
	if !validPort(port) {
		return model.Resource{}, ErrInvalidPort
	}
	return s.insertResource(ctx, model.ResourceTypeTCPPort, strconv.Itoa(port), nil)
}

// CreateUDPPortResource 创建一个 UDP 公网端口资源。端口须在 1-65535。
func (s *Store) CreateUDPPortResource(ctx context.Context, port int) (model.Resource, error) {
	if !validPort(port) {
		return model.Resource{}, ErrInvalidPort
	}
	return s.insertResource(ctx, model.ResourceTypeUDPPort, strconv.Itoa(port), nil)
}

// insertResource 是创建资源的内部实现,status 默认 available;(type,value) 重复返回 ErrDuplicateResource。
func (s *Store) insertResource(ctx context.Context, typ, value string, domainZoneID *int64) (model.Resource, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO resources (type, value, domain_zone_id, status) VALUES (?, ?, ?, ?)`,
		typ, value, domainZoneID, model.ResourceStatusAvailable)
	if err != nil {
		if isUniqueViolation(err) {
			return model.Resource{}, ErrDuplicateResource
		}
		return model.Resource{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return model.Resource{}, err
	}
	return s.GetResourceByID(ctx, id)
}

// GetResourceByID 按 id 查询资源,不存在返回 ErrResourceNotFound。
func (s *Store) GetResourceByID(ctx context.Context, id int64) (model.Resource, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+resourceColumns+` FROM resources WHERE id = ?`, id)
	r, err := scanResource(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Resource{}, ErrResourceNotFound
	}
	return r, err
}

// ListResources 返回全部资源,按 id 升序。
func (s *Store) ListResources(ctx context.Context) ([]model.Resource, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+resourceColumns+` FROM resources ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.Resource
	for rows.Next() {
		r, err := scanResource(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

// UpdateResourceStatus 更新资源状态(available/disabled)。不存在返回 ErrResourceNotFound。
func (s *Store) UpdateResourceStatus(ctx context.Context, id int64, status string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE resources SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, status, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrResourceNotFound
	}
	return nil
}

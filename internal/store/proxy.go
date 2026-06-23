package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/fengheasia/frp-warden/internal/model"
)

const proxyColumns = "id, tenant_id, resource_id, name, proxy_type, local_ip, local_port, status"

func scanProxy(s rowScanner) (model.Proxy, error) {
	var p model.Proxy
	err := s.Scan(&p.ID, &p.TenantID, &p.ResourceID, &p.Name, &p.ProxyType, &p.LocalIP, &p.LocalPort, &p.Status)
	return p, err
}

// ProxyInput 是创建映射配置的输入。
type ProxyInput struct {
	TenantID   int64
	ResourceID int64
	Name       string
	ProxyType  string
	LocalIP    string
	LocalPort  int
}

// validProxyType 判断 proxy 类型是否合法。
func validProxyType(t string) bool {
	switch t {
	case model.ProxyTypeHTTP, model.ProxyTypeHTTPS, model.ProxyTypeTCP, model.ProxyTypeUDP:
		return true
	default:
		return false
	}
}

// proxyTypeMatchesResource 校验 proxy 类型与资源类型是否匹配:
// http/https 必须用 subdomain 资源;tcp 必须用 tcp_port;udp 必须用 udp_port。
func proxyTypeMatchesResource(proxyType, resourceType string) bool {
	switch proxyType {
	case model.ProxyTypeHTTP, model.ProxyTypeHTTPS:
		return resourceType == model.ResourceTypeSubdomain
	case model.ProxyTypeTCP:
		return resourceType == model.ResourceTypeTCPPort
	case model.ProxyTypeUDP:
		return resourceType == model.ResourceTypeUDPPort
	default:
		return false
	}
}

// CreateProxy 创建一个映射配置。
//
// 校验:name 必填、proxy_type 合法、local_port 在 1-65535;资源必须存在;该资源必须
// 已 enabled 授权给本租户(否则 ErrResourceNotGrantedToTenant);proxy_type 必须与资源
// 类型匹配(否则 ErrProxyTypeResourceMismatch);同一租户下 name 不可重复。
// local_ip 为空时默认 127.0.0.1。
func (s *Store) CreateProxy(ctx context.Context, in ProxyInput) (model.Proxy, error) {
	if strings.TrimSpace(in.Name) == "" {
		return model.Proxy{}, ErrEmptyField
	}
	if !validProxyType(in.ProxyType) {
		return model.Proxy{}, ErrInvalidProxyType
	}
	if !validPort(in.LocalPort) {
		return model.Proxy{}, ErrInvalidPort
	}
	localIP := strings.TrimSpace(in.LocalIP)
	if localIP == "" {
		localIP = model.DefaultLocalIP
	}

	resource, err := s.GetResourceByID(ctx, in.ResourceID)
	if err != nil {
		return model.Proxy{}, err // 可能为 ErrResourceNotFound
	}

	// 资源必须已授权给本租户且授权处于 enabled 状态。
	grant, err := s.GetGrantByResource(ctx, in.ResourceID)
	if errors.Is(err, ErrGrantNotFound) {
		return model.Proxy{}, ErrResourceNotGrantedToTenant
	}
	if err != nil {
		return model.Proxy{}, err
	}
	if grant.TenantID != in.TenantID || grant.Status != model.StatusEnabled {
		return model.Proxy{}, ErrResourceNotGrantedToTenant
	}

	// proxy 类型与资源类型必须匹配。
	if !proxyTypeMatchesResource(in.ProxyType, resource.Type) {
		return model.Proxy{}, ErrProxyTypeResourceMismatch
	}

	res, err := s.db.ExecContext(ctx,
		`INSERT INTO proxies (tenant_id, resource_id, name, proxy_type, local_ip, local_port, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		in.TenantID, in.ResourceID, in.Name, in.ProxyType, localIP, in.LocalPort, model.StatusEnabled)
	if err != nil {
		if isUniqueViolation(err) {
			return model.Proxy{}, ErrDuplicateProxyName
		}
		return model.Proxy{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return model.Proxy{}, err
	}
	return s.getProxyByID(ctx, id)
}

func (s *Store) getProxyByID(ctx context.Context, id int64) (model.Proxy, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+proxyColumns+` FROM proxies WHERE id = ?`, id)
	return scanProxy(row)
}

// GetProxyByTenantAndName 按租户与名称查询 proxy,以 (proxy, found, error) 形式返回存在性。
func (s *Store) GetProxyByTenantAndName(ctx context.Context, tenantID int64, name string) (model.Proxy, bool, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+proxyColumns+` FROM proxies WHERE tenant_id = ? AND name = ?`, tenantID, name)
	p, err := scanProxy(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Proxy{}, false, nil
	}
	if err != nil {
		return model.Proxy{}, false, err
	}
	return p, true, nil
}

// ListProxiesByTenant 返回某租户的全部 proxy,按 id 升序。
func (s *Store) ListProxiesByTenant(ctx context.Context, tenantID int64) ([]model.Proxy, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+proxyColumns+` FROM proxies WHERE tenant_id = ? ORDER BY id`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.Proxy
	for rows.Next() {
		p, err := scanProxy(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

// UpdateProxyStatus 更新 proxy 状态(enabled/disabled)。不存在返回 sql.ErrNoRows。
func (s *Store) UpdateProxyStatus(ctx context.Context, id int64, status string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE proxies SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, status, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// UpdateProxyStatusByID 按 proxy id 更新状态(enabled/disabled)。不存在返回 ErrProxyNotFound。
func (s *Store) UpdateProxyStatusByID(ctx context.Context, id int64, status string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE proxies SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, status, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrProxyNotFound
	}
	return nil
}

// ListProxies 返回全部 proxy,按 id 升序。
func (s *Store) ListProxies(ctx context.Context) ([]model.Proxy, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+proxyColumns+` FROM proxies ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []model.Proxy
	for rows.Next() {
		p, err := scanProxy(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

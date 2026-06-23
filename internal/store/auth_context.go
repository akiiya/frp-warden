package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fengheasia/frp-warden/internal/model"
)

// ProxyAuthContext 聚合一次 NewProxy 鉴权所需的后台数据,避免 plugin 层散落多条 SQL。
//
// 字段含义:
//   - Proxy:后台登记的映射配置(按 tenant + name 命中)。
//   - Resource:该 proxy 引用的资源。
//   - Grant / GrantFound:该资源的授权记录及其是否存在(资源未授权时 GrantFound 为 false)。
//
// 注意:本结构只负责"取数据";状态(enabled/available)、类型匹配、subdomain/remote_port
// 值是否一致等鉴权判断由 plugin 层完成,以便给出明确的中文 reject 原因。
type ProxyAuthContext struct {
	Proxy      model.Proxy
	Resource   model.Resource
	Grant      model.ResourceGrant
	GrantFound bool
}

// GetProxyAuthContext 按 (tenantID, proxyName) 组装 NewProxy 鉴权所需的上下文。
//
// proxy 不存在返回 ErrProxyNotFound;其余查询错误原样返回(plugin 层据此 fail-closed)。
func (s *Store) GetProxyAuthContext(ctx context.Context, tenantID int64, proxyName string) (ProxyAuthContext, error) {
	proxy, found, err := s.GetProxyByTenantAndName(ctx, tenantID, proxyName)
	if err != nil {
		return ProxyAuthContext{}, err
	}
	if !found {
		return ProxyAuthContext{}, ErrProxyNotFound
	}

	resource, err := s.GetResourceByID(ctx, proxy.ResourceID)
	if err != nil {
		return ProxyAuthContext{}, err
	}

	out := ProxyAuthContext{Proxy: proxy, Resource: resource}

	grant, err := s.GetGrantByResource(ctx, resource.ID)
	switch {
	case err == nil:
		out.Grant = grant
		out.GrantFound = true
	case errors.Is(err, ErrGrantNotFound):
		out.GrantFound = false
	default:
		return ProxyAuthContext{}, err
	}
	return out, nil
}

// ListEnabledProxyAuthContextsByTenant 返回某租户所有 enabled proxy 的鉴权上下文,
// 用于 frpc 配置生成。只返回 proxy.status=enabled、resource.status=available、
// grant.status=enabled 且 grant.tenant_id=tenantID 的记录,按 proxy.name 排序。
func (s *Store) ListEnabledProxyAuthContextsByTenant(ctx context.Context, tenantID int64) ([]ProxyAuthContext, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT p.id, p.tenant_id, p.resource_id, p.name, p.proxy_type, p.local_ip, p.local_port, p.status,
		        r.id, r.type, r.value, r.domain_zone_id, r.status,
		        g.id, g.tenant_id, g.resource_id, g.status
		 FROM proxies p
		 JOIN resources r ON r.id = p.resource_id
		 JOIN resource_grants g ON g.resource_id = r.id AND g.tenant_id = p.tenant_id
		 WHERE p.tenant_id = ?
		   AND p.status = 'enabled'
		   AND r.status = 'available'
		   AND g.status = 'enabled'
		 ORDER BY p.name`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []ProxyAuthContext
	for rows.Next() {
		var ac ProxyAuthContext
		var resZoneID sql.NullInt64
		if err := rows.Scan(
			&ac.Proxy.ID, &ac.Proxy.TenantID, &ac.Proxy.ResourceID,
			&ac.Proxy.Name, &ac.Proxy.ProxyType, &ac.Proxy.LocalIP, &ac.Proxy.LocalPort, &ac.Proxy.Status,
			&ac.Resource.ID, &ac.Resource.Type, &ac.Resource.Value, &resZoneID, &ac.Resource.Status,
			&ac.Grant.ID, &ac.Grant.TenantID, &ac.Grant.ResourceID, &ac.Grant.Status,
		); err != nil {
			return nil, err
		}
		if resZoneID.Valid {
			v := resZoneID.Int64
			ac.Resource.DomainZoneID = &v
		}
		ac.GrantFound = true
		list = append(list, ac)
	}
	return list, rows.Err()
}

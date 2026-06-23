package store

import (
	"context"
	"errors"
	"testing"

	"github.com/fengheasia/frp-warden/internal/model"
)

// enabledZone 在 store 中创建一个 enabled 的顶级域名区域,返回其 id。
func enabledZone(t *testing.T, s *Store) int64 {
	t.Helper()
	z, err := s.CreateDomainZone(context.Background(), "区域", "*.frp.example.com")
	if err != nil {
		t.Fatalf("创建区域失败: %v", err)
	}
	return z.ID
}

// 资源:有 enabled 区域后可建 subdomain;(type,value) 唯一;可建端口资源;非法端口报错。
func TestResource(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	zoneID := enabledZone(t, s)

	sub, err := s.CreateSubdomainResource(ctx, "ufi001", zoneID)
	if err != nil {
		t.Fatalf("创建 subdomain 资源失败: %v", err)
	}
	if sub.Type != model.ResourceTypeSubdomain || sub.DomainZoneID == nil || *sub.DomainZoneID != zoneID {
		t.Errorf("subdomain 资源结果异常: %+v", sub)
	}

	// (type, value) 唯一约束。
	if _, err := s.CreateSubdomainResource(ctx, "ufi001", zoneID); !errors.Is(err, ErrDuplicateResource) {
		t.Errorf("重复 subdomain 应返回 ErrDuplicateResource,实际 %v", err)
	}

	// 端口资源。
	tcp, err := s.CreateTCPPortResource(ctx, 61001)
	if err != nil {
		t.Fatalf("创建 tcp_port 资源失败: %v", err)
	}
	if tcp.Type != model.ResourceTypeTCPPort || tcp.DomainZoneID != nil {
		t.Errorf("tcp_port 资源结果异常: %+v", tcp)
	}
	if _, err := s.CreateUDPPortResource(ctx, 61002); err != nil {
		t.Fatalf("创建 udp_port 资源失败: %v", err)
	}

	// 非法端口。
	if _, err := s.CreateTCPPortResource(ctx, 0); !errors.Is(err, ErrInvalidPort) {
		t.Errorf("端口 0 应返回 ErrInvalidPort,实际 %v", err)
	}
	if _, err := s.CreateTCPPortResource(ctx, 70000); !errors.Is(err, ErrInvalidPort) {
		t.Errorf("端口 70000 应返回 ErrInvalidPort,实际 %v", err)
	}
}

// 授权:可授权;同一资源不能授权给第二个租户;禁用资源/禁用租户不可授权。
func TestGrant(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	zoneID := enabledZone(t, s)

	a, _ := s.CreateTenant(ctx, TenantInput{Code: "a", Name: "A", TokenHash: "ha"})
	b, _ := s.CreateTenant(ctx, TenantInput{Code: "b", Name: "B", TokenHash: "hb"})
	res, _ := s.CreateSubdomainResource(ctx, "ufi001", zoneID)

	// 正常授权给 A。
	if _, err := s.GrantResourceToTenant(ctx, a.ID, res.ID); err != nil {
		t.Fatalf("授权给 A 失败: %v", err)
	}

	// 同一资源不能再授权给 B —— 多租户资源隔离的关键。
	if _, err := s.GrantResourceToTenant(ctx, b.ID, res.ID); !errors.Is(err, ErrResourceAlreadyGranted) {
		t.Errorf("重复授权应返回 ErrResourceAlreadyGranted,实际 %v", err)
	}

	// 禁用资源不可授权。
	res2, _ := s.CreateTCPPortResource(ctx, 61001)
	if err := s.UpdateResourceStatus(ctx, res2.ID, model.ResourceStatusDisabled); err != nil {
		t.Fatalf("禁用资源失败: %v", err)
	}
	if _, err := s.GrantResourceToTenant(ctx, a.ID, res2.ID); !errors.Is(err, ErrResourceDisabled) {
		t.Errorf("禁用资源授权应返回 ErrResourceDisabled,实际 %v", err)
	}

	// 禁用租户不可授权。
	res3, _ := s.CreateTCPPortResource(ctx, 61002)
	if err := s.UpdateTenantStatus(ctx, b.ID, model.StatusDisabled); err != nil {
		t.Fatalf("禁用租户失败: %v", err)
	}
	if _, err := s.GrantResourceToTenant(ctx, b.ID, res3.ID); !errors.Is(err, ErrTenantDisabled) {
		t.Errorf("禁用租户授权应返回 ErrTenantDisabled,实际 %v", err)
	}
}

// proxy:已授权 subdomain 可建 http proxy;未授权资源不可建;类型不匹配报错;name 唯一;非法端口报错。
func TestProxy(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	zoneID := enabledZone(t, s)

	tn, _ := s.CreateTenant(ctx, TenantInput{Code: "wifi", Name: "WiFi", TokenHash: "h"})
	sub, _ := s.CreateSubdomainResource(ctx, "ufi001", zoneID)
	tcp, _ := s.CreateTCPPortResource(ctx, 61001)

	// 把 subdomain 授权给租户。
	if _, err := s.GrantResourceToTenant(ctx, tn.ID, sub.ID); err != nil {
		t.Fatalf("授权 subdomain 失败: %v", err)
	}

	// 用已授权 subdomain 创建 http proxy。
	p, err := s.CreateProxy(ctx, ProxyInput{
		TenantID: tn.ID, ResourceID: sub.ID, Name: "web", ProxyType: model.ProxyTypeHTTP, LocalPort: 8080,
	})
	if err != nil {
		t.Fatalf("创建 http proxy 失败: %v", err)
	}
	if p.LocalIP != model.DefaultLocalIP {
		t.Errorf("local_ip 默认应为 %s,实际 %s", model.DefaultLocalIP, p.LocalIP)
	}

	// 未授权资源(tcp 未授权)不可创建 proxy。
	if _, err := s.CreateProxy(ctx, ProxyInput{
		TenantID: tn.ID, ResourceID: tcp.ID, Name: "t1", ProxyType: model.ProxyTypeTCP, LocalPort: 9000,
	}); !errors.Is(err, ErrResourceNotGrantedToTenant) {
		t.Errorf("未授权资源应返回 ErrResourceNotGrantedToTenant,实际 %v", err)
	}

	// http proxy 不能使用 tcp_port 资源(先把 tcp 授权给租户,验证类型不匹配)。
	if _, err := s.GrantResourceToTenant(ctx, tn.ID, tcp.ID); err != nil {
		t.Fatalf("授权 tcp 失败: %v", err)
	}
	if _, err := s.CreateProxy(ctx, ProxyInput{
		TenantID: tn.ID, ResourceID: tcp.ID, Name: "t2", ProxyType: model.ProxyTypeHTTP, LocalPort: 9000,
	}); !errors.Is(err, ErrProxyTypeResourceMismatch) {
		t.Errorf("http+tcp_port 应返回 ErrProxyTypeResourceMismatch,实际 %v", err)
	}

	// tcp proxy 不能使用 subdomain 资源。
	if _, err := s.CreateProxy(ctx, ProxyInput{
		TenantID: tn.ID, ResourceID: sub.ID, Name: "t3", ProxyType: model.ProxyTypeTCP, LocalPort: 9000,
	}); !errors.Is(err, ErrProxyTypeResourceMismatch) {
		t.Errorf("tcp+subdomain 应返回 ErrProxyTypeResourceMismatch,实际 %v", err)
	}

	// 同一租户下 proxy name 唯一(web 已存在)。
	if _, err := s.CreateProxy(ctx, ProxyInput{
		TenantID: tn.ID, ResourceID: sub.ID, Name: "web", ProxyType: model.ProxyTypeHTTP, LocalPort: 8081,
	}); !errors.Is(err, ErrDuplicateProxyName) {
		t.Errorf("重名 proxy 应返回 ErrDuplicateProxyName,实际 %v", err)
	}

	// 非法 local_port。
	if _, err := s.CreateProxy(ctx, ProxyInput{
		TenantID: tn.ID, ResourceID: tcp.ID, Name: "t4", ProxyType: model.ProxyTypeTCP, LocalPort: 0,
	}); !errors.Is(err, ErrInvalidPort) {
		t.Errorf("非法 local_port 应返回 ErrInvalidPort,实际 %v", err)
	}
}

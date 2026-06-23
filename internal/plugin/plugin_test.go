package plugin

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fengheasia/frp-warden/internal/config"
	"github.com/fengheasia/frp-warden/internal/db"
	"github.com/fengheasia/frp-warden/internal/model"
	"github.com/fengheasia/frp-warden/internal/security"
	"github.com/fengheasia/frp-warden/internal/store"
)

const goodToken = "good-token-123"

// fixture 持有一套预置好的数据,供各用例复用。
type fixture struct {
	h     *Handler
	st    *store.Store
	sdb   *sql.DB
	tenID int64
	subID int64
	tcpID int64
	udpID int64
	// 各资源对应的 grant id,便于禁用。
	subGrantID int64
	// 各 proxy id,便于禁用。
	webProxyID int64
}

// newFixture 在临时 SQLite 上迁移并预置:enabled 区域、enabled 租户(带 token hash)、
// subdomain/tcp/udp 资源、对应授权,以及 web(http)/ssh(tcp)/game(udp) 三个 proxy。
func newFixture(t *testing.T) *fixture {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "plugin.db")
	sdb, dialect, err := db.Open(config.DatabaseConfig{Driver: "sqlite", DSN: dsn})
	if err != nil {
		t.Fatalf("打开数据库失败: %v", err)
	}
	t.Cleanup(func() { _ = sdb.Close() })
	if _, err := db.Migrate(context.Background(), sdb, dialect); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}

	st := store.New(sdb)
	ctx := context.Background()

	zone, err := st.CreateDomainZone(ctx, "区域", "*.frp.example.com")
	if err != nil {
		t.Fatalf("创建区域失败: %v", err)
	}
	hash, err := security.HashSecret(goodToken)
	if err != nil {
		t.Fatalf("计算 token hash 失败: %v", err)
	}
	tn, err := st.CreateTenant(ctx, store.TenantInput{Code: "ufi001", Name: "随身WiFi", TokenHash: hash})
	if err != nil {
		t.Fatalf("创建租户失败: %v", err)
	}
	sub, _ := st.CreateSubdomainResource(ctx, "ufi001", zone.ID)
	tcp, _ := st.CreateTCPPortResource(ctx, 61001)
	udp, _ := st.CreateUDPPortResource(ctx, 62001)

	subGrant, _ := st.GrantResourceToTenant(ctx, tn.ID, sub.ID)
	if _, err := st.GrantResourceToTenant(ctx, tn.ID, tcp.ID); err != nil {
		t.Fatalf("授权 tcp 失败: %v", err)
	}
	if _, err := st.GrantResourceToTenant(ctx, tn.ID, udp.ID); err != nil {
		t.Fatalf("授权 udp 失败: %v", err)
	}

	web, _ := st.CreateProxy(ctx, store.ProxyInput{TenantID: tn.ID, ResourceID: sub.ID, Name: "web", ProxyType: model.ProxyTypeHTTP, LocalPort: 8080})
	st.CreateProxy(ctx, store.ProxyInput{TenantID: tn.ID, ResourceID: tcp.ID, Name: "ssh", ProxyType: model.ProxyTypeTCP, LocalPort: 22})
	st.CreateProxy(ctx, store.ProxyInput{TenantID: tn.ID, ResourceID: udp.ID, Name: "game", ProxyType: model.ProxyTypeUDP, LocalPort: 5000})

	return &fixture{
		h: NewHandler(st), st: st, sdb: sdb,
		tenID: tn.ID, subID: sub.ID, tcpID: tcp.ID, udpID: udp.ID,
		subGrantID: subGrant.ID, webProxyID: web.ID,
	}
}

// do 向 handler 发起一次请求,返回状态码与解析后的响应。
func do(t *testing.T, h *Handler, op string, content any) (int, pluginResponse) {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"content": content})
	req := httptest.NewRequest(http.MethodPost, "/plugin/frp?op="+op, bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var resp pluginResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	return rec.Code, resp
}

func userObj(code, token string) map[string]any {
	return map[string]any{"user": code, "metas": map[string]any{"token": token}}
}

// ---------------- HTTP / 解析层 ----------------

func TestPluginHTTPLayer(t *testing.T) {
	f := newFixture(t)

	// GET → 405
	req := httptest.NewRequest(http.MethodGet, "/plugin/frp?op=Login", nil)
	rec := httptest.NewRecorder()
	f.h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应返回 405,实际 %d", rec.Code)
	}

	// 无效 JSON → 400
	req = httptest.NewRequest(http.MethodPost, "/plugin/frp?op=Login", strings.NewReader("{not-json"))
	rec = httptest.NewRecorder()
	f.h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("无效 JSON 应返回 400,实际 %d", rec.Code)
	}

	// 请求体过大 → 413
	big := strings.Repeat("a", int(defaultMaxBodyBytes)+1024)
	body, _ := json.Marshal(map[string]any{"content": map[string]any{"user": big}})
	req = httptest.NewRequest(http.MethodPost, "/plugin/frp?op=Login", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	f.h.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("超大请求体应返回 413,实际 %d", rec.Code)
	}

	// 未知 op → 200 + reject
	code, resp := do(t, f.h, "Unknown", map[string]any{"user": "x"})
	if code != http.StatusOK || !resp.Reject {
		t.Errorf("未知 op 应 200+reject,实际 code=%d reject=%v", code, resp.Reject)
	}
}

// ---------------- Login ----------------

func TestLogin(t *testing.T) {
	f := newFixture(t)
	cases := []struct {
		name       string
		content    map[string]any
		wantReject bool
	}{
		{"正确凭据", map[string]any{"user": "ufi001", "metas": map[string]any{"token": goodToken}}, false},
		{"缺少 user", map[string]any{"metas": map[string]any{"token": goodToken}}, true},
		{"缺少 token", map[string]any{"user": "ufi001"}, true},
		{"租户不存在", map[string]any{"user": "nope", "metas": map[string]any{"token": goodToken}}, true},
		{"token 错误", map[string]any{"user": "ufi001", "metas": map[string]any{"token": "wrong"}}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, resp := do(t, f.h, "Login", c.content)
			if code != http.StatusOK {
				t.Fatalf("应返回 200,实际 %d", code)
			}
			if resp.Reject != c.wantReject {
				t.Errorf("reject = %v,期望 %v(原因:%s)", resp.Reject, c.wantReject, resp.RejectReason)
			}
		})
	}
}

func TestLoginDisabledTenant(t *testing.T) {
	f := newFixture(t)
	if err := f.st.UpdateTenantStatus(context.Background(), f.tenID, model.StatusDisabled); err != nil {
		t.Fatalf("禁用租户失败: %v", err)
	}
	_, resp := do(t, f.h, "Login", map[string]any{"user": "ufi001", "metas": map[string]any{"token": goodToken}})
	if !resp.Reject {
		t.Error("禁用租户应被拒绝")
	}
}

// ---------------- NewProxy: http ----------------

func TestNewProxyHTTP(t *testing.T) {
	f := newFixture(t)

	// 正确:已授权 subdomain + 正确 proxy_name + 正确 token → allow
	_, resp := do(t, f.h, "NewProxy", map[string]any{
		"user": userObj("ufi001", goodToken), "proxy_name": "web", "proxy_type": "http", "subdomain": "ufi001",
	})
	if resp.Reject {
		t.Fatalf("合法 http proxy 不应被拒绝:%s", resp.RejectReason)
	}

	// custom_domains → reject
	_, resp = do(t, f.h, "NewProxy", map[string]any{
		"user": userObj("ufi001", goodToken), "proxy_name": "web", "proxy_type": "http",
		"subdomain": "ufi001", "custom_domains": []string{"evil.example.com"},
	})
	if !resp.Reject {
		t.Error("使用 custom_domains 应被拒绝")
	}

	// subdomain 不匹配 → reject
	_, resp = do(t, f.h, "NewProxy", map[string]any{
		"user": userObj("ufi001", goodToken), "proxy_name": "web", "proxy_type": "http", "subdomain": "other",
	})
	if !resp.Reject {
		t.Error("subdomain 不匹配应被拒绝")
	}

	// proxy_name 不存在 → reject
	_, resp = do(t, f.h, "NewProxy", map[string]any{
		"user": userObj("ufi001", goodToken), "proxy_name": "ghost", "proxy_type": "http", "subdomain": "ufi001",
	})
	if !resp.Reject {
		t.Error("不存在的 proxy 应被拒绝")
	}

	// 用 http 去碰一个 tcp 资源的 proxy(ssh)→ reject(后台类型不一致)
	_, resp = do(t, f.h, "NewProxy", map[string]any{
		"user": userObj("ufi001", goodToken), "proxy_name": "ssh", "proxy_type": "http", "subdomain": "ufi001",
	})
	if !resp.Reject {
		t.Error("http 请求碰 tcp proxy 应被拒绝")
	}

	// token 错误 → reject
	_, resp = do(t, f.h, "NewProxy", map[string]any{
		"user": userObj("ufi001", "wrong"), "proxy_name": "web", "proxy_type": "http", "subdomain": "ufi001",
	})
	if !resp.Reject {
		t.Error("token 错误应被拒绝")
	}
}

func TestNewProxyHTTPDisabledChains(t *testing.T) {
	ctx := context.Background()
	req := func(f *fixture) (int, pluginResponse) {
		return do(t, f.h, "NewProxy", map[string]any{
			"user": userObj("ufi001", goodToken), "proxy_name": "web", "proxy_type": "http", "subdomain": "ufi001",
		})
	}

	// proxy disabled → reject
	f := newFixture(t)
	if err := f.st.UpdateProxyStatus(ctx, f.webProxyID, model.StatusDisabled); err != nil {
		t.Fatalf("禁用 proxy 失败: %v", err)
	}
	if _, resp := req(f); !resp.Reject {
		t.Error("proxy 禁用应被拒绝")
	}

	// grant disabled → reject
	f = newFixture(t)
	if err := f.st.UpdateGrantStatus(ctx, f.subGrantID, model.StatusDisabled); err != nil {
		t.Fatalf("禁用授权失败: %v", err)
	}
	if _, resp := req(f); !resp.Reject {
		t.Error("授权禁用应被拒绝")
	}

	// resource disabled → reject
	f = newFixture(t)
	if err := f.st.UpdateResourceStatus(ctx, f.subID, model.ResourceStatusDisabled); err != nil {
		t.Fatalf("禁用资源失败: %v", err)
	}
	if _, resp := req(f); !resp.Reject {
		t.Error("资源禁用应被拒绝")
	}
}

// ---------------- NewProxy: tcp / udp ----------------

func TestNewProxyTCPUDP(t *testing.T) {
	f := newFixture(t)

	// tcp 正确 → allow
	_, resp := do(t, f.h, "NewProxy", map[string]any{
		"user": userObj("ufi001", goodToken), "proxy_name": "ssh", "proxy_type": "tcp", "remote_port": 61001,
	})
	if resp.Reject {
		t.Fatalf("合法 tcp proxy 不应被拒绝:%s", resp.RejectReason)
	}

	// tcp remote_port 不匹配 → reject
	_, resp = do(t, f.h, "NewProxy", map[string]any{
		"user": userObj("ufi001", goodToken), "proxy_name": "ssh", "proxy_type": "tcp", "remote_port": 9999,
	})
	if !resp.Reject {
		t.Error("remote_port 不匹配应被拒绝")
	}

	// 用 tcp 去碰 subdomain 资源的 proxy(web)→ reject
	_, resp = do(t, f.h, "NewProxy", map[string]any{
		"user": userObj("ufi001", goodToken), "proxy_name": "web", "proxy_type": "tcp", "remote_port": 61001,
	})
	if !resp.Reject {
		t.Error("tcp 请求碰 subdomain proxy 应被拒绝")
	}

	// udp 正确 → allow
	_, resp = do(t, f.h, "NewProxy", map[string]any{
		"user": userObj("ufi001", goodToken), "proxy_name": "game", "proxy_type": "udp", "remote_port": 62001,
	})
	if resp.Reject {
		t.Fatalf("合法 udp proxy 不应被拒绝:%s", resp.RejectReason)
	}

	// udp remote_port 不匹配 → reject
	_, resp = do(t, f.h, "NewProxy", map[string]any{
		"user": userObj("ufi001", goodToken), "proxy_name": "game", "proxy_type": "udp", "remote_port": 1,
	})
	if !resp.Reject {
		t.Error("udp remote_port 不匹配应被拒绝")
	}
}

// ---------------- Ping ----------------

func TestPing(t *testing.T) {
	f := newFixture(t)

	// 正确 token → allow,并更新 last_seen_at
	_, resp := do(t, f.h, "Ping", map[string]any{"user": userObj("ufi001", goodToken)})
	if resp.Reject {
		t.Fatalf("合法 Ping 不应被拒绝:%s", resp.RejectReason)
	}
	// 直接查库验证 last_seen_at 已被更新(store 的常规查询不读取该列)。
	var lastSeen sql.NullString
	if err := f.sdb.QueryRow(`SELECT last_seen_at FROM tenants WHERE id = ?`, f.tenID).Scan(&lastSeen); err != nil {
		t.Fatalf("查询 last_seen_at 失败: %v", err)
	}
	if !lastSeen.Valid {
		t.Error("Ping 后 last_seen_at 应被更新")
	}

	// token 错误 → reject
	_, resp = do(t, f.h, "Ping", map[string]any{"user": userObj("ufi001", "wrong")})
	if !resp.Reject {
		t.Error("token 错误的 Ping 应被拒绝")
	}

	// disabled tenant → reject
	if err := f.st.UpdateTenantStatus(context.Background(), f.tenID, model.StatusDisabled); err != nil {
		t.Fatalf("禁用租户失败: %v", err)
	}
	_, resp = do(t, f.h, "Ping", map[string]any{"user": userObj("ufi001", goodToken)})
	if !resp.Reject {
		t.Error("禁用租户的 Ping 应被拒绝")
	}
}

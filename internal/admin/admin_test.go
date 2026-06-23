package admin

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
	"github.com/fengheasia/frp-warden/internal/security"
	"github.com/fengheasia/frp-warden/internal/store"
)

// testEnv 持有测试环境的全部组件。
type testEnv struct {
	h       *Handler
	st      *store.Store
	sdb     *sql.DB
	cookie  string // 登录后的 session cookie
	adminID int64
}

// newTestEnv 在临时 SQLite 上迁移、创建默认管理员、登录并返回带 cookie 的测试环境。
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "admin.db")
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

	// 创建默认管理员(密码 "testpass123")。
	hash, _ := security.HashPassword("testpass123")
	_, err = sdb.ExecContext(ctx,
		`INSERT INTO admins (username, password_hash, must_change_password, status) VALUES (?, ?, 1, 'enabled')`,
		"admin", hash)
	if err != nil {
		t.Fatalf("创建管理员失败: %v", err)
	}
	var adminID int64
	_ = sdb.QueryRow(`SELECT id FROM admins WHERE username = 'admin'`).Scan(&adminID)

	cfg := config.Default()
	h := NewHandler(st, cfg)

	// 登录获取 cookie。
	body, _ := json.Marshal(map[string]string{"username": "admin", "password": "testpass123"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("登录失败: %d %s", rec.Code, rec.Body.String())
	}
	cookie := extractCookie(rec, sessionCookieName)
	if cookie == "" {
		t.Fatal("登录后未设置 cookie")
	}

	return &testEnv{h: h, st: st, sdb: sdb, cookie: cookie, adminID: adminID}
}

// do 发起一次带 cookie 的请求。
func (e *testEnv) do(method, path string, body any) *httptest.ResponseRecorder {
	var reqBody *bytes.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		reqBody = bytes.NewReader(data)
	} else {
		reqBody = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reqBody)
	if e.cookie != "" {
		req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: e.cookie})
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	return rec
}

// extractCookie 从响应中提取指定 cookie 的值。
func extractCookie(rec *httptest.ResponseRecorder, name string) string {
	for _, c := range rec.Result().Cookies() {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}

// parseResp 解析 API 响应。
func parseResp(t *testing.T, rec *httptest.ResponseRecorder) apiResponse {
	t.Helper()
	var resp apiResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v body=%s", err, rec.Body.String())
	}
	return resp
}

// ========== Auth 测试 ==========

func TestLoginSuccess(t *testing.T) {
	e := newTestEnv(t)
	// newTestEnv 已经登录成功,验证 cookie 存在即可。
	if e.cookie == "" {
		t.Error("登录后应有 cookie")
	}
}

func TestLoginFailureNoLeak(t *testing.T) {
	e := newTestEnv(t)
	// 用错误密码登录。
	body, _ := json.Marshal(map[string]string{"username": "admin", "password": "wrong"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("错误密码应返回 401,实际 %d", rec.Code)
	}
	resp := parseResp(t, rec)
	if resp.Err == nil || !strings.Contains(resp.Err.Message, "用户名或密码错误") {
		t.Errorf("应统一提示'用户名或密码错误',实际 %v", resp.Err)
	}
	// 不应泄露"用户名存在但密码错误"——统一提示"用户名或密码错误"。
	// 不应出现"用户名不存在"或"密码错误"等区分性提示。
	if resp.Err.Message != "用户名或密码错误" {
		t.Errorf("应统一提示'用户名或密码错误',实际 '%s'", resp.Err.Message)
	}
}

func TestMeUnauthorized(t *testing.T) {
	e := newTestEnv(t)
	e.cookie = "" // 清除 cookie。
	rec := e.do(http.MethodGet, "/api/auth/me", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("未登录应返回 401,实际 %d", rec.Code)
	}
}

func TestMeAfterLogin(t *testing.T) {
	e := newTestEnv(t)
	rec := e.do(http.MethodGet, "/api/auth/me", nil)
	if rec.Code != http.StatusOK {
		t.Errorf("登录后 /api/auth/me 应返回 200,实际 %d", rec.Code)
	}
	resp := parseResp(t, rec)
	data, _ := resp.Data.(map[string]any)
	if data["username"] != "admin" {
		t.Errorf("username = %v,期望 admin", data["username"])
	}
}

func TestLogout(t *testing.T) {
	e := newTestEnv(t)
	rec := e.do(http.MethodPost, "/api/auth/logout", nil)
	if rec.Code != http.StatusOK {
		t.Errorf("logout 应返回 200,实际 %d", rec.Code)
	}
	// logout 后 session 应失效。
	rec2 := e.do(http.MethodGet, "/api/auth/me", nil)
	if rec2.Code != http.StatusUnauthorized {
		t.Errorf("logout 后 /api/auth/me 应返回 401,实际 %d", rec2.Code)
	}
}

func TestChangePassword(t *testing.T) {
	e := newTestEnv(t)

	// 旧密码错误。
	rec := e.do(http.MethodPost, "/api/auth/change-password", map[string]string{
		"old_password": "wrong", "new_password": "newpass12345",
	})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("旧密码错误应返回 400,实际 %d", rec.Code)
	}

	// 新密码太短。
	rec = e.do(http.MethodPost, "/api/auth/change-password", map[string]string{
		"old_password": "testpass123", "new_password": "short",
	})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("弱密码应返回 400,实际 %d", rec.Code)
	}

	// 成功修改。
	rec = e.do(http.MethodPost, "/api/auth/change-password", map[string]string{
		"old_password": "testpass123", "new_password": "newpass12345",
	})
	if rec.Code != http.StatusOK {
		t.Errorf("修改密码应返回 200,实际 %d", rec.Code)
	}

	// must_change_password 应为 false。
	rec = e.do(http.MethodGet, "/api/auth/me", nil)
	resp := parseResp(t, rec)
	data, _ := resp.Data.(map[string]any)
	if data["must_change_password"] != false {
		t.Errorf("修改密码后 must_change_password 应为 false,实际 %v", data["must_change_password"])
	}

	// 用新密码登录。
	body, _ := json.Marshal(map[string]string{"username": "admin", "password": "newpass12345"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	rec2 := httptest.NewRecorder()
	e.h.ServeHTTP(rec2, req)
	if rec2.Code != http.StatusOK {
		t.Errorf("新密码登录应成功,实际 %d", rec2.Code)
	}
}

// ========== Tenant API 测试 ==========

func TestTenantCRUD(t *testing.T) {
	e := newTestEnv(t)

	// 未登录返回 401。
	e2 := &testEnv{h: e.h}
	rec := e2.do(http.MethodGet, "/api/tenants", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("未登录应返回 401,实际 %d", rec.Code)
	}

	// 创建 tenant。
	rec = e.do(http.MethodPost, "/api/tenants", map[string]string{
		"code": "ufi001", "name": "测试设备",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("创建 tenant 应返回 200,实际 %d %s", rec.Code, rec.Body.String())
	}
	resp := parseResp(t, rec)
	data, _ := resp.Data.(map[string]any)
	plainToken, _ := data["plain_token"].(string)
	if plainToken == "" {
		t.Error("创建 tenant 应返回 plain_token")
	}
	tenant, _ := data["tenant"].(map[string]any)
	if tenant["code"] != "ufi001" {
		t.Errorf("tenant code = %v,期望 ufi001", tenant["code"])
	}

	// 验证数据库只保存 hash,不保存明文。
	var tokenHash string
	_ = e.sdb.QueryRow(`SELECT token_hash FROM tenants WHERE code = 'ufi001'`).Scan(&tokenHash)
	if tokenHash == plainToken {
		t.Error("数据库不应保存明文 token")
	}
	if !security.VerifySecret(tokenHash, plainToken) {
		t.Error("token_hash 应能通过 bcrypt 校验 plain_token")
	}

	// code 重复返回 409。
	rec = e.do(http.MethodPost, "/api/tenants", map[string]string{
		"code": "ufi001", "name": "重复",
	})
	if rec.Code != http.StatusConflict {
		t.Errorf("重复 code 应返回 409,实际 %d", rec.Code)
	}

	// 列表。
	rec = e.do(http.MethodGet, "/api/tenants", nil)
	if rec.Code != http.StatusOK {
		t.Errorf("列表应返回 200,实际 %d", rec.Code)
	}

	// 禁用。
	tenantID := int64(tenant["id"].(float64))
	rec = e.do(http.MethodPost, "/api/tenants/"+i64s(tenantID)+"/disable", nil)
	if rec.Code != http.StatusOK {
		t.Errorf("禁用应返回 200,实际 %d", rec.Code)
	}

	// 启用。
	rec = e.do(http.MethodPost, "/api/tenants/"+i64s(tenantID)+"/enable", nil)
	if rec.Code != http.StatusOK {
		t.Errorf("启用应返回 200,实际 %d", rec.Code)
	}

	// 重置 token。
	rec = e.do(http.MethodPost, "/api/tenants/"+i64s(tenantID)+"/reset-token", nil)
	if rec.Code != http.StatusOK {
		t.Errorf("重置 token 应返回 200,实际 %d", rec.Code)
	}
	resp2 := parseResp(t, rec)
	data2, _ := resp2.Data.(map[string]any)
	newToken, _ := data2["plain_token"].(string)
	if newToken == "" || newToken == plainToken {
		t.Error("重置 token 应返回新的 plain_token")
	}
	// 新 token 的 hash 也应有效。
	var newHash string
	_ = e.sdb.QueryRow(`SELECT token_hash FROM tenants WHERE id = ?`, tenantID).Scan(&newHash)
	if !security.VerifySecret(newHash, newToken) {
		t.Error("重置后的 token_hash 应能校验新 plain_token")
	}
}

// ========== Resource/Grant/Proxy API 测试 ==========

func TestResourceGrantProxyAPI(t *testing.T) {
	e := newTestEnv(t)

	// 没有 domain_zone 时创建 subdomain resource 失败。
	rec := e.do(http.MethodPost, "/api/resources/subdomain", map[string]any{
		"domain_zone_id": 1, "value": "ufi001",
	})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("无 domain_zone 时应返回 400,实际 %d", rec.Code)
	}

	// 创建 domain_zone。
	rec = e.do(http.MethodPost, "/api/domain-zones", map[string]string{
		"name": "默认", "zone": "*.frp.example.com",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("创建 domain_zone 应返回 200,实际 %d %s", rec.Code, rec.Body.String())
	}
	resp := parseResp(t, rec)
	zoneData, _ := resp.Data.(map[string]any)
	zoneID := int64(zoneData["id"].(float64))

	// 创建 subdomain resource。
	rec = e.do(http.MethodPost, "/api/resources/subdomain", map[string]any{
		"domain_zone_id": zoneID, "value": "ufi001",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("创建 subdomain resource 应返回 200,实际 %d %s", rec.Code, rec.Body.String())
	}
	subResp := parseResp(t, rec)
	subData, _ := subResp.Data.(map[string]any)
	subResID := int64(subData["id"].(float64))

	// 创建 tcp_port resource。
	rec = e.do(http.MethodPost, "/api/resources/tcp-port", map[string]string{"value": "61001"})
	if rec.Code != http.StatusOK {
		t.Errorf("创建 tcp_port 应返回 200,实际 %d", rec.Code)
	}
	tcpResp := parseResp(t, rec)
	tcpData, _ := tcpResp.Data.(map[string]any)
	tcpResID := int64(tcpData["id"].(float64))

	// 创建 tenant。
	rec = e.do(http.MethodPost, "/api/tenants", map[string]string{"code": "wifi", "name": "WiFi"})
	if rec.Code != http.StatusOK {
		t.Fatalf("创建 tenant 应返回 200,实际 %d", rec.Code)
	}
	tnResp := parseResp(t, rec)
	tnData, _ := tnResp.Data.(map[string]any)
	tn, _ := tnData["tenant"].(map[string]any)
	tenantID := int64(tn["id"].(float64))

	// 授权 resource 给 tenant。
	rec = e.do(http.MethodPost, "/api/grants", map[string]any{
		"tenant_id": tenantID, "resource_id": subResID,
	})
	if rec.Code != http.StatusOK {
		t.Errorf("授权应返回 200,实际 %d %s", rec.Code, rec.Body.String())
	}

	// 同一 resource 再授权给另一个 tenant 返回 409。
	rec = e.do(http.MethodPost, "/api/tenants", map[string]string{"code": "wifi2", "name": "WiFi2"})
	tn2Resp := parseResp(t, rec)
	tn2Data, _ := tn2Resp.Data.(map[string]any)
	tn2, _ := tn2Data["tenant"].(map[string]any)
	tenant2ID := int64(tn2["id"].(float64))
	rec = e.do(http.MethodPost, "/api/grants", map[string]any{
		"tenant_id": tenant2ID, "resource_id": subResID,
	})
	if rec.Code != http.StatusConflict {
		t.Errorf("重复授权应返回 409,实际 %d", rec.Code)
	}

	// 创建 http proxy。
	rec = e.do(http.MethodPost, "/api/proxies", map[string]any{
		"tenant_id": tenantID, "resource_id": subResID, "name": "web",
		"proxy_type": "http", "local_port": 8080,
	})
	if rec.Code != http.StatusOK {
		t.Errorf("创建 http proxy 应返回 200,实际 %d %s", rec.Code, rec.Body.String())
	}

	// 用未授权 resource 创建 proxy 失败。
	rec = e.do(http.MethodPost, "/api/proxies", map[string]any{
		"tenant_id": tenantID, "resource_id": tcpResID, "name": "ssh",
		"proxy_type": "tcp", "local_port": 22,
	})
	if rec.Code != http.StatusConflict {
		t.Errorf("未授权 resource 应返回 409,实际 %d", rec.Code)
	}

	// 授权 tcp resource 给 tenant。
	rec = e.do(http.MethodPost, "/api/grants", map[string]any{
		"tenant_id": tenantID, "resource_id": tcpResID,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("授权 tcp 应返回 200,实际 %d", rec.Code)
	}

	// proxy_type 与 resource type 不匹配返回 409。
	rec = e.do(http.MethodPost, "/api/proxies", map[string]any{
		"tenant_id": tenantID, "resource_id": tcpResID, "name": "bad",
		"proxy_type": "http", "local_port": 8080,
	})
	if rec.Code != http.StatusConflict {
		t.Errorf("类型不匹配应返回 409,实际 %d", rec.Code)
	}

	// 创建 tcp proxy。
	rec = e.do(http.MethodPost, "/api/proxies", map[string]any{
		"tenant_id": tenantID, "resource_id": tcpResID, "name": "ssh",
		"proxy_type": "tcp", "local_port": 22,
	})
	if rec.Code != http.StatusOK {
		t.Errorf("创建 tcp proxy 应返回 200,实际 %d %s", rec.Code, rec.Body.String())
	}

	// disable/enable grant。
	rec = e.do(http.MethodGet, "/api/grants?tenant_id="+i64s(tenantID), nil)
	grantsResp := parseResp(t, rec)
	grantsList, _ := grantsResp.Data.([]any)
	grant0, _ := grantsList[0].(map[string]any)
	grantID := int64(grant0["id"].(float64))
	rec = e.do(http.MethodPost, "/api/grants/"+i64s(grantID)+"/disable", nil)
	if rec.Code != http.StatusOK {
		t.Errorf("disable grant 应返回 200,实际 %d", rec.Code)
	}
	rec = e.do(http.MethodPost, "/api/grants/"+i64s(grantID)+"/enable", nil)
	if rec.Code != http.StatusOK {
		t.Errorf("enable grant 应返回 200,实际 %d", rec.Code)
	}

	// disable/enable proxy。
	rec = e.do(http.MethodGet, "/api/proxies?tenant_id="+i64s(tenantID), nil)
	proxiesResp := parseResp(t, rec)
	proxiesList, _ := proxiesResp.Data.([]any)
	proxy0, _ := proxiesList[0].(map[string]any)
	proxyID := int64(proxy0["id"].(float64))
	rec = e.do(http.MethodPost, "/api/proxies/"+i64s(proxyID)+"/disable", nil)
	if rec.Code != http.StatusOK {
		t.Errorf("disable proxy 应返回 200,实际 %d", rec.Code)
	}
	rec = e.do(http.MethodPost, "/api/proxies/"+i64s(proxyID)+"/enable", nil)
	if rec.Code != http.StatusOK {
		t.Errorf("enable proxy 应返回 200,实际 %d", rec.Code)
	}
}

// ========== Audit 测试 ==========

func TestAuditLogs(t *testing.T) {
	e := newTestEnv(t)

	// 创建一个 tenant 触发审计。
	e.do(http.MethodPost, "/api/tenants", map[string]string{"code": "audit001", "name": "审计测试"})

	// 查询审计日志。
	rec := e.do(http.MethodGet, "/api/audit-logs", nil)
	if rec.Code != http.StatusOK {
		t.Errorf("查询审计日志应返回 200,实际 %d", rec.Code)
	}
	resp := parseResp(t, rec)
	logs, _ := resp.Data.([]any)
	if len(logs) == 0 {
		t.Error("应有审计日志记录")
	}

	// 审计日志不应包含 plain_token 或 password。
	for _, l := range logs {
		log, _ := l.(map[string]any)
		msg, _ := log["message"].(string)
		if strings.Contains(msg, "testpass") || strings.Contains(msg, "plain_token") {
			t.Errorf("审计日志不应包含敏感信息: %s", msg)
		}
	}
}

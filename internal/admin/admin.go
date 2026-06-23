// Package admin 实现管理后台 REST API。
//
// 本轮(Phase 5)实现了管理员登录/退出、session 认证、修改密码、获取当前管理员信息、
// tenant/resource/grant/proxy 的基础管理 API,以及审计日志写入与查询。
//
// 安全规则:
//   - 除 POST /api/auth/login 和 GET /api/healthz 外,所有路由要求登录(session cookie)。
//   - session token 明文只存于 cookie(HttpOnly + SameSiteLax),数据库只存 SHA-256 哈希。
//   - 不返回 password_hash / session_secret / token_hash 给前端。
//   - 不在日志/审计/错误中输出密码/明文 token/session token。
//   - 创建/重置 tenant token 时明文只返回一次,数据库只存 bcrypt hash。
//   - 登录失败不泄露"用户名是否存在",统一提示"用户名或密码错误"。
package admin

import (
	"net/http"

	"github.com/fengheasia/frp-warden/internal/config"
	"github.com/fengheasia/frp-warden/internal/store"
)

// Handler 管理后台 HTTP handler,聚合所有路由。
type Handler struct {
	store *store.Store
	cfg   config.Config
	mux   *http.ServeMux
}

// NewHandler 用给定 store 与配置构造管理后台 handler 并注册所有路由。
func NewHandler(st *store.Store, cfg config.Config) *Handler {
	h := &Handler{store: st, cfg: cfg, mux: http.NewServeMux()}
	h.registerRoutes()
	return h
}

// ServeHTTP 实现 http.Handler。
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

// registerRoutes 注册所有管理 API 路由。
func (h *Handler) registerRoutes() {
	// 公开路由(不要求登录)。
	h.mux.HandleFunc("/api/healthz", h.handleHealthz)
	h.mux.HandleFunc("/api/auth/login", h.handleLogin)

	// 认证相关(要求登录)。
	h.mux.HandleFunc("/api/auth/logout", h.requireAuth(h.handleLogout))
	h.mux.HandleFunc("/api/auth/me", h.requireAuth(h.handleMe))
	h.mux.HandleFunc("/api/auth/change-password", h.requireAuth(h.handleChangePassword))

	// 租户管理与 frpc 配置(要求登录)。
	h.mux.HandleFunc("/api/tenants", h.requireAuth(h.handleTenantRoutes))
	h.mux.HandleFunc("/api/tenants/", h.requireAuth(h.handleTenantAndFrpcRoutes))

	// 顶级域名区域(要求登录)。
	h.mux.HandleFunc("/api/domain-zones", h.requireAuth(h.handleDomainZoneRoutes))
	h.mux.HandleFunc("/api/domain-zones/", h.requireAuth(h.handleDomainZoneActionRoutes))

	// 资源管理(要求登录)。
	h.mux.HandleFunc("/api/resources", h.requireAuth(h.handleListResources))
	h.mux.HandleFunc("/api/resources/subdomain", h.requireAuth(h.handleCreateSubdomainResource))
	h.mux.HandleFunc("/api/resources/tcp-port", h.requireAuth(h.handleCreateTCPPortResource))
	h.mux.HandleFunc("/api/resources/udp-port", h.requireAuth(h.handleCreateUDPPortResource))

	// 授权管理(要求登录)。
	h.mux.HandleFunc("/api/grants", h.requireAuth(h.handleGrantRoutes))
	h.mux.HandleFunc("/api/grants/", h.requireAuth(h.handleGrantActionRoutes))

	// 映射管理(要求登录)。
	h.mux.HandleFunc("/api/proxies", h.requireAuth(h.handleProxyRoutes))
	h.mux.HandleFunc("/api/proxies/", h.requireAuth(h.handleProxyActionRoutes))

	// 审计日志(要求登录)。
	h.mux.HandleFunc("/api/audit-logs", h.requireAuth(h.handleListAuditLogs))
}

// handleHealthz 处理 GET /api/healthz。不要求登录。
func (h *Handler) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeOK(w, map[string]string{"status": "ok"})
}

// handleTenantAndFrpcRoutes 按路径分派 /api/tenants/ 下的所有路由。
// 包括 tenant 操作(disable/enable/reset-token)与 frpc 配置相关端点。
func (h *Handler) handleTenantAndFrpcRoutes(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(r.URL.Path)
	if len(parts) < 3 {
		writeError(w, http.StatusNotFound, CodeNotFound, "路由不存在")
		return
	}

	// /api/tenants/{id}/frpc-config 或 /api/tenants/{id}/frpc-config/download
	if len(parts) >= 4 && parts[3] == "frpc-config" {
		if len(parts) >= 5 && parts[4] == "download" {
			h.handleGetFrpcConfigDownload(w, r)
		} else {
			h.handleGetFrpcConfig(w, r)
		}
		return
	}

	// /api/tenants/{id}/disable|enable|reset-token
	if len(parts) >= 4 {
		action := parts[3]
		switch action {
		case "disable":
			h.handleDisableTenant(w, r)
		case "enable":
			h.handleEnableTenant(w, r)
		case "reset-token":
			h.handleResetTenantToken(w, r)
		default:
			writeError(w, http.StatusNotFound, CodeNotFound, "路由不存在")
		}
		return
	}

	writeError(w, http.StatusNotFound, CodeNotFound, "路由不存在")
}

// handleTenantRoutes 按方法分派 /api/tenants。
func (h *Handler) handleTenantRoutes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleListTenants(w, r)
	case http.MethodPost:
		h.handleCreateTenant(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, CodeBadRequest, "不支持的请求方法")
	}
}

// handleTenantActionRoutes 按路径分派 /api/tenants/{id}/*。
func (h *Handler) handleTenantActionRoutes(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(r.URL.Path)
	if len(parts) < 4 {
		writeError(w, http.StatusNotFound, CodeNotFound, "路由不存在")
		return
	}
	action := parts[3]
	switch action {
	case "disable":
		h.handleDisableTenant(w, r)
	case "enable":
		h.handleEnableTenant(w, r)
	case "reset-token":
		h.handleResetTenantToken(w, r)
	default:
		writeError(w, http.StatusNotFound, CodeNotFound, "路由不存在")
	}
}

// handleDomainZoneRoutes 按方法分派 /api/domain-zones。
func (h *Handler) handleDomainZoneRoutes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleListDomainZones(w, r)
	case http.MethodPost:
		h.handleCreateDomainZone(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, CodeBadRequest, "不支持的请求方法")
	}
}

// handleDomainZoneActionRoutes 按路径分派 /api/domain-zones/{id}/*。
func (h *Handler) handleDomainZoneActionRoutes(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(r.URL.Path)
	if len(parts) < 4 {
		writeError(w, http.StatusNotFound, CodeNotFound, "路由不存在")
		return
	}
	action := parts[3]
	switch action {
	case "enable":
		h.handleEnableDomainZone(w, r)
	case "disable":
		h.handleDisableDomainZone(w, r)
	default:
		writeError(w, http.StatusNotFound, CodeNotFound, "路由不存在")
	}
}

// handleGrantRoutes 按方法分派 /api/grants。
func (h *Handler) handleGrantRoutes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleListGrants(w, r)
	case http.MethodPost:
		h.handleCreateGrant(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, CodeBadRequest, "不支持的请求方法")
	}
}

// handleGrantActionRoutes 按路径分派 /api/grants/{id}/*。
func (h *Handler) handleGrantActionRoutes(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(r.URL.Path)
	if len(parts) < 4 {
		writeError(w, http.StatusNotFound, CodeNotFound, "路由不存在")
		return
	}
	action := parts[3]
	switch action {
	case "enable":
		h.handleEnableGrant(w, r)
	case "disable":
		h.handleDisableGrant(w, r)
	default:
		writeError(w, http.StatusNotFound, CodeNotFound, "路由不存在")
	}
}

// handleProxyRoutes 按方法分派 /api/proxies。
func (h *Handler) handleProxyRoutes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleListProxies(w, r)
	case http.MethodPost:
		h.handleCreateProxy(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, CodeBadRequest, "不支持的请求方法")
	}
}

// handleProxyActionRoutes 按路径分派 /api/proxies/{id}/*。
func (h *Handler) handleProxyActionRoutes(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(r.URL.Path)
	if len(parts) < 4 {
		writeError(w, http.StatusNotFound, CodeNotFound, "路由不存在")
		return
	}
	action := parts[3]
	switch action {
	case "enable":
		h.handleEnableProxy(w, r)
	case "disable":
		h.handleDisableProxy(w, r)
	default:
		writeError(w, http.StatusNotFound, CodeNotFound, "路由不存在")
	}
}

// splitPath 按 "/" 分割路径,去除空段。
func splitPath(path string) []string {
	var parts []string
	start := 0
	for i := 0; i <= len(path); i++ {
		if i == len(path) || path[i] == '/' {
			if i > start {
				parts = append(parts, path[start:i])
			}
			start = i + 1
		}
	}
	return parts
}

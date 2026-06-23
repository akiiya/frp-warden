package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/fengheasia/frp-warden/internal/model"
	"github.com/fengheasia/frp-warden/internal/store"
)

// userSpec 是 NewProxy / Ping 中嵌套的用户信息(frpc 身份)。
type userSpec struct {
	User  string            `json:"user"`
	Metas map[string]string `json:"metas"`
}

func (u userSpec) token() string { return u.Metas["token"] }

// validProxyType 判断请求的 proxy 类型是否为受支持的类型。
func validProxyType(t string) bool {
	switch t {
	case model.ProxyTypeHTTP, model.ProxyTypeHTTPS, model.ProxyTypeTCP, model.ProxyTypeUDP:
		return true
	default:
		return false
	}
}

// ---------------- Login ----------------

// loginContent 是 Login 的请求内容。
type loginContent struct {
	User          string            `json:"user"`
	Metas         map[string]string `json:"metas"`
	RunID         string            `json:"run_id"`
	ClientAddress string            `json:"client_address"`
}

// handleLogin 仅做身份校验:校验 tenant code + token,以及租户是否启用。
// 不在此处判断资源授权(那是 NewProxy 的职责)。
func (h *Handler) handleLogin(ctx context.Context, raw json.RawMessage) pluginResponse {
	var c loginContent
	if err := json.Unmarshal(raw, &c); err != nil {
		return reject("Login 内容无法解析")
	}
	if _, resp, ok := h.authTenant(ctx, c.User, c.Metas["token"]); !ok {
		return resp
	}
	// 审计:Login 成功/失败的审计写入计划在后续阶段实现(见 docs/FRP_PLUGIN.md);
	// 即便实现也绝不写入 token 明文。run_id / client_address 可作为审计上下文。
	return allow()
}

// ---------------- NewProxy ----------------

// newProxyContent 是 NewProxy 的请求内容(按 frp plugin 常见结构兼容解析)。
type newProxyContent struct {
	User          userSpec `json:"user"`
	ProxyName     string   `json:"proxy_name"`
	ProxyType     string   `json:"proxy_type"`
	Subdomain     string   `json:"subdomain"`
	CustomDomains []string `json:"custom_domains"`
	RemotePort    int      `json:"remote_port"`
}

// handleNewProxy 是最关键的资源授权强制点。即使 Login 成功,客户端也不能随意创建 proxy:
// 必须重新校验租户与 token,并强制 proxy 必须在后台登记、类型一致、所用资源已授权给本租户、
// 且请求的 subdomain / remote_port 与后台资源完全一致。客户端无法靠改这些字段越权。
func (h *Handler) handleNewProxy(ctx context.Context, raw json.RawMessage) pluginResponse {
	var c newProxyContent
	if err := json.Unmarshal(raw, &c); err != nil {
		return reject("NewProxy 内容无法解析")
	}

	// 1) 再次校验身份与 token(不信任 Login 的结果)。
	tenant, resp, ok := h.authTenant(ctx, c.User.User, c.User.token())
	if !ok {
		return resp
	}

	// 2) 基础字段校验。
	if strings.TrimSpace(c.ProxyName) == "" {
		return reject("proxy_name 不能为空")
	}
	if !validProxyType(c.ProxyType) {
		return reject("不支持的 proxy_type")
	}
	// 前期一律禁止 custom_domains,避免绕过资源池。
	if len(c.CustomDomains) > 0 {
		return reject("当前不允许使用 custom_domains")
	}

	// 3) 取后台登记的 proxy 及其资源、授权。
	actx, err := h.store.GetProxyAuthContext(ctx, tenant.ID, c.ProxyName)
	if errors.Is(err, store.ErrProxyNotFound) {
		return reject("proxy 不存在或未在后台登记")
	}
	if err != nil {
		return reject("内部错误,已拒绝以保证安全")
	}

	// 4) proxy 自身状态与类型。
	if actx.Proxy.Status != model.StatusEnabled {
		return reject("proxy 已禁用")
	}
	if actx.Proxy.ProxyType != c.ProxyType {
		return reject("proxy 类型与后台登记不一致")
	}

	// 5) 授权校验:资源必须已授权给"本"租户,且授权处于 enabled。
	if !actx.GrantFound || actx.Grant.TenantID != tenant.ID {
		return reject("该资源未授权给此租户")
	}
	if actx.Grant.Status != model.StatusEnabled {
		return reject("资源授权已禁用")
	}

	// 6) 资源状态。
	if actx.Resource.Status != model.ResourceStatusAvailable {
		return reject("资源已禁用")
	}

	// 7) 类型与请求值(subdomain / remote_port)必须与后台资源严格一致。
	return h.checkProxyResourceMatch(c, actx.Resource)
}

// checkProxyResourceMatch 校验请求的 subdomain / remote_port 是否与后台资源一致,
// 以及 proxy 类型与资源类型是否匹配。
func (h *Handler) checkProxyResourceMatch(c newProxyContent, resource model.Resource) pluginResponse {
	switch c.ProxyType {
	case model.ProxyTypeHTTP, model.ProxyTypeHTTPS:
		if resource.Type != model.ResourceTypeSubdomain {
			return reject("proxy 类型与资源类型不匹配")
		}
		if strings.TrimSpace(c.Subdomain) == "" {
			return reject("subdomain 不能为空")
		}
		if c.Subdomain != resource.Value {
			return reject("subdomain 与后台授权资源不一致")
		}
	case model.ProxyTypeTCP:
		if resource.Type != model.ResourceTypeTCPPort {
			return reject("proxy 类型与资源类型不匹配")
		}
		if c.RemotePort <= 0 {
			return reject("remote_port 非法")
		}
		if strconv.Itoa(c.RemotePort) != resource.Value {
			return reject("remote_port 与后台授权资源不一致")
		}
	case model.ProxyTypeUDP:
		if resource.Type != model.ResourceTypeUDPPort {
			return reject("proxy 类型与资源类型不匹配")
		}
		if c.RemotePort <= 0 {
			return reject("remote_port 非法")
		}
		if strconv.Itoa(c.RemotePort) != resource.Value {
			return reject("remote_port 与后台授权资源不一致")
		}
	default:
		return reject("不支持的 proxy_type")
	}
	return allow()
}

// ---------------- Ping ----------------

// pingContent 是 Ping 的请求内容。
type pingContent struct {
	User      userSpec `json:"user"`
	Timestamp int64    `json:"timestamp"`
}

// handlePing 刷新租户在线状态:校验身份与 token,更新 last_seen_at;
// 租户被禁用或 token 错误则拒绝,使禁用能尽快让已连接客户端失效。
func (h *Handler) handlePing(ctx context.Context, raw json.RawMessage) pluginResponse {
	var c pingContent
	if err := json.Unmarshal(raw, &c); err != nil {
		return reject("Ping 内容无法解析")
	}
	tenant, resp, ok := h.authTenant(ctx, c.User.User, c.User.token())
	if !ok {
		return resp
	}
	if err := h.store.UpdateTenantLastSeen(ctx, tenant.ID); err != nil {
		return reject("内部错误,已拒绝以保证安全")
	}
	return allow()
}

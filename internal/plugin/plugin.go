// Package plugin 实现 frps server plugin 的 HTTP hooks:Login / NewProxy / Ping。
//
// frps 通过 HTTP 调用这些 hooks(见 docs/FRP_PLUGIN.md)。这里是 frp-warden 的
// 核心鉴权强制点:frpc 客户端配置不可信,因此每次 Login / NewProxy / Ping 都必须
// 在此处对照数据库进行校验,客户端不能靠修改 subdomain / remote_port / custom_domains 越权。
//
// 默认 fail-closed:任何解析失败、未知 op、内部错误都不放行;业务拒绝返回
// HTTP 200 + {"reject": true, "reject_reason": "..."},以兼容 frps plugin 行为。
//
// 安全:reject_reason 以中文为主,且绝不包含 token / hash / session_secret 等敏感信息;
// 日志与审计也不输出 token 明文。
package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/fengheasia/frp-warden/internal/model"
	"github.com/fengheasia/frp-warden/internal/security"
	"github.com/fengheasia/frp-warden/internal/store"
)

// defaultMaxBodyBytes 限制 frps plugin 请求体大小,防止异常的超大请求。
const defaultMaxBodyBytes int64 = 1 << 20 // 1MB

// Handler 提供 frps plugin 接口(默认路径 /plugin/frp)。
type Handler struct {
	store   *store.Store
	maxBody int64
}

// NewHandler 用给定的 store 构造 plugin 处理器。
func NewHandler(st *store.Store) *Handler {
	return &Handler{store: st, maxBody: defaultMaxBodyBytes}
}

// pluginResponse 是 frps server plugin 的响应信封,字段命名与 frp plugin 协议兼容。
type pluginResponse struct {
	Reject       bool   `json:"reject"`
	RejectReason string `json:"reject_reason,omitempty"`
	Unchange     bool   `json:"unchange"`
}

// allow 返回"放行且不改写"的响应。
func allow() pluginResponse { return pluginResponse{Reject: false, Unchange: true} }

// reject 返回带中文原因的"拒绝"响应。reason 不得包含敏感信息。
func reject(reason string) pluginResponse { return pluginResponse{Reject: true, RejectReason: reason} }

// envelope 是 frps 请求的外层信封,content 延迟到各 op 再解析。
type envelope struct {
	Content json.RawMessage `json:"content"`
}

// ServeHTTP 解析并分派 frps plugin 请求。op 通过 query 参数传入。
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "仅支持 POST", http.StatusMethodNotAllowed)
		return
	}

	op := r.URL.Query().Get("op")

	// 限制请求体大小;超限时 MaxBytesReader 会在读取时报错。
	r.Body = http.MaxBytesReader(w, r.Body, h.maxBody)
	var env envelope
	if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			http.Error(w, "请求体过大", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "请求 JSON 无法解析", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	var resp pluginResponse
	switch op {
	case "Login":
		resp = h.handleLogin(ctx, env.Content)
	case "NewProxy":
		resp = h.handleNewProxy(ctx, env.Content)
	case "Ping":
		resp = h.handlePing(ctx, env.Content)
	default:
		// 未知 op:fail-closed,返回 200 + reject。
		resp = reject("不支持的插件操作")
	}
	writeJSON(w, resp)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// authTenant 是 Login/NewProxy/Ping 共用的身份校验:
// 校验 code 与 token,返回租户;校验不通过时返回对应的 reject 响应,ok=false。
//
// 为避免泄露"是用户不存在还是 token 错误",两者返回相同的模糊提示;
// 但租户被禁用会明确告知,以便运维定位。绝不在原因中包含 token 或 hash。
func (h *Handler) authTenant(ctx context.Context, code, token string) (model.Tenant, pluginResponse, bool) {
	if strings.TrimSpace(code) == "" {
		return model.Tenant{}, reject("缺少 user(tenant code)"), false
	}
	if token == "" {
		return model.Tenant{}, reject("缺少 token"), false
	}

	t, err := h.store.GetTenantByCode(ctx, code)
	if errors.Is(err, store.ErrTenantNotFound) {
		return model.Tenant{}, reject("租户不存在或凭据无效"), false
	}
	if err != nil {
		return model.Tenant{}, reject("内部错误,已拒绝以保证安全"), false
	}
	if t.Status != model.StatusEnabled {
		return model.Tenant{}, reject("租户已禁用"), false
	}
	if !security.VerifySecret(t.TokenHash, token) {
		return model.Tenant{}, reject("租户不存在或凭据无效"), false
	}
	return t, pluginResponse{}, true
}

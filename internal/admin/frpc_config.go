package admin

import (
	"context"
	"errors"
	"net/http"

	"github.com/fengheasia/frp-warden/internal/frpcconfig"
	"github.com/fengheasia/frp-warden/internal/store"
)

// handleGetFrpcConfig 处理 GET /api/tenants/{id}/frpc-config。
// 返回 token 占位符模板(不含真实 token)。
func (h *Handler) handleGetFrpcConfig(w http.ResponseWriter, r *http.Request) {
	tenantID, err := parseIDFromPath(r, 2)
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeBadRequest, "无效的 tenant id")
		return
	}

	t, err := h.store.GetTenantByID(r.Context(), tenantID)
	if errors.Is(err, store.ErrTenantNotFound) {
		writeError(w, http.StatusNotFound, CodeNotFound, "租户不存在")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, CodeInternal, "内部错误")
		return
	}

	cfg, err := generateFrpcConfigForTenant(r.Context(), h.store, t.Code, t.ID,
		"", true, h.cfg.FRP.ServerAddr, h.cfg.FRP.ServerPort)
	if err != nil {
		writeError(w, http.StatusInternalServerError, CodeInternal, "生成配置失败")
		return
	}

	writeOK(w, map[string]any{
		"tenant_id":         t.ID,
		"tenant_code":       t.Code,
		"mode":              "template",
		"token_placeholder": true,
		"config":            cfg,
	})
}

// handleGetFrpcConfigDownload 处理 GET /api/tenants/{id}/frpc-config/download。
// 返回 token 占位符模板的 TOML 文件下载。
func (h *Handler) handleGetFrpcConfigDownload(w http.ResponseWriter, r *http.Request) {
	tenantID, err := parseIDFromPath(r, 2)
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeBadRequest, "无效的 tenant id")
		return
	}

	t, err := h.store.GetTenantByID(r.Context(), tenantID)
	if errors.Is(err, store.ErrTenantNotFound) {
		writeError(w, http.StatusNotFound, CodeNotFound, "租户不存在")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, CodeInternal, "内部错误")
		return
	}

	cfg, err := generateFrpcConfigForTenant(r.Context(), h.store, t.Code, t.ID,
		"", true, h.cfg.FRP.ServerAddr, h.cfg.FRP.ServerPort)
	if err != nil {
		writeError(w, http.StatusInternalServerError, CodeInternal, "生成配置失败")
		return
	}

	filename := "frpc-" + t.Code + ".toml"
	w.Header().Set("Content-Type", "application/toml; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Write([]byte(cfg))
}

// buildFrpcConfigWithToken 为创建/重置 token 时生成含真实 token 的完整配置。
// plainToken 由调用方提供(创建/重置瞬间仍在内存中)。
// 安全:plainToken 绝不写入审计日志或持久化。
func (h *Handler) buildFrpcConfigWithToken(tenantID int64, tenantCode string, plainToken string) string {
	ctx := context.Background()
	cfg, err := generateFrpcConfigForTenant(ctx, h.store, tenantCode, tenantID,
		plainToken, false, h.cfg.FRP.ServerAddr, h.cfg.FRP.ServerPort)
	if err != nil {
		return ""
	}
	return cfg
}

// generateFrpcConfigForTenant 根据 tenant 与 proxy/resource/grant 数据生成 frpc.toml。
//
// 安全原则:
//   - tokenIsPlaceholder=true 时使用占位符,不含真实 token。
//   - tokenIsPlaceholder=false 时使用传入的 plainToken(仅在创建/重置瞬间调用)。
//   - 调用方必须确保 plainToken 不被写入审计日志或持久化。
func generateFrpcConfigForTenant(ctx context.Context, st *store.Store,
	tenantCode string, tenantID int64, plainToken string, tokenIsPlaceholder bool,
	serverAddr string, serverPort int) (string, error) {

	token := plainToken
	if tokenIsPlaceholder {
		token = "请粘贴创建/重置时显示的一次性 token"
	}

	// 获取该 tenant 的所有 enabled proxy 及其 resource/grant。
	proxyCtxs, err := st.ListEnabledProxyAuthContextsByTenant(ctx, tenantID)
	if err != nil {
		return "", err
	}

	var proxies []frpcconfig.ProxyEntry
	for _, pc := range proxyCtxs {
		entry := frpcconfig.ProxyEntry{
			Name:      pc.Proxy.Name,
			Type:      pc.Proxy.ProxyType,
			LocalIP:   pc.Proxy.LocalIP,
			LocalPort: pc.Proxy.LocalPort,
		}

		switch pc.Proxy.ProxyType {
		case "http", "https":
			entry.Subdomain = pc.Resource.Value
		case "tcp", "udp":
			port, err := frpcconfig.ParsePort(pc.Resource.Value)
			if err != nil {
				return "", err
			}
			entry.RemotePort = port
		}

		proxies = append(proxies, entry)
	}

	cfg := frpcconfig.Config{
		ServerAddr: serverAddr,
		ServerPort: serverPort,
		User:       tenantCode,
		Token:      token,
		Proxies:    proxies,
	}

	return frpcconfig.Generate(cfg)
}

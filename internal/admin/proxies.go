package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/fengheasia/frp-warden/internal/model"
	"github.com/fengheasia/frp-warden/internal/store"
)

// handleListProxies 处理 GET /api/proxies?tenant_id=1。要求登录。
func (h *Handler) handleListProxies(w http.ResponseWriter, r *http.Request) {
	tidStr := r.URL.Query().Get("tenant_id")
	if tidStr != "" {
		tid, err := strconv.ParseInt(tidStr, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, CodeBadRequest, "无效的 tenant_id")
			return
		}
		proxies, err := h.store.ListProxiesByTenant(r.Context(), tid)
		if err != nil {
			writeError(w, http.StatusInternalServerError, CodeInternal, "内部错误")
			return
		}
		out := make([]map[string]any, 0, len(proxies))
		for _, p := range proxies {
			out = append(out, proxyToJSON(p))
		}
		writeOK(w, out)
		return
	}

	proxies, err := h.store.ListProxies(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, CodeInternal, "内部错误")
		return
	}
	out := make([]map[string]any, 0, len(proxies))
	for _, p := range proxies {
		out = append(out, proxyToJSON(p))
	}
	writeOK(w, out)
}

// handleCreateProxy 处理 POST /api/proxies。要求登录。
func (h *Handler) handleCreateProxy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID   int64  `json:"tenant_id"`
		ResourceID int64  `json:"resource_id"`
		Name       string `json:"name"`
		ProxyType  string `json:"proxy_type"`
		LocalIP    string `json:"local_ip"`
		LocalPort  int    `json:"local_port"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, CodeBadRequest, "请求格式错误")
		return
	}

	p, err := h.store.CreateProxy(r.Context(), store.ProxyInput{
		TenantID:   req.TenantID,
		ResourceID: req.ResourceID,
		Name:       strings.TrimSpace(req.Name),
		ProxyType:  strings.TrimSpace(req.ProxyType),
		LocalIP:    strings.TrimSpace(req.LocalIP),
		LocalPort:  req.LocalPort,
	})
	if errors.Is(err, store.ErrInvalidProxyType) {
		writeError(w, http.StatusBadRequest, CodeBadRequest, "非法的 proxy 类型")
		return
	}
	if errors.Is(err, store.ErrInvalidPort) {
		writeError(w, http.StatusBadRequest, CodeBadRequest, "端口范围非法（应在 1-65535）")
		return
	}
	if errors.Is(err, store.ErrResourceNotGrantedToTenant) {
		writeError(w, http.StatusConflict, CodeConflict, "该资源未授权给此租户")
		return
	}
	if errors.Is(err, store.ErrProxyTypeResourceMismatch) {
		writeError(w, http.StatusConflict, CodeConflict, "proxy 类型与资源类型不匹配")
		return
	}
	if errors.Is(err, store.ErrDuplicateProxyName) {
		writeError(w, http.StatusConflict, CodeConflict, "同一租户下已存在同名 proxy")
		return
	}
	if errors.Is(err, store.ErrEmptyField) {
		writeError(w, http.StatusBadRequest, CodeBadRequest, "必填字段不能为空")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, CodeInternal, "内部错误")
		return
	}

	h.writeAudit(r, getSessionID(r.Context()), "proxy.created", "proxy", i64s(p.ID), "创建 proxy "+p.Name)
	writeOK(w, proxyToJSON(p))
}

// handleEnableProxy 处理 POST /api/proxies/{id}/enable。要求登录。
func (h *Handler) handleEnableProxy(w http.ResponseWriter, r *http.Request) {
	h.updateProxyStatus(w, r, model.StatusEnabled)
}

// handleDisableProxy 处理 POST /api/proxies/{id}/disable。要求登录。
func (h *Handler) handleDisableProxy(w http.ResponseWriter, r *http.Request) {
	h.updateProxyStatus(w, r, model.StatusDisabled)
}

func (h *Handler) updateProxyStatus(w http.ResponseWriter, r *http.Request, status string) {
	id, err := parseIDFromPath(r, 2)
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeBadRequest, "无效的 id")
		return
	}
	if err := h.store.UpdateProxyStatusByID(r.Context(), id, status); err != nil {
		if errors.Is(err, store.ErrProxyNotFound) {
			writeError(w, http.StatusNotFound, CodeNotFound, "proxy 不存在")
			return
		}
		writeError(w, http.StatusInternalServerError, CodeInternal, "内部错误")
		return
	}

	action := "proxy.disabled"
	if status == model.StatusEnabled {
		action = "proxy.enabled"
	}
	h.writeAudit(r, getSessionID(r.Context()), action, "proxy", i64s(id), action)
	writeOK(w, nil)
}

func proxyToJSON(p model.Proxy) map[string]any {
	return map[string]any{
		"id":          p.ID,
		"tenant_id":   p.TenantID,
		"resource_id": p.ResourceID,
		"name":        p.Name,
		"proxy_type":  p.ProxyType,
		"local_ip":    p.LocalIP,
		"local_port":  p.LocalPort,
		"status":      p.Status,
	}
}

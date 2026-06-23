package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/fengheasia/frp-warden/internal/model"
	"github.com/fengheasia/frp-warden/internal/security"
	"github.com/fengheasia/frp-warden/internal/store"
)

// handleListTenants 处理 GET /api/tenants。要求登录。
func (h *Handler) handleListTenants(w http.ResponseWriter, r *http.Request) {
	tenants, err := h.store.ListTenants(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, CodeInternal, "内部错误")
		return
	}
	// 不返回 token_hash。
	out := make([]map[string]any, 0, len(tenants))
	for _, t := range tenants {
		out = append(out, tenantToJSON(t))
	}
	writeOK(w, out)
}

// handleCreateTenant 处理 POST /api/tenants。要求登录。
// 创建时自动生成随机 token,明文只在本次响应返回,数据库只存 hash。
func (h *Handler) handleCreateTenant(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code        string `json:"code"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, CodeBadRequest, "请求格式错误")
		return
	}

	// 生成随机 tenant token。
	plainToken, err := security.GenerateRandomPassword(32)
	if err != nil {
		writeError(w, http.StatusInternalServerError, CodeInternal, "内部错误")
		return
	}
	tokenHash, err := security.HashSecret(plainToken)
	if err != nil {
		writeError(w, http.StatusInternalServerError, CodeInternal, "内部错误")
		return
	}

	t, err := h.store.CreateTenant(r.Context(), store.TenantInput{
		Code:        strings.TrimSpace(req.Code),
		Name:        strings.TrimSpace(req.Name),
		TokenHash:   tokenHash,
		Description: req.Description,
	})
	if errors.Is(err, store.ErrEmptyField) {
		writeError(w, http.StatusBadRequest, CodeBadRequest, "code 和 name 不能为空")
		return
	}
	if errors.Is(err, store.ErrDuplicateCode) {
		writeError(w, http.StatusConflict, CodeConflict, "租户 code 已存在")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, CodeInternal, "内部错误")
		return
	}

	h.writeAudit(r, getSessionID(r.Context()), "tenant.created", "tenant", i64s(t.ID), "创建租户 "+t.Code)

	// 生成含真实 token 的完整 frpc 配置(仅在此刻,plain_token 仍在内存中)。
	frpcConfig := h.buildFrpcConfigWithToken(t.ID, t.Code, plainToken)

	writeOK(w, map[string]any{
		"tenant":      tenantToJSON(t),
		"plain_token": plainToken,
		"frpc_config": frpcConfig,
	})
}

// handleDisableTenant 处理 POST /api/tenants/{id}/disable。要求登录。
func (h *Handler) handleDisableTenant(w http.ResponseWriter, r *http.Request) {
	h.updateTenantStatus(w, r, model.StatusDisabled)
}

// handleEnableTenant 处理 POST /api/tenants/{id}/enable。要求登录。
func (h *Handler) handleEnableTenant(w http.ResponseWriter, r *http.Request) {
	h.updateTenantStatus(w, r, model.StatusEnabled)
}

func (h *Handler) updateTenantStatus(w http.ResponseWriter, r *http.Request, status string) {
	id, err := parseIDFromPath(r, 2) // /api/tenants/{id}/enable → 第 3 段
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeBadRequest, "无效的 id")
		return
	}
	if err := h.store.UpdateTenantStatus(r.Context(), id, status); err != nil {
		if errors.Is(err, store.ErrTenantNotFound) {
			writeError(w, http.StatusNotFound, CodeNotFound, "租户不存在")
			return
		}
		writeError(w, http.StatusInternalServerError, CodeInternal, "内部错误")
		return
	}

	action := "tenant.disabled"
	if status == model.StatusEnabled {
		action = "tenant.enabled"
	}
	h.writeAudit(r, getSessionID(r.Context()), action, "tenant", i64s(id), action)
	writeOK(w, nil)
}

// handleResetTenantToken 处理 POST /api/tenants/{id}/reset-token。要求登录。
func (h *Handler) handleResetTenantToken(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r, 2)
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeBadRequest, "无效的 id")
		return
	}

	plainToken, err := security.GenerateRandomPassword(32)
	if err != nil {
		writeError(w, http.StatusInternalServerError, CodeInternal, "内部错误")
		return
	}
	tokenHash, err := security.HashSecret(plainToken)
	if err != nil {
		writeError(w, http.StatusInternalServerError, CodeInternal, "内部错误")
		return
	}
	if err := h.store.UpdateTenantTokenHash(r.Context(), id, tokenHash); err != nil {
		if errors.Is(err, store.ErrTenantNotFound) {
			writeError(w, http.StatusNotFound, CodeNotFound, "租户不存在")
			return
		}
		writeError(w, http.StatusInternalServerError, CodeInternal, "内部错误")
		return
	}

	h.writeAudit(r, getSessionID(r.Context()), "tenant.token.reset", "tenant", i64s(id), "重置租户 token")

	// 生成含新 token 的完整 frpc 配置(仅在此刻,plain_token 仍在内存中)。
	tenant, _ := h.store.GetTenantByID(r.Context(), id)
	frpcConfig := h.buildFrpcConfigWithToken(id, tenant.Code, plainToken)

	writeOK(w, map[string]any{
		"plain_token": plainToken,
		"frpc_config": frpcConfig,
	})
}

// tenantToJSON 把 tenant 转为不含 token_hash 的 JSON map。
func tenantToJSON(t model.Tenant) map[string]any {
	return map[string]any{
		"id":           t.ID,
		"code":         t.Code,
		"name":         t.Name,
		"status":       t.Status,
		"description":  t.Description,
		"last_seen_at": t.LastSeenAt,
	}
}

// parseIDFromPath 从 URL 路径中按 "/" 分割后取第 segment 段并解析为 int64。
// 例如 /api/tenants/123/enable,segment=3 → 123。
func parseIDFromPath(r *http.Request, segment int) (int64, error) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if segment >= len(parts) {
		return 0, strconv.ErrSyntax
	}
	return strconv.ParseInt(parts[segment], 10, 64)
}

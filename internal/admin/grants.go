package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/fengheasia/frp-warden/internal/model"
	"github.com/fengheasia/frp-warden/internal/store"
)

// handleListGrants 处理 GET /api/grants?tenant_id=1。要求登录。
func (h *Handler) handleListGrants(w http.ResponseWriter, r *http.Request) {
	tidStr := r.URL.Query().Get("tenant_id")
	if tidStr != "" {
		tid, err := strconv.ParseInt(tidStr, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, CodeBadRequest, "无效的 tenant_id")
			return
		}
		grants, err := h.store.ListGrantsByTenant(r.Context(), tid)
		if err != nil {
			writeError(w, http.StatusInternalServerError, CodeInternal, "内部错误")
			return
		}
		out := make([]map[string]any, 0, len(grants))
		for _, g := range grants {
			out = append(out, grantToJSON(g))
		}
		writeOK(w, out)
		return
	}

	grants, err := h.store.ListGrants(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, CodeInternal, "内部错误")
		return
	}
	out := make([]map[string]any, 0, len(grants))
	for _, g := range grants {
		out = append(out, grantToJSON(g))
	}
	writeOK(w, out)
}

// handleCreateGrant 处理 POST /api/grants。要求登录。
func (h *Handler) handleCreateGrant(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID   int64 `json:"tenant_id"`
		ResourceID int64 `json:"resource_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, CodeBadRequest, "请求格式错误")
		return
	}

	g, err := h.store.GrantResourceToTenant(r.Context(), req.TenantID, req.ResourceID)
	if errors.Is(err, store.ErrTenantNotFound) {
		writeError(w, http.StatusNotFound, CodeNotFound, "租户不存在")
		return
	}
	if errors.Is(err, store.ErrTenantDisabled) {
		writeError(w, http.StatusForbidden, CodeForbidden, "租户已禁用")
		return
	}
	if errors.Is(err, store.ErrResourceNotFound) {
		writeError(w, http.StatusNotFound, CodeNotFound, "资源不存在")
		return
	}
	if errors.Is(err, store.ErrResourceDisabled) {
		writeError(w, http.StatusForbidden, CodeForbidden, "资源已禁用")
		return
	}
	if errors.Is(err, store.ErrResourceAlreadyGranted) {
		writeError(w, http.StatusConflict, CodeConflict, "该资源已被授权给某个租户，不能重复授权")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, CodeInternal, "内部错误")
		return
	}

	h.writeAudit(r, getSessionID(r.Context()), "grant.created", "grant", i64s(g.ID), "授权资源给租户")
	writeOK(w, grantToJSON(g))
}

// handleEnableGrant 处理 POST /api/grants/{id}/enable。要求登录。
func (h *Handler) handleEnableGrant(w http.ResponseWriter, r *http.Request) {
	h.updateGrantStatus(w, r, model.StatusEnabled)
}

// handleDisableGrant 处理 POST /api/grants/{id}/disable。要求登录。
func (h *Handler) handleDisableGrant(w http.ResponseWriter, r *http.Request) {
	h.updateGrantStatus(w, r, model.StatusDisabled)
}

func (h *Handler) updateGrantStatus(w http.ResponseWriter, r *http.Request, status string) {
	id, err := parseIDFromPath(r, 2)
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeBadRequest, "无效的 id")
		return
	}
	if err := h.store.UpdateGrantStatusByID(r.Context(), id, status); err != nil {
		if errors.Is(err, store.ErrGrantNotFound) {
			writeError(w, http.StatusNotFound, CodeNotFound, "授权记录不存在")
			return
		}
		writeError(w, http.StatusInternalServerError, CodeInternal, "内部错误")
		return
	}

	action := "grant.disabled"
	if status == model.StatusEnabled {
		action = "grant.enabled"
	}
	h.writeAudit(r, getSessionID(r.Context()), action, "grant", i64s(id), action)
	writeOK(w, nil)
}

func grantToJSON(g model.ResourceGrant) map[string]any {
	return map[string]any{
		"id":          g.ID,
		"tenant_id":   g.TenantID,
		"resource_id": g.ResourceID,
		"status":      g.Status,
	}
}

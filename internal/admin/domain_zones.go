package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/fengheasia/frp-warden/internal/model"
	"github.com/fengheasia/frp-warden/internal/store"
)

// handleListDomainZones 处理 GET /api/domain-zones。要求登录。
func (h *Handler) handleListDomainZones(w http.ResponseWriter, r *http.Request) {
	zones, err := h.store.ListDomainZones(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, CodeInternal, "内部错误")
		return
	}
	out := make([]map[string]any, 0, len(zones))
	for _, z := range zones {
		out = append(out, domainZoneToJSON(z))
	}
	writeOK(w, out)
}

// handleCreateDomainZone 处理 POST /api/domain-zones。要求登录。
func (h *Handler) handleCreateDomainZone(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Zone string `json:"zone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, CodeBadRequest, "请求格式错误")
		return
	}

	z, err := h.store.CreateDomainZone(r.Context(), strings.TrimSpace(req.Name), strings.TrimSpace(req.Zone))
	if errors.Is(err, store.ErrEmptyField) {
		writeError(w, http.StatusBadRequest, CodeBadRequest, "name 和 zone 不能为空")
		return
	}
	if errors.Is(err, store.ErrDuplicateZone) {
		writeError(w, http.StatusConflict, CodeConflict, "顶级域名区域 zone 已存在")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, CodeInternal, "内部错误")
		return
	}

	h.writeAudit(r, getSessionID(r.Context()), "domain_zone.created", "domain_zone", i64s(z.ID), "创建顶级域名区域 "+z.Zone)
	writeOK(w, domainZoneToJSON(z))
}

// handleEnableDomainZone 处理 POST /api/domain-zones/{id}/enable。要求登录。
func (h *Handler) handleEnableDomainZone(w http.ResponseWriter, r *http.Request) {
	h.updateDomainZoneStatus(w, r, model.StatusEnabled)
}

// handleDisableDomainZone 处理 POST /api/domain-zones/{id}/disable。要求登录。
func (h *Handler) handleDisableDomainZone(w http.ResponseWriter, r *http.Request) {
	h.updateDomainZoneStatus(w, r, model.StatusDisabled)
}

func (h *Handler) updateDomainZoneStatus(w http.ResponseWriter, r *http.Request, status string) {
	id, err := parseIDFromPath(r, 2)
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeBadRequest, "无效的 id")
		return
	}
	if err := h.store.UpdateDomainZoneStatus(r.Context(), id, status); err != nil {
		if errors.Is(err, store.ErrDomainZoneNotFound) {
			writeError(w, http.StatusNotFound, CodeNotFound, "顶级域名区域不存在")
			return
		}
		writeError(w, http.StatusInternalServerError, CodeInternal, "内部错误")
		return
	}

	action := "domain_zone.disabled"
	if status == model.StatusEnabled {
		action = "domain_zone.enabled"
	}
	h.writeAudit(r, getSessionID(r.Context()), action, "domain_zone", i64s(id), action)
	writeOK(w, nil)
}

func domainZoneToJSON(z model.DomainZone) map[string]any {
	return map[string]any{
		"id":     z.ID,
		"name":   z.Name,
		"zone":   z.Zone,
		"status": z.Status,
	}
}

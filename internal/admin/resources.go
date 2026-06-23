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

// handleListResources 处理 GET /api/resources。要求登录。
func (h *Handler) handleListResources(w http.ResponseWriter, r *http.Request) {
	resources, err := h.store.ListResources(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, CodeInternal, "内部错误")
		return
	}
	out := make([]map[string]any, 0, len(resources))
	for _, res := range resources {
		out = append(out, resourceToJSON(res))
	}
	writeOK(w, out)
}

// handleCreateSubdomainResource 处理 POST /api/resources/subdomain。要求登录。
func (h *Handler) handleCreateSubdomainResource(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DomainZoneID int64  `json:"domain_zone_id"`
		Value        string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, CodeBadRequest, "请求格式错误")
		return
	}

	res, err := h.store.CreateSubdomainResource(r.Context(), strings.TrimSpace(req.Value), req.DomainZoneID)
	if errors.Is(err, store.ErrNoEnabledDomainZone) {
		writeError(w, http.StatusBadRequest, CodeBadRequest, "没有可用的顶级域名区域")
		return
	}
	if errors.Is(err, store.ErrSubdomainNeedsZone) {
		writeError(w, http.StatusBadRequest, CodeBadRequest, "subdomain 资源必须绑定一个 enabled 的顶级域名区域")
		return
	}
	if errors.Is(err, store.ErrDomainZoneDisabled) {
		writeError(w, http.StatusBadRequest, CodeBadRequest, "顶级域名区域已禁用")
		return
	}
	if errors.Is(err, store.ErrDuplicateResource) {
		writeError(w, http.StatusConflict, CodeConflict, "相同 type 与 value 的资源已存在")
		return
	}
	if errors.Is(err, store.ErrEmptyField) {
		writeError(w, http.StatusBadRequest, CodeBadRequest, "value 不能为空")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, CodeInternal, "内部错误")
		return
	}

	h.writeAudit(r, getSessionID(r.Context()), "resource.created", "resource", i64s(res.ID), "创建 subdomain 资源 "+res.Value)
	writeOK(w, resourceToJSON(res))
}

// handleCreateTCPPortResource 处理 POST /api/resources/tcp-port。要求登录。
func (h *Handler) handleCreateTCPPortResource(w http.ResponseWriter, r *http.Request) {
	h.createPortResource(w, r, model.ResourceTypeTCPPort)
}

// handleCreateUDPPortResource 处理 POST /api/resources/udp-port。要求登录。
func (h *Handler) handleCreateUDPPortResource(w http.ResponseWriter, r *http.Request) {
	h.createPortResource(w, r, model.ResourceTypeUDPPort)
}

func (h *Handler) createPortResource(w http.ResponseWriter, r *http.Request, resType string) {
	var req struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, CodeBadRequest, "请求格式错误")
		return
	}

	port, err := strconv.Atoi(strings.TrimSpace(req.Value))
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeBadRequest, "端口必须为数字")
		return
	}

	var res model.Resource
	switch resType {
	case model.ResourceTypeTCPPort:
		res, err = h.store.CreateTCPPortResource(r.Context(), port)
	case model.ResourceTypeUDPPort:
		res, err = h.store.CreateUDPPortResource(r.Context(), port)
	}

	if errors.Is(err, store.ErrInvalidPort) {
		writeError(w, http.StatusBadRequest, CodeBadRequest, "端口范围非法（应在 1-65535）")
		return
	}
	if errors.Is(err, store.ErrDuplicateResource) {
		writeError(w, http.StatusConflict, CodeConflict, "相同 type 与 value 的资源已存在")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, CodeInternal, "内部错误")
		return
	}

	h.writeAudit(r, getSessionID(r.Context()), "resource.created", "resource", i64s(res.ID),
		"创建 "+resType+" 资源 "+res.Value)
	writeOK(w, resourceToJSON(res))
}

func resourceToJSON(r model.Resource) map[string]any {
	m := map[string]any{
		"id":     r.ID,
		"type":   r.Type,
		"value":  r.Value,
		"status": r.Status,
	}
	if r.DomainZoneID != nil {
		m["domain_zone_id"] = *r.DomainZoneID
	}
	return m
}

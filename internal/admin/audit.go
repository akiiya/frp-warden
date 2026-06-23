package admin

import (
	"fmt"
	"net/http"

	"github.com/fengheasia/frp-warden/internal/model"
)

// handleListAuditLogs 处理 GET /api/audit-logs。要求登录。
// 返回最近 100 条审计日志,按 created_at 降序。
func (h *Handler) handleListAuditLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := h.store.ListAuditLogs(r.Context(), 100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, CodeInternal, "内部错误")
		return
	}
	out := make([]map[string]any, 0, len(logs))
	for _, e := range logs {
		out = append(out, auditLogToJSON(e))
	}
	writeOK(w, out)
}

// writeAudit 写入审计日志。message 中文为主,绝不包含密码/token/hash。
// 该方法在其它 handler 中调用,写入失败不阻塞主流程(仅记录到 store)。
// targetID 为字符串形式的目标 id(如 "123");登录失败等无明确目标时传空串。
func (h *Handler) writeAudit(r *http.Request, sessionID, action, targetType, targetID, message string) {
	var actorID *int64
	aid := getAdminID(r.Context())
	if aid > 0 {
		actorID = &aid
	}
	_ = h.store.InsertAuditLog(r.Context(), model.AuditLog{
		ActorType:  "admin",
		ActorID:    actorID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Message:    message,
		IP:         r.RemoteAddr,
	})
}

func auditLogToJSON(e model.AuditLog) map[string]any {
	return map[string]any{
		"id":          e.ID,
		"actor_type":  e.ActorType,
		"actor_id":    e.ActorID,
		"action":      e.Action,
		"target_type": e.TargetType,
		"target_id":   e.TargetID,
		"message":     e.Message,
		"ip":          e.IP,
		"created_at":  e.CreatedAt,
	}
}

// i64s 把 int64 转为字符串,用于审计日志的 targetID 参数。
func i64s(n int64) string { return fmt.Sprintf("%d", n) }

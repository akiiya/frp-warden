package admin

import (
	"context"
	"net/http"
)

// contextKey 是 context 值的键类型,避免与其他包冲突。
type contextKey string

const (
	// ctxAdminID 是已认证管理员的 id。
	ctxAdminID contextKey = "admin_id"
	// ctxSessionID 是当前 session id(用于 logout 撤销)。
	ctxSessionID contextKey = "session_id"
)

// sessionCookieName 是 session cookie 名称。
const sessionCookieName = "fw_session"

// requireAuth 返回一个中间件,校验 session cookie 并把 adminID 注入 context。
// 未认证返回 401;认证通过后调用 next。
func (h *Handler) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || cookie.Value == "" {
			writeError(w, http.StatusUnauthorized, CodeUnauthorized, "请先登录")
			return
		}

		tokenHash := hashSessionToken(cookie.Value)
		sess, err := h.store.GetSessionByTokenHash(r.Context(), tokenHash)
		if err != nil {
			// session 不存在或已过期/撤销。
			writeError(w, http.StatusUnauthorized, CodeUnauthorized, "请先登录")
			return
		}

		// 把 adminID 和 sessionID 注入 context,供后续 handler 使用。
		ctx := context.WithValue(r.Context(), ctxAdminID, sess.AdminID)
		ctx = context.WithValue(ctx, ctxSessionID, sess.ID)
		next(w, r.WithContext(ctx))
	}
}

// getAdminID 从 context 取已认证的管理员 id。
func getAdminID(ctx context.Context) int64 {
	v, _ := ctx.Value(ctxAdminID).(int64)
	return v
}

// getSessionID 从 context 取当前 session id。
func getSessionID(ctx context.Context) string {
	v, _ := ctx.Value(ctxSessionID).(string)
	return v
}

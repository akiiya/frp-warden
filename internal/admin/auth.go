package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/fengheasia/frp-warden/internal/model"
	"github.com/fengheasia/frp-warden/internal/security"
	"github.com/fengheasia/frp-warden/internal/store"
)

// minPasswordLength 是新密码的最小长度。
const minPasswordLength = 10

// hashSessionToken 计算 session token 的 SHA-256 哈希(委托给 security 包)。
func hashSessionToken(token string) string {
	return security.HashSHA256(token)
}

// handleLogin 处理 POST /api/auth/login。
func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, CodeBadRequest, "请求格式错误")
		return
	}

	// 查管理员。
	admin, err := h.store.GetAdminByUsername(r.Context(), strings.TrimSpace(req.Username))
	if errors.Is(err, store.ErrAdminNotFound) {
		// 统一提示,不泄露"用户名是否存在"。
		h.writeAudit(r, "", "admin.login.failed", "", "", "用户名或密码错误")
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "用户名或密码错误")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, CodeInternal, "内部错误")
		return
	}

	// 检查状态。
	if admin.Status != model.StatusEnabled {
		h.writeAudit(r, "", "admin.login.failed", "", "", "账号已禁用")
		writeError(w, http.StatusForbidden, CodeForbidden, "账号已禁用")
		return
	}

	// 校验密码。
	if !security.VerifyPassword(admin.PasswordHash, req.Password) {
		h.writeAudit(r, "", "admin.login.failed", "", "", "用户名或密码错误")
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "用户名或密码错误")
		return
	}

	// 创建 session。
	token, err := store.GenerateSessionToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, CodeInternal, "内部错误")
		return
	}
	tokenHash := hashSessionToken(token)
	expiresAt := time.Now().Add(store.SessionTTL)
	sessionID, err := h.store.CreateSession(r.Context(), admin.ID, tokenHash, expiresAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, CodeInternal, "内部错误")
		return
	}

	// 更新 last_login_at。
	_ = h.store.UpdateAdminLastLogin(r.Context(), admin.ID)

	// 设置 cookie。
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
	})

	h.writeAudit(r, sessionID, "admin.login.success", "admin", i64s(admin.ID), "管理员登录成功")

	writeOK(w, map[string]any{
		"admin": map[string]any{
			"id":                   admin.ID,
			"username":             admin.Username,
			"must_change_password": admin.MustChangePassword,
		},
	})
}

// handleLogout 处理 POST /api/auth/logout。要求登录。
func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	sessionID := getSessionID(r.Context())
	if sessionID != "" {
		_ = h.store.RevokeSession(r.Context(), sessionID)
	}

	// 清除 cookie。
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	h.writeAudit(r, sessionID, "admin.logout", "admin", i64s(getAdminID(r.Context())), "管理员登出")
	writeOK(w, nil)
}

// handleMe 处理 GET /api/auth/me。要求登录。
func (h *Handler) handleMe(w http.ResponseWriter, r *http.Request) {
	adminID := getAdminID(r.Context())
	admin, err := h.store.GetAdminByID(r.Context(), adminID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, CodeInternal, "内部错误")
		return
	}
	writeOK(w, map[string]any{
		"id":                   admin.ID,
		"username":             admin.Username,
		"must_change_password": admin.MustChangePassword,
	})
}

// handleChangePassword 处理 POST /api/auth/change-password。要求登录。
func (h *Handler) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	adminID := getAdminID(r.Context())

	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, CodeBadRequest, "请求格式错误")
		return
	}

	admin, err := h.store.GetAdminByID(r.Context(), adminID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, CodeInternal, "内部错误")
		return
	}

	// 校验旧密码。
	if !security.VerifyPassword(admin.PasswordHash, req.OldPassword) {
		writeError(w, http.StatusBadRequest, CodeBadRequest, "旧密码错误")
		return
	}

	// 校验新密码强度。
	if len(strings.TrimSpace(req.NewPassword)) < minPasswordLength {
		writeError(w, http.StatusBadRequest, CodeWeakPassword, "新密码长度至少 10 位")
		return
	}

	// 哈希并更新。
	newHash, err := security.HashPassword(req.NewPassword)
	if err != nil {
		writeError(w, http.StatusInternalServerError, CodeInternal, "内部错误")
		return
	}
	if err := h.store.UpdateAdminPassword(r.Context(), adminID, newHash); err != nil {
		writeError(w, http.StatusInternalServerError, CodeInternal, "内部错误")
		return
	}

	// 撤销该管理员的所有其它 session(当前 session 保持有效,后续可改为全量撤销)。
	// 本轮保留当前 session,文档说明策略。
	// h.store.RevokeAllSessionsByAdminID(r.Context(), adminID)

	h.writeAudit(r, getSessionID(r.Context()), "admin.password.changed", "admin", i64s(adminID), "管理员修改密码")
	writeOK(w, nil)
}

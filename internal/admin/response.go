package admin

import (
	"encoding/json"
	"net/http"
)

// apiResponse 是管理 API 的统一 JSON 响应格式。
type apiResponse struct {
	OK   bool    `json:"ok"`
	Data any     `json:"data,omitempty"`
	Err  *apiErr `json:"error,omitempty"`
}

// apiErr 是 API 错误详情。code 使用稳定英文常量便于前端判断;message 中文为主。
type apiErr struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// 错误码常量。
const (
	CodeBadRequest   = "BAD_REQUEST"
	CodeUnauthorized = "UNAUTHORIZED"
	CodeForbidden    = "FORBIDDEN"
	CodeNotFound     = "NOT_FOUND"
	CodeConflict     = "CONFLICT"
	CodeInternal     = "INTERNAL"
	CodeWeakPassword = "WEAK_PASSWORD"
)

// writeJSON 写入 JSON 响应。
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeOK 写入成功响应。
func writeOK(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: data})
}

// writeError 写入错误响应。
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, apiResponse{OK: false, Err: &apiErr{Code: code, Message: message}})
}

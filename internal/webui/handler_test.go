package webui

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

// newTestHandler 用测试专用的 MapFS 构造 handler,不依赖实际 dist。
func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	testFS := fstest.MapFS{
		"dist/index.html":       {Data: []byte("<!DOCTYPE html><html><body>index</body></html>")},
		"dist/assets/app.js":    {Data: []byte("console.log('app')")},
		"dist/assets/style.css": {Data: []byte("body{}")},
		"dist/favicon.ico":      {Data: []byte("ico")},
	}
	return NewHandler(testFS)
}

func TestIndexReturnsIndexHTML(t *testing.T) {
	h := newTestHandler(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("/ 应返回 200,实际 %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("/ Content-Type = %q,期望 text/html", ct)
	}
}

func TestSPAFallbackReturnsIndexHTML(t *testing.T) {
	h := newTestHandler(t)
	paths := []string{"/dashboard", "/tenants", "/resources", "/domain-zones", "/grants", "/proxies", "/audit-logs", "/account"}
	for _, path := range paths {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("%s 应返回 200,实际 %d", path, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
			t.Errorf("%s Content-Type = %q,期望 text/html", path, ct)
		}
	}
}

func TestStaticAssetsReturned(t *testing.T) {
	h := newTestHandler(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("/assets/app.js 应返回 200,实际 %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/javascript; charset=utf-8" {
		t.Errorf("/assets/app.js Content-Type = %q,期望 application/javascript", ct)
	}
	if rec.Body.String() != "console.log('app')" {
		t.Errorf("/assets/app.js body = %q", rec.Body.String())
	}
}

func TestStaticCSSReturned(t *testing.T) {
	h := newTestHandler(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/assets/style.css", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("/assets/style.css 应返回 200,实际 %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/css; charset=utf-8" {
		t.Errorf("/assets/style.css Content-Type = %q,期望 text/css", ct)
	}
}

func TestEmptyFSReturns503(t *testing.T) {
	// 空 FS(未执行 sync-web-dist)应优雅降级,不 panic。
	emptyFS := fstest.MapFS{}
	h := NewHandler(emptyFS)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("空 FS 应返回 503,实际 %d", rec.Code)
	}
}

func TestAPIPathNotHandledByWebui(t *testing.T) {
	// webui handler 不应处理 /api/* 路径(这由 admin handler 处理)。
	// 但如果 webui handler 被错误挂载到 /api/*,它会返回 index.html(因为文件不存在)。
	// 这个测试验证 webui handler 对 /api/ 路径也会 fallback 到 index.html
	// (实际路由中 /api/ 由 adminMux 优先匹配,不会到这里)。
	h := newTestHandler(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	h.ServeHTTP(rec, req)

	// webui handler 会 fallback 到 index.html(因为 dist 中没有 api/auth/me 文件)。
	// 实际部署中 /api/ 由 adminMux 优先匹配,不会到达 webui handler。
	if rec.Code != http.StatusOK {
		t.Errorf("webui fallback 应返回 200,实际 %d", rec.Code)
	}
}

func TestGuessContentType(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"app.js", "application/javascript; charset=utf-8"},
		{"style.css", "text/css; charset=utf-8"},
		{"index.html", "text/html; charset=utf-8"},
		{"data.json", "application/json"},
		{"icon.svg", "image/svg+xml"},
		{"font.woff2", "font/woff2"},
		{"unknown.xyz", "application/octet-stream"},
	}
	for _, tt := range tests {
		got := guessContentType(tt.path)
		if got != tt.want {
			t.Errorf("guessContentType(%q) = %q,期望 %q", tt.path, got, tt.want)
		}
	}
}

// 确保 fstest.MapFS 满足 fs.FS 接口(编译时检查)。
var _ fs.FS = fstest.MapFS{}

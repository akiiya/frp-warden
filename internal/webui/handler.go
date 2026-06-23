package webui

import (
	"io/fs"
	"net/http"
	gopath "path"
	"strings"
)

// Handler 提供内嵌前端静态文件服务与 SPA fallback。
//
// 路由规则:
//   - /assets/* 等静态资源:直接返回,设置正确的 Content-Type。
//   - /、/dashboard、/tenants 等 SPA 路由:返回 index.html(由 Vue Router 处理)。
//   - 如果 fs 为空(未嵌入 dist),所有请求返回 503 提示。
type Handler struct {
	distFS fs.FS
	index  []byte
}

// NewHandler 用给定的 fs.FS 构造 webui handler。
//
// 生产环境传入 DistFS(内嵌的 web/dist);测试环境可传入 fstest.MapFS。
// 如果 fs 为空或 index.html 不存在,handler 会优雅降级(返回 503)。
func NewHandler(distFS fs.FS) *Handler {
	// 读取 index.html 内容,供 SPA fallback 使用。
	indexData, err := fs.ReadFile(distFS, "dist/index.html")
	if err != nil {
		// dist 未生成或 index.html 不存在,标记为空。
		return &Handler{distFS: distFS, index: nil}
	}
	return &Handler{distFS: distFS, index: indexData}
}

// ServeHTTP 实现 http.Handler。
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// dist 未嵌入,返回 503。
	if h.index == nil {
		http.Error(w, "前端资源未构建。请先执行: cd web && npm run build && cd .. && go run ./tools/sync-web-dist", http.StatusServiceUnavailable)
		return
	}

	// 清理路径。
	reqPath := strings.TrimPrefix(r.URL.Path, "/")
	if reqPath == "" {
		reqPath = "index.html"
	}

	// 尝试从嵌入 FS 中读取静态资源。
	// 注意:embed.FS 始终使用正斜杠,必须用 path.Join(而非 filepath.Join)。
	fullPath := gopath.Join("dist", reqPath)
	data, err := fs.ReadFile(h.distFS, fullPath)
	if err == nil {
		// 文件存在,设置 Content-Type 并返回。
		w.Header().Set("Content-Type", guessContentType(reqPath))
		w.Header().Set("Cache-Control", "public, max-age=31536000") // 静态资源长期缓存
		w.Write(data)
		return
	}

	// 文件不存在 → SPA fallback:返回 index.html。
	// Vue Router 会在客户端解析路由并渲染正确的页面。
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache") // index.html 不缓存
	w.Write(h.index)
}

// guessContentType 根据文件扩展名猜测 Content-Type。
func guessContentType(p string) string {
	ext := strings.ToLower(gopath.Ext(p))
	switch ext {
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js", ".mjs":
		return "application/javascript; charset=utf-8"
	case ".json":
		return "application/json"
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".ico":
		return "image/x-icon"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".eot":
		return "application/vnd.ms-fontobject"
	case ".map":
		return "application/json"
	default:
		return "application/octet-stream"
	}
}

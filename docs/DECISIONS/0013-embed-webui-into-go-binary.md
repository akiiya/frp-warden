# 0013 — 将前端内嵌进 Go 二进制

- 状态:已接受
- 日期:2026-06-23

## 背景

Phase 7 需要将 Vue 前端构建产物(`web/dist`)通过 Go `embed` 内嵌到单一可执行文件中,
使 frp-warden 启动后即可直接访问管理后台页面,无需额外部署前端。

## 决策与理由

### 为什么使用 Go embed

Go 1.16 引入的 `embed` 标准库支持将静态文件内嵌到二进制中,无需外部依赖。对于
frp-warden 这种"单二进制部署"的目标,embed 是最自然的选择:编译时将前端资源打包进
可执行文件,运行时零外部文件依赖。

### 为什么不把 web/dist 提交到仓库

`web/dist` 是构建产物,包含哈希文件名的 JS/CSS,每次构建都会变化。提交到仓库会导致
大量无意义的 diff 和仓库膨胀。因此 `web/dist/` 和 `internal/webui/dist/` 都被
`.gitignore` 忽略,只在构建时生成。

### 为什么使用 sync-web-dist 工具

Go `embed` 的路径限制:只能嵌入当前包目录或子目录下的文件。`web/dist` 在项目根目录下,
而 `internal/webui` 在另一个目录,不能直接 `//go:embed ../../web/dist`。因此需要一个
跨平台工具将 `web/dist` 同步到 `internal/webui/dist`。`tools/sync-web-dist` 是纯 Go
实现,Windows/Linux/macOS 都能运行,不依赖 shell 脚本或 Makefile。

### 为什么 /api/* 必须优先于 SPA fallback

admin server 同时挂载 API 和前端。`/api/*` 必须优先匹配 admin API handler,否则
`/api/auth/me` 等路径会被 webui handler 的 SPA fallback 吞掉,返回 `index.html` 而非
JSON 响应。Go 的 `http.ServeMux` 按最长前缀匹配,`/api/` 比 `/` 更具体,天然优先。

### 为什么 /plugin/frp 不应被 admin server 的 SPA fallback 处理

`/plugin/frp` 是 frps 的 server-plugin 接口,由独立的 plugin server 监听(默认
`127.0.0.1:9000`)。admin server 监听另一个端口(默认 `0.0.0.0:8080`),两者完全独立。
如果有人误访问 admin server 的 `/plugin/frp`,webui handler 会 fallback 到 `index.html`
(因为没有这个静态文件),这是可接受的——admin server 上根本没有 plugin handler。

### 为什么本轮不做 frpc.toml 生成

frpc 配置生成属于 Phase 8,需要根据 tenant 的授权资源生成完整的 frpc 配置文件。
这与前端内嵌是独立的功能,不应混在同一个阶段。

### 为什么 CI 需要先构建前端再构建 Go

Go `embed` 在编译时将文件打包进二进制。如果 `internal/webui/dist` 不存在,embed 指令
会失败(或嵌入空 FS)。因此 CI 流程必须是:前端构建 → sync-web-dist → Go 构建。
CI workflow 已更新为这个顺序。

## 影响

- `internal/webui/` 包含 `embed.go`(`//go:embed dist/*`)和 `handler.go`(静态文件
  服务 + SPA fallback)。
- `tools/sync-web-dist/main.go`:跨平台 Go 工具,清空 `internal/webui/dist` 并复制
  `web/dist` 的全部文件。
- `.gitignore` 新增 `internal/webui/dist/`。
- admin server 路由:`/api/*` → admin handler;`/` → webui handler。
- 构建单二进制的完整流程:

  ```sh
  cd web && npm install && npm run build && cd ..
  go run ./tools/sync-web-dist
  go build ./cmd/frp-warden
  ```

- 如果 `internal/webui/dist` 为空(未执行 sync),webui handler 返回 503 提示,不 panic。

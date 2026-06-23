// Package webui 将 Vue 前端构建产物通过 Go embed 内嵌到二进制中,并提供
// 静态文件服务与 SPA fallback。
//
// 内嵌流程:
//  1. 前端构建:cd web && npm run build → web/dist
//  2. 同步资源:go run ./tools/sync-web-dist → internal/webui/dist
//  3. Go 编译://go:embed dist/* 自动将 internal/webui/dist 内嵌到二进制
//
// 注意://go:embed 只能嵌入当前包目录下的文件,因此 web/dist 需先同步到
// internal/webui/dist(见 tools/sync-web-dist)。internal/webui/dist 不提交到仓库,
// 由 .gitignore 忽略。
package webui

import "embed"

// DistFS 是内嵌的前端构建产物。如果 dist 目录为空(未执行 sync-web-dist),
// 该变量仍会被初始化为空 FS,不会 panic;handler 会优雅降级。
//
//go:embed dist/*
var DistFS embed.FS

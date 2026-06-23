# 0015 — Release 多平台构建与产物发布

- 状态:已接受
- 日期:2026-06-23

## 背景

Phase 9 需要完善 GitHub Actions CI,并实现打 tag 时自动构建多平台 Release 产物、
创建 GitHub Release。需要在自动化、可复现性、安全性之间取得平衡。

## 决策与理由

### 为什么 CI 与 Release workflow 分离

CI workflow(`build.yml`)在每次 push/PR 时运行,快速反馈代码质量(gofmt/vet/test/
编译验证)。Release workflow(`release.yml`)只在打 tag 时运行,执行完整的构建+打包+
发布流程。两者职责不同、触发条件不同、复杂度不同,分离更清晰。

### 为什么 tag 触发 Release

tag 是 Git 中标记发布点的标准方式。`v*` 格式的 tag(如 `v0.1.0`、`v0.1.0-rc1`)
语义明确,易于自动化。tag 从 main 分支创建,保证 Release 来自经过 CI 验证的代码。

### 为什么先构建前端再 sync-web-dist 再 Go build

Go `embed` 在编译时将文件打包进二进制。如果 `internal/webui/dist` 不存在,embed
会失败。因此构建顺序必须是:前端构建(web/dist) → sync-web-dist(internal/webui/dist)
→ Go 构建(内嵌)。

### 为什么 Release 产物是单二进制

frp-warden 的核心设计目标之一就是"单一可执行文件部署"。前端已通过 Go embed 内嵌,
数据库使用纯 Go SQLite 驱动(无 CGO),配置自动生成。运维只需下载一个文件即可运行。

### 为什么使用 CGO_ENABLED=0

CGO 需要 C 编译器,增加构建复杂度,且跨平台编译困难。项目使用的 `modernc.org/sqlite`
是纯 Go 实现,无需 CGO。`CGO_ENABLED=0` 保证交叉编译简单、产物静态链接、兼容性好。

### 为什么生成 checksums.txt

checksums(SHA-256)让用户可以验证下载的产物未被篡改。这是发布安全的基本实践。

### 为什么不提交 web/dist 和 internal/webui/dist

这些是构建产物,每次构建都会变化。提交到仓库会导致大量无意义 diff 和仓库膨胀。
它们被 `.gitignore` 忽略,只在构建时生成。

### 为什么本轮不做 Docker

Docker 化属于独立的部署优化,与 Release 构建是不同的关注点。当前单二进制已经足够
简单,后续可以按需添加 Dockerfile。

### 为什么 main 只能 PR 合并,正式 Release 应从 main tag 触发

main 是受保护分支,代表可发布状态。PR 合并保证代码经过 review 和 CI 验证。
从 main 打 tag 触发 Release,保证每个 Release 都来自经过验证的代码。

## 影响

- `.github/workflows/build.yml`:CI workflow,验证前端+Go 构建。
- `.github/workflows/release.yml`:Release workflow,打 tag 时构建+打包+发布。
- `.gitignore`:新增 `release/`、`artifacts/`、`*.zip`、`*.tar.gz`、`/checksums.txt`。
- Release 产物:6 平台(zip/tar.gz + checksums.txt)。
- 版本注入:ldflags 注入 `Version`/`Commit`/`BuildDate`。

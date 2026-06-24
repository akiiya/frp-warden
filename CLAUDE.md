# CLAUDE.md — frp-warden 的 Claude Code 上下文

本文件是 Claude Code 在本仓库工作的操作手册。动手之前请完整阅读。
(其它 AI 工具见 [AGENTS.md](AGENTS.md),内容一致。)

## 本项目是什么

frp-warden 是一个**面向 frp 的多租户授权控制面**。它决定哪个 tenant 可以使用
哪个 subdomain/端口,并通过 frps server plugin 的 HTTP hooks 强制执行。
写代码前请先读 [docs/PROJECT_CONTEXT.md](docs/PROJECT_CONTEXT.md) 了解完整模型。

## 语言规范(强制)

- **所有文档、代码注释、面向用户的说明必须以中文为主。**
- README 可保留必要的英文 tagline(`Multi-tenant authorization control plane for frp.`),
  但正文以中文为主。
- 技术术语保留英文,不要强行翻译:tenant、token、proxy、subdomain、frps plugin、
  Login、NewProxy、Ping、subdomainHost、YAML、SQLite 等。
- 不要翻译协议字段、配置键名、API 路径、Go 包名/标识符。
- 错误信息可中可英,但面向用户的说明优先中文。

## 每次任务前必读

先读以下文档,再读你改动所涉及的具体文档:

1. 本文件(`CLAUDE.md`)
2. [docs/PROJECT_CONTEXT.md](docs/PROJECT_CONTEXT.md)
3. [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
4. [docs/ROADMAP.md](docs/ROADMAP.md) — 确认当前阶段与范围
5. 你所在领域的具体文档:`DATA_MODEL`、`FRP_PLUGIN`、`API`、`CONFIGURATION`、`SECURITY`

## 硬性规则

- **禁止在 `main` 分支开发。** 日常开发在 `dev`。若发现自己在 `main`,先切到 `dev`。
- **未经明确要求不得 commit 或 push。** 只修改工作区,交给用户评审。
- **文档不是可选项。** 每次代码改动都必须在同一轮里同步更新相关文档。
- **不要过度实现。** 只做当前阶段范围内的事,不要顺手实现后续阶段的功能。
- **不要使用"AI 记忆"保存项目上下文。** 上下文必须沉淀到仓库文档,确保换 session、
  换工具也能凭文档继续开发。
- **安全不变量不可妥协**(见 [docs/SECURITY.md](docs/SECURITY.md)):不存在 `admin/admin`,
  密码与 token 仅存 hash,plugin 端口默认 `127.0.0.1`,所有授权在服务端强制
  (`NewProxy` 是关键校验点),frpc 客户端配置不可信。

## 技术栈

- **后端:** Go,单一二进制,无 CGO,YAML 配置,默认 SQLite
  (后续支持 mysql/mariadb/postgresql),SQLite 使用纯 Go 驱动。
- **前端:** Vue 3 + Vite + TypeScript,Naive UI,构建到 `web/dist` 并经 Go `embed` 内嵌。
- **CI:** GitHub Actions,多平台构建矩阵。

## 当前 MVP 范围

整体按阶段推进,见 [docs/ROADMAP.md](docs/ROADMAP.md)。当前已完成 **Phase 0~10**:
项目初始化与文档、配置加载与自动初始化、数据库连接/迁移框架/管理员初始化、
tenant/resource/grant/proxy 核心数据模型与唯一约束、frps plugin Login/NewProxy/Ping
真实鉴权、管理后台 REST API 与管理员 session、Vue 3 + Naive UI 管理后台前端、
前端内嵌进 Go 二进制、frpc 配置生成、GitHub Actions Release 多平台构建,以及
**Phase 10(发布前清理、安装部署文档、端到端冒烟测试准备)**。
数据库目前仅 **SQLite 完整可用**(纯 Go 驱动,无 CGO);MySQL/MariaDB/PostgreSQL
仅有连接骨架,迁移未实现(fail fast)。**状态:v0.1.0-rc 准备中。** 版本号由根目录 `VERSION` 文件控制;main 合并后自动发布。

## 任务结束必须输出(固定格式)

完成任务时,按如下结构总结:

1. **修改的代码文件** — 列出路径。
2. **修改的文档文件** — 列出路径。
3. **运行的验证命令及结果**(`gofmt`、`go vet`、`go test ./...`、
   `go build ./cmd/frp-warden`,以及前端构建若适用)。
4. **是否需要后续迁移或补充** — 任何被推迟、或需后续数据迁移/手动步骤的事项。

## 环境说明

- 本机 Go 可能位于 `C:\go_v1.26`(`C:\go_v1.26\bin\go.exe`),且未必在 `PATH` 中。
  使用前先检测,不要假设 `go` 一定可用。
- 验证命令:`go fmt ./...`、`go vet ./...`、`go test ./...`、`go build ./cmd/frp-warden`。

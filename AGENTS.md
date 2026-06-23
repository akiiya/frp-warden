# AGENTS.md — 面向 AI 编码代理的说明

本文件面向**任何** AI 编码工具(Claude Code、Cursor、Copilot agents、Aider 等)。
它刻意保持工具中立。Claude Code 用户:[CLAUDE.md](CLAUDE.md) 承载同样的规则。

## 一句话项目定位

frp-warden 是一个**面向 frp 的多租户授权控制面** —— 它决定哪个 tenant 可以使用
哪个 subdomain/端口,并通过 frps server plugin 的 HTTP hooks 强制执行。
完整背景见 [docs/PROJECT_CONTEXT.md](docs/PROJECT_CONTEXT.md)。

## 语言规范(强制)

- 所有文档、代码注释、面向用户的说明**必须以中文为主**。
- README 可保留必要英文 tagline,但正文以中文为主。
- 技术术语保留英文(tenant、token、proxy、subdomain、frps plugin、Login、
  NewProxy、Ping、YAML、SQLite 等);不要翻译协议字段、配置键名、API 路径、Go 包名。

## 文档驱动开发

本仓库是文档驱动的。动手前:

1. 阅读 [docs/PROJECT_CONTEXT.md](docs/PROJECT_CONTEXT.md)、
   [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) 与 [docs/ROADMAP.md](docs/ROADMAP.md),
   了解模型与当前阶段。
2. 阅读你所改动领域的具体文档(`DATA_MODEL`、`FRP_PLUGIN`、`API`、`CONFIGURATION`、
   `SECURITY`)。

每次代码改动**都必须**在同一轮里更新相关文档。

## 基本规则

- 在 `dev` 分支开发,绝不直接在 `main` 上开发。
- 未经明确要求不要 commit 或 push;把改动留在工作区供评审。
- 把改动控制在当前阶段范围内(见 [docs/ROADMAP.md](docs/ROADMAP.md)),不要提前实现
  后续阶段的功能。
- 保持依赖精简,优先标准库与无 CGO 的包。
- 不要使用任何"AI 记忆"保存项目上下文;上下文沉淀到仓库文档。

## 安全不变量(必须始终成立)

见 [docs/SECURITY.md](docs/SECURITY.md)。简要:

- 不存在硬编码的 `admin/admin`;初始管理员密码随机生成。
- 密码与 tenant token 仅以 **hash** 存储,绝不明文。
- frps plugin 端点默认绑定 `127.0.0.1`,不得暴露公网。
- 所有授权在服务端强制执行,`NewProxy` 是关键校验点;frpc 客户端配置不可信。

## 技术栈

- 后端:Go,单一无 CGO 二进制,YAML 配置,默认 SQLite(纯 Go 驱动 `modernc.org/sqlite`),
  数据访问用标准库 `database/sql` + 手写迁移,**不引入 ORM**。
  当前仅 SQLite 完整可用;mysql/mariadb/postgresql 仅连接骨架、迁移未实现(fail fast)。
- 已完成至 Phase 10(发布前清理、安装部署文档、端到端冒烟测试准备)。
  详见 [docs/ROADMAP.md](docs/ROADMAP.md)。
- 前端:Vue 3 + Vite + TypeScript + Naive UI,经 Go `embed` 内嵌。
- CI:GitHub Actions 多平台构建矩阵。

## 验证你的改动

运行并汇报结果:

```sh
go fmt ./...
go vet ./...
go test ./...
go build ./cmd/frp-warden
```

(前端存在后,再加上前端构建。)

## 完成时的汇报格式

列出:修改的代码文件、修改的文档文件、运行的验证命令及结果,以及任何后续/迁移事项。

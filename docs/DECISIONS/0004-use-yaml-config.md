# 0004 — 使用 YAML 配置文件

- 状态:已接受
- 日期:2026-06-12

## 背景

frp-warden 需要面向运维人员的配置(监听地址、数据库、
密钥、frp 参数)。它应当易读、便于手工编辑、支持
注释,并让 frp 运维人员感到熟悉(frp 使用 TOML,但 YAML 在
这个生态中被广泛使用,且团队在此倾向于使用它)。

## 决策

使用**单个 YAML 配置文件**。其 schema 与 `internal/config` 中的 Go
struct 相对应,并在 [../CONFIGURATION.md](../CONFIGURATION.md) 中记录。
首次运行时,frp-warden 会生成带有文档化默认值的配置文件;同时生成一个空的
`session_secret` 并写回文件。

## 影响

- 对人友好的配置,支持注释与层级结构。
- 零配置首次启动:文件缺失时会被自动创建。
- 每次改动都必须让配置 struct 与 `CONFIGURATION.md` 保持同步。
- 密钥保存在该文件中(session secret),因此必须将其视为敏感信息
  (参见 [../SECURITY.md](../SECURITY.md))。初始管理员密码**不会**
  存放于此 —— 数据库中仅保存其哈希值。

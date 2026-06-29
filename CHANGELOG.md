# 更新日志

本文件记录 frp-warden 的重要变更。格式参考 [Keep a Changelog](https://keepachangelog.com/),
遵循 [语义化版本](https://semver.org/)。

## v0.1.0-rc2

### 变更

- 基于 v0.1.0-rc1 的重新构建:修正 git 历史中的作者邮箱( contributors 修正)。
- 功能代码与 v0.1.0-rc1 完全一致,无业务逻辑改动。
- 二进制 `-version` 显示的 commit 哈希与当前历史一致。

### 已知限制

- MySQL/MariaDB/PostgreSQL 迁移尚未完整实现。
- Docker 尚未提供。
- SBOM/签名尚未提供。
- 暗色主题尚未实现。
- CloseProxy 暂未启用。

## v0.1.0-rc1

### 新增

- **Phase 0~9 完整功能**(详见 [docs/ROADMAP.md](docs/ROADMAP.md)):
  - 配置加载与自动初始化(Phase 1)
  - SQLite 数据库连接、迁移框架、管理员初始化(Phase 2)
  - tenant/resource/grant/proxy 核心数据模型与唯一约束(Phase 3)
  - frps plugin Login/NewProxy/Ping 真实鉴权(Phase 4)
  - 管理后台 REST API 与 session 认证(Phase 5)
  - Vue 3 + Naive UI 管理后台前端(Phase 6)
  - 前端内嵌进 Go 二进制(Phase 7)
  - frpc 配置生成(Phase 8)
  - GitHub Actions CI 与 Release 多平台构建(Phase 9)
- **Phase 10 发布准备**:
  - 安装部署文档(docs/INSTALL.md)
  - frps 配置指南(docs/FRPS_SETUP.md)
  - systemd 部署示例(docs/SYSTEMD.md)
  - 反向代理建议(docs/REVERSE_PROXY.md)
  - 端到端冒烟测试文档(docs/SMOKE_TEST.md)
  - 常见问题(docs/TROUBLESHOOTING.md)
  - GitHub 仓库与发布说明(docs/GITHUB.md)
  - 配置示例(config.example.yaml)
  - frps/frpc 配置示例(examples/)
  - MIT License
  - 安全政策(SECURITY.md)
- **数据库**:SQLite 完整可用(纯 Go 驱动,无 CGO)。
- **支持平台**:windows/amd64、windows/386、linux/amd64、linux/386、linux/arm64、linux/arm v7。

### 安全

- 管理员密码随机生成,仅显示一次,仅存 bcrypt hash。
- tenant token 仅在创建/重置时显示一次,数据库只存 bcrypt hash。
- frps plugin 端口默认 `127.0.0.1:9000`,不允许暴露公网。
- session cookie HttpOnly + SameSiteLax。
- 审计日志不记录密码/token/secret。

### 已知限制

- MySQL/MariaDB/PostgreSQL 迁移尚未完整实现(仅连接骨架)。
- Docker 尚未提供。
- SBOM/签名尚未提供。
- 暗色主题尚未实现。
- CloseProxy 暂未启用。
- frpc 配置生成依赖 token 明文(只在创建/重置时可用)。

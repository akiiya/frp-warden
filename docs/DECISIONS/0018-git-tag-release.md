# 0018 — Git tag 触发发布(取代 VERSION 文件)

- 状态:已接受
- 日期:2026-06-24
- 取代:[0017 — VERSION 文件控制版本与 main 自动发布](0017-version-file-and-main-auto-release.md)

## 背景

0017 采用"VERSION 文件控制版本号 + main 分支 push 时 workflow 读取 VERSION 自动打 tag 发版"的模型。
实践中发现该模型存在摩擦:每次发布都要改 VERSION 文件、合并 PR;且"workflow 自建 tag"与
"tag push 触发另一个 workflow"在 GITHUB_TOKEN 下存在递归触发限制,必须把建 tag 与建 Release
塞进同一个 workflow,逻辑较重。

为此,改为与其它 Go 项目一致的、更轻量的"Git tag 触发发布"标准。

## 决策

**版本号以 Git tag 为唯一来源,不使用 VERSION 文件。**

1. 正式版本号 = 推送的 `vX.Y.Z` tag 去掉前缀 `v`,通过 ldflags 注入 `internal/version.Version`。
2. 日常 / CI 构建的版本号由 `git describe --tags --always --dirty` 派生(可被环境变量 `VERSION` 覆盖)。
3. 发版动作 = **人工打一个新的 `v*` tag**(命令行 push 或 GitHub 网页 Draft a new release 新建 tag),
   由 `release.yml` 自动完成:测试 → 多平台打包 → 发布 Release。
4. 分支模型不变:`main` 受保护(必须 PR、禁止 force-push),日常在 `dev`,`dev → main` 走 PR,提交遵循 Conventional Commits。

### 工件

- `internal/version/version.go`:`var Version = "dev"`,由 ldflags 注入。
- `scripts/build.sh`:本地单二进制(版本来自 git describe)。
- `scripts/release.sh`:`go vet`/`go test` → 前端构建并同步 → 多平台打包 → SHA256SUMS。
- `Makefile`:`build` / `test` / `vet` / `release`(版本同样来自 git describe)。
- `.github/workflows/ci.yml`:dev 推送 + PR 触发,只验证不发布。
- `.github/workflows/release.yml`:仅 `push: tags: v*`(及 `workflow_dispatch` 重跑)触发发布。
- 统一构建参数:`-trimpath -ldflags "-s -w -X <MODULE>/internal/version.Version=${VERSION}"`、`CGO_ENABLED=0`。

## 与 0017 的差异

| 方面 | 0017(已取代) | 0018(现行) |
|---|---|---|
| 版本来源 | VERSION 文件 | Git tag |
| 发版触发 | main push 自动 | 推送 `v*` tag |
| 建 tag 者 | workflow 自建 | 人工推送 |
| VERSION 文件 | 需要 | 删除 |

## 为什么改回 tag 触发

- **更简单**:发版就是打 tag,无需改文件、无需 workflow 自建 tag 的复杂逻辑。
- **更标准**:与社区主流 Go 项目的发布模式一致,运维易于理解。
- **避免递归触发陷阱**:tag 由人工推送,`push: tags: v*` 直接触发发布,不依赖 workflow 建 tag。

## 影响

- 删除 `VERSION` 文件及一切读取它的逻辑。
- 删除旧的 `build.yml`(由 `ci.yml` 取代)与旧的 main-push release 逻辑。
- `main` 合并不再自动发版;发版改由人工打 tag 触发。
- 已存在的 `v0.1.0-rc1` tag/Release 保持不变;后续发版打新 tag。

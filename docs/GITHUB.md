# GitHub 仓库与发布说明

## 分支策略

| 分支 | 用途 | 说明 |
|---|---|---|
| `dev` | 日常开发 | 所有开发在 `dev` 进行,提交遵循 Conventional Commits |
| `main` | 受保护分支 | 只通过 PR 合并,禁止 force push |

### 分支保护建议

在 GitHub 仓库设置中为 `main` 分支启用保护规则:

- 要求 PR 才能合并。
- 禁止 force push。
- 建议要求 CI 通过后才能合并。
- 建议要求至少 1 个 review。

## 版本来源

**版本号以 Git tag 为唯一来源,不使用 VERSION 文件。**

- 正式版本号 = 推送的 `vX.Y.Z` tag 去掉前缀 `v`,通过 ldflags 注入二进制。
- 日常 / CI 构建的版本号由 `git describe --tags --always --dirty` 派生。
- 发版动作 = **打一个新的 `v*` tag**。

## CI

CI workflow (`.github/workflows/ci.yml`) 在以下情况运行:

- push 到 `dev`
- 任意 pull_request

CI 任务:前端构建并同步 → `go vet` → `go test` → `go build`(版本来自 git describe)。
CI **不**创建 tag,不创建 Release。

## Release

Release workflow (`.github/workflows/release.yml`) 在**推送 `v*` tag** 时自动触发。

### 发布流程

1. 在 `dev` 用 Conventional Commits 开发 → 提 PR 到 `main` → CI 检查 → 审批合并。
2. 发版:打一个**新的** `vX.Y.Z` tag。二选一:
   - 命令行:`git tag vX.Y.Z && git push origin vX.Y.Z`
   - 网页:Draft a new release → Choose a tag 处输入新 tag → Target 选 `main` → Publish。
3. Release workflow 自动:测试 → 多平台打包 → 发布 Release(附压缩包 + SHA256SUMS,发布说明自动生成)。
4. 重发同一版本:用 workflow_dispatch 指定 tag 重跑;或新建更高版本 tag。**版本号 tag 不要复用。**

### 不需要 VERSION 文件

版本号完全来自 tag。仓库根目录没有 VERSION 文件,也不要手改任何版本文件来发版。

### 多平台产物

CI/Release 构建目标(`scripts/release.sh`):

| 平台 | 归档 |
|---|---|
| linux/amd64 | tar.gz |
| linux/arm64 | tar.gz |
| linux/386 | tar.gz |
| linux/armv7 | tar.gz |
| windows/amd64 | zip |
| windows/386 | zip |

每个归档含二进制 + `README.md` + `LICENSE`;`dist/SHA256SUMS` 提供校验和。

### 预发布判断

由 tag 名是否含 `rc`/`beta`/`alpha` 等决定(GitHub 自动生成 release notes 时按 tag 语义)。

## 本地构建/打包

```sh
# 单二进制(版本来自 git describe)
bash ./scripts/build.sh        # 或 make build

# 完整多平台打包(产出 dist/*.tar.gz、*.zip、SHA256SUMS,不真正发版)
bash ./scripts/release.sh      # 或 make release
```

## tag ruleset

如果仓库启用了 tag ruleset,需要允许维护者创建 `v*` tag(Release workflow 不主动建 tag,tag 由人工推送)。

## 不要提交的文件

被 `.gitignore` 忽略:`dist/`、`web/dist/`、`internal/webui/dist/`、`data/`、`*.db`、`config.yaml`、`node_modules/`、`release/`、`artifacts/`。

## 版本信息注入

构建时通过 ldflags 注入:

```
-trimpath -ldflags "-s -w -X github.com/fengheasia/frp-warden/internal/version.Version=${VERSION}"
```

```sh
frp-warden -version
# frp-warden X.Y.Z-rcN(正式版则为 X.Y.Z)
```

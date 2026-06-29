# GitHub 仓库与发布说明

## 分支策略

| 分支 | 用途 | 说明 |
|---|---|---|
| `dev` | 日常开发 | 所有开发在 `dev` 进行 |
| `main` | 受保护分支 | 只通过 PR 合并,不允许直接 push |

### 分支保护建议

在 GitHub 仓库设置中为 `main` 分支启用保护规则:

- 要求 PR 才能合并。
- 禁止 force push。
- 建议要求 CI 通过后才能合并。
- 建议要求至少 1 个 review。

## 版本控制

版本号由根目录 `VERSION` 文件唯一控制。

- `VERSION` 文件内容不带 `v` 前缀,例如 `X.Y.Z` 或 `X.Y.Z-rcN`。
- Release workflow 读取 `VERSION` 后自动生成 tag `v${VERSION}`。
- `VERSION` 文件必须通过 `dev → PR → main` 流程修改。
- 不要手动在 main 上修改 `VERSION`。

## CI

CI workflow (`.github/workflows/build.yml`) 在以下情况运行:

- push 到 `dev`
- PR 到 `main`

CI 任务:

1. 前端构建(npm ci + type-check + build)
2. 同步前端资源(sync-web-dist)
3. Go 检查(gofmt + vet + test)
4. 多平台编译验证

CI **不**创建 tag,不创建 Release。

## Release

Release workflow (`.github/workflows/release.yml`) 在 **push 到 main** 时自动触发。

### 发布流程

1. 在 `dev` 分支修改代码。
2. 如果准备发布,修改 `VERSION` 文件(改为新版本号)。
3. push `dev`。
4. 创建 PR 到 `main`。
5. CI 通过后合并 PR。
6. `main` push 自动触发 Release workflow。
7. workflow 读取 `VERSION`,生成 tag `v${VERSION}`。
8. 如果 tag 已存在,workflow **明确失败**,提示更新 `VERSION`。
9. workflow 执行:前端构建 → sync-web-dist → Go 测试 → 多平台构建 → 创建 tag → 创建 Release → 上传资产。

### 不需要人工打 tag

`main` 分支合并后,Release workflow 自动完成所有操作,包括创建 tag。

### tag 已存在时的行为

如果 `VERSION` 对应的 tag 已存在,workflow 会明确失败,并提示:

> 当前 VERSION 已发布。请在 dev 分支更新 VERSION 后重新合并 main。

解决方法:在 `dev` 分支更新 `VERSION` 文件,重新 PR 合并 `main`。

### tag 命名规则

- `VERSION = X.Y.Z-rcN` → tag = `vX.Y.Z-rcN`
- `VERSION = X.Y.Z` → tag = `vX.Y.Z`

### 预发布判断

- `VERSION` 包含 `rc`/`beta`/`alpha` 时,标记为 prerelease。
- 否则标记为正式 release。

### 手动触发

Release workflow 支持 `workflow_dispatch` 手动触发(备用),但主流程是 main push 自动触发。

## tag ruleset

如果仓库启用了 tag ruleset,需要允许 `github-actions[bot]` 创建 `v*` tag。

## 不要提交的文件

以下文件/目录被 `.gitignore` 忽略,不要提交:

- `web/dist/` — 前端构建产物
- `internal/webui/dist/` — 同步后的嵌入资源
- `data/` — SQLite 数据库
- `config.yaml` — 运行时配置(含 session_secret)
- `release/`、`artifacts/` — Release 构建产物
- `*.zip`、`*.tar.gz` — 压缩包
- `node_modules/` — 前端依赖

`VERSION` 文件**不**被忽略,必须提交到仓库。

## 版本信息注入

构建时通过 ldflags 注入版本信息:

- `Version`: tag 名(如 `vX.Y.Z-rcN`)
- `Commit`: git commit hash 前 8 位
- `BuildDate`: UTC 构建时间

```sh
frp-warden -version
# frp-warden vX.Y.Z-rcN (commit abc12345, built 2026-06-23T12:00:00Z)
```

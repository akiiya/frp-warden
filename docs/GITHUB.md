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

## CI

CI workflow (`.github/workflows/build.yml`) 在以下情况运行:

- push 到 `dev`
- push 到 `main`
- PR 到 `main`

CI 任务:

1. 前端构建(npm ci + type-check + build)
2. 同步前端资源(sync-web-dist)
3. Go 检查(gofmt + vet + test)
4. 多平台编译验证

CI 不上传 Release 产物。

## Release

Release workflow (`.github/workflows/release.yml`) 在打 tag 时触发。

### tag 命名

- 正式版: `v0.1.0`、`v0.2.0`
- 预发布: `v0.1.0-rc1`、`v0.1.0-beta1`、`v0.1.0-alpha1`

### 如何触发 Release

```sh
# 1. 确保 main 分支代码是最新的
git checkout main
git pull origin main

# 2. 打 tag
git tag v0.1.0-rc1

# 3. 推送 tag
git push origin v0.1.0-rc1
```

推送 tag 后,Release workflow 自动运行:

1. 前端构建
2. 同步前端资源
3. 6 平台交叉编译(CGO_ENABLED=0)
4. 打包(zip/tar.gz)
5. 生成 checksums.txt
6. 创建 GitHub Release 并上传产物

### 查看 artifacts

在 GitHub 仓库的 Actions 页面查看 workflow 运行状态。完成后,在 Releases 页面
查看上传的产物。

### 预发布判断

- tag 含 `-rc`/`-beta`/`-alpha` 时,标记为 prerelease。
- 否则标记为正式 release。

## 不要提交的文件

以下文件/目录被 `.gitignore` 忽略,不要提交:

- `web/dist/` — 前端构建产物
- `internal/webui/dist/` — 同步后的嵌入资源
- `data/` — SQLite 数据库
- `config.yaml` — 运行时配置(含 session_secret)
- `release/`、`artifacts/` — Release 构建产物
- `*.zip`、`*.tar.gz` — 压缩包
- `node_modules/` — 前端依赖

## 版本信息

构建时通过 ldflags 注入版本信息:

- `Version`: tag 名或 `0.0.0-dev`
- `Commit`: git commit hash
- `BuildDate`: UTC 构建时间

```sh
frp-warden -version
# frp-warden v0.1.0-rc1 (commit abc12345, built 2026-06-23T12:00:00Z)
```

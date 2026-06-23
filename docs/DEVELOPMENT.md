# 开发

## 先决条件

- **Go**(模块目标 `go 1.25.0`;本轮引入的 `modernc.org/sqlite`、`pgx`、`x/crypto`
  等依赖要求 Go 1.25+。本仓库使用 Go 1.26 开发)。某些开发机上 Go 位于
  `C:\go_v1.26\bin\go.exe` 且未必在 `PATH` 中——使用前先检测,不要假设 `go` 可用:

  ```sh
  go version || /c/go_v1.26/bin/go.exe version
  ```

- **Node.js + 包管理器**(npm/pnpm),用于 `web/` 下的前端(Phase 6+)。

> 部署拓扑提醒:frps 与 frp-warden 都部署在公网服务器 / VPS(同机或同内网),
> frp-warden 的 `plugin_addr` 默认仅监听 `127.0.0.1`;随身 WiFi / Debian 等设备只运行
> frpc 客户端。详见 [PROJECT_CONTEXT.md](PROJECT_CONTEXT.md) 与 [ARCHITECTURE.md](ARCHITECTURE.md)。

## 目录结构

```
cmd/frp-warden/        入口程序(配置→建库→迁移→管理员初始化→启动摘要)
internal/config/       配置 struct、默认值、YAML 加载/生成、校验、数据目录
internal/db/           数据库连接、方言判定、轻量迁移框架(SQLite 完整)
internal/security/     密码相关安全原语(随机密码生成、bcrypt 哈希/校验)
internal/bootstrap/    首次启动初始化(默认管理员创建)
internal/server/       组装 admin + plugin HTTP 服务
internal/admin/        管理后台 REST API(Phase 5,session cookie 认证)
internal/plugin/       frps server plugin hooks:Login/NewProxy/Ping 真实鉴权(Phase 4,fail closed)
internal/store/        tenant/resource/grant/proxy 数据访问层(Phase 3,含 ProxyAuthContext)
internal/frpcconfig/   frpc.toml 配置生成器(Phase 8)
internal/model/        领域实体
internal/version/      构建/版本元信息
internal/webui/        Go embed 内嵌前端静态资源 + SPA fallback(Phase 7)
tools/sync-web-dist/   跨平台工具:web/dist → internal/webui/dist
web/                   Vue 3 + Vite + TS 前端
docs/                  文档(本目录)
.github/workflows/     CI(build.yml) + Release(release.yml)
```

## 常用命令

后端:

```sh
go fmt ./...
go vet ./...
go test ./...
go build -o frp-warden ./cmd/frp-warden

# 运行
./frp-warden -version          # 打印版本后退出
./frp-warden                   # 加载/生成配置→建库→迁移→管理员初始化→打印启动摘要(Phase 2)
./frp-warden -c ./config.yaml  # 指定配置文件路径
./frp-warden -serve            # 可选:启动占位的 admin + plugin 服务
```

> 本机请优先使用 `C:\go_v1.26\bin\go.exe` 代替 `go`。

带版本元信息构建(与 CI 一致):

```sh
go build -ldflags "\
  -X github.com/fengheasia/frp-warden/internal/version.Version=v0.1.0 \
  -X github.com/fengheasia/frp-warden/internal/version.Commit=$(git rev-parse --short HEAD) \
  -X github.com/fengheasia/frp-warden/internal/version.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o frp-warden ./cmd/frp-warden
```

前端(Phase 6 已完成 —— 见 [`web/README.md`](../web/README.md)):

```sh
cd web
npm install
npm run type-check  # TypeScript 类型检查
npm run dev         # 本地开发服务器(端口 5173, /api 代理到 :8080)
npm run build       # 输出 web/dist(Phase 7 内嵌进二进制)
```

## 交叉编译

由于避免了 CGO,交叉编译只需设置 `GOOS`/`GOARCH`:

```sh
GOOS=linux   GOARCH=amd64 go build -o dist/frp-warden-linux-amd64   ./cmd/frp-warden
GOOS=windows GOARCH=amd64 go build -o dist/frp-warden-windows-amd64.exe ./cmd/frp-warden
GOOS=linux   GOARCH=arm64 go build -o dist/frp-warden-linux-arm64   ./cmd/frp-warden
```

目标平台:windows/amd64、windows/386、linux/amd64、linux/386、linux/arm64、
linux/arm/v7。见 [.github/workflows/build.yml](../.github/workflows/build.yml)。

## 依赖

- YAML 解析使用 `gopkg.in/yaml.v3`(轻量、成熟)。
- 数据库:`modernc.org/sqlite`(纯 Go,无 CGO);`github.com/go-sql-driver/mysql`
  与 `github.com/jackc/pgx/v5/stdlib`(已注册,迁移待实现)。数据访问基于标准库
  `database/sql`,手写迁移。
- 密码哈希:`golang.org/x/crypto/bcrypt`。
- 保持依赖精简:**不引入** ORM(GORM/Ent)、Web 框架(Gin/Echo/Fiber)、复杂迁移工具、
  前端依赖或大型日志框架。

## 工作流约定

- 在 `dev` 分支开发,绝不在 `main`。
- 代码改动同步更新文档与 `CHANGELOG.md`。
- 把改动控制在当前阶段范围内([ROADMAP.md](ROADMAP.md))。
- 文档与注释以中文为主(见 [CLAUDE.md](../CLAUDE.md) / [AGENTS.md](../AGENTS.md))。
- 未经要求不要 commit / push。

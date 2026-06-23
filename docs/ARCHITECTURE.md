# 架构

frp-warden 是单个 Go 二进制文件,内置嵌入式 Vue 前端。它运行两个 HTTP 监听器,
并连接一个数据库。它是与 frps 相互独立的进程。

## 组件总览

```
                         frp-warden (single binary)
   +--------------------------------------------------------------+
   |  config loader (YAML)        version/build info              |
   |        |                                                     |
   |        v                                                     |
   |  +-------------+      +----------------------------------+   |
   |  |  internal/  |      |  admin server  (internal/admin)  |   |
   |  |  config     |----->|  :8080  REST API + embedded Vue  |<--+--- admin / browser
   |  +-------------+      +----------------------------------+   |
   |        |                            |                        |
   |        v                            v                        |
   |  +-------------+      +----------------------------------+   |
   |  |  db (sqlite |<---->|  plugin server (internal/plugin) |<--+--- frps (HTTP hooks)
   |  |  default)   |      |  :9000  /plugin/frp  LOOPBACK    |   |
   |  +-------------+      +----------------------------------+   |
   |        ^                                                     |
   |        |  models (internal/model) + frpc config generation  |
   +--------------------------------------------------------------+
```

## 进程模型与部署拓扑

- **frps** 与 **frp-warden** 都部署在**正规的公网服务器 / VPS** 上,位于同一台主机或
  同一内网可达。frps 监听 frpc 客户端(默认 `:7000`),并通过 HTTP 调用 frp-warden 的
  plugin 接口进行授权;frp-warden 从不转发流量,只回答策略问题并提供管理控制台。
- 因为 frps 与 frp-warden 同机/同内网,frp-warden 的 plugin 接口默认仅监听
  `127.0.0.1:9000`,**只接受本机 frps 调用,不暴露公网**。
- **随身 WiFi / Debian 等设备只作为客户端运行 frpc**,既不运行 frps,也不运行 frp-warden。
  运维在 frp-warden 管理页面创建 tenant、授权资源、生成 frpc 配置,再复制到设备上启动。

```
   公网服务器 / VPS                              客户端设备(随身 WiFi / Debian)
   +------------------------------+              +-----------------------------+
   |  frps  :7000 ……………… 控制端口  | <----------- |  frpc(携带 tenant token)    |
   |   |  ^ HTTP plugin hooks      |   公网隧道    +-----------------------------+
   |   v  |  127.0.0.1:9000        |
   |  frp-warden(管理后台 + plugin)|  <--- 管理员浏览器(创建 tenant / 授权 / 生成 frpc 配置)
   +------------------------------+
```

## 监听器

- **管理控制台**(`server.admin_addr`,默认 `0.0.0.0:8080`):提供管理用 REST API
  和内嵌的 Vue SPA。需要登录(Phase 2+)。可在 LAN/VPN 上暴露;请像保护任何
  管理面板那样保护它。
- **frps plugin 端点**(`server.plugin_addr`,默认 `127.0.0.1:9000`,
  路径 `/plugin/frp`):接收 frps 的 server-plugin 钩子调用。**默认仅 loopback**,
  且不得公开暴露(参见 [SECURITY.md](SECURITY.md))。

## 包(Go)

| Package              | 职责                                                        |
|----------------------|-------------------------------------------------------------|
| `cmd/frp-warden`     | 进程入口、flag 解析、组件装配、生命周期。                   |
| `internal/version`   | 构建/版本元信息(可通过 `-ldflags` 覆盖)。                 |
| `internal/config`    | 配置结构体 + 默认值、YAML 加载/生成、校验、数据目录。       |
| `internal/db`        | 打开数据库、方言判定、轻量迁移框架(SQLite 完整)。(Phase 2)|
| `internal/security`  | 密码安全原语:随机密码生成、bcrypt 哈希/校验。(Phase 2)   |
| `internal/bootstrap` | 首次启动初始化:默认管理员创建。(Phase 2)                 |
| `internal/model`     | 领域实体(tenant、资源、授权等)。                          |
| `internal/store`     | tenant/resource/grant/proxy 数据访问与服务层(Phase 3)。    |
| `internal/frpcconfig`| frpc.toml 配置生成器(Phase 8)。                            |
| `internal/admin`     | 管理后台 REST API(Phase 5,session cookie 认证);内嵌前端服务(Phase 7)。 |
| `internal/plugin`    | frps server-plugin 钩子:Login/NewProxy/Ping 真实鉴权,fail-closed。(Phase 4) |
| `internal/server`    | 装配 admin + plugin 的 HTTP 服务器,优雅关闭。              |
| `web/`               | Vue 3 + Vite + TS 前端;构建产物输出到 `web/dist`。         |

## 生命周期

1. **加载配置** —— 从磁盘读取 YAML;若不存在,则生成默认值并写入文件。
   空的 `session_secret` 会被生成并持久化。(Phase 1,已实现)
2. **初始化数据目录** —— 使用 SQLite 时确保数据目录(如 `./data`)存在。(Phase 1,已实现)
3. **打开数据库 + 迁移** —— 按配置打开驱动并 Ping,运行迁移(`schema_migrations` 控制
   每条只执行一次,失败 fail fast)。(Phase 2,SQLite 已实现)
4. **初始化管理员** —— 在 `admins` 为空时创建初始管理员,密码随机生成、一次性打印,
   仅存 bcrypt hash,`must_change_password=true`。(Phase 2,已实现)
5. **启动服务器** —— admin(管理 API)与 plugin(真实鉴权)监听器启动;阻塞直到收到信号,
   然后优雅关闭。使用 `-version` 打印版本后退出。

admin 接口已接入管理 API(Phase 5,session cookie 认证);plugin 接口已接入真实鉴权
(Phase 4,fail-closed)。

## 配置

YAML,在启动时加载。schema 与默认值见 [CONFIGURATION.md](CONFIGURATION.md)。
`internal/config` 中的 Go 结构体是事实来源(source of truth),必须与该文档保持一致。

## 数据库

默认通过一个纯 Go(免 CGO)驱动使用 SQLite,以保持交叉编译的简单 ——
参见 [DECISIONS/0005-use-sqlite-by-default.md](DECISIONS/0005-use-sqlite-by-default.md)。
mysql/mariadb/postgresql 通过同一套 `internal/db` 抽象支持:本轮(Phase 2)已注册其驱动、
可建立连接,但**迁移尚未实现**(执行迁移时 fail fast 报错)。数据访问基于标准库
`database/sql` + 手写迁移,不使用 ORM(见
[DECISIONS/0007-database-migrations.md](DECISIONS/0007-database-migrations.md))。

## 前端内嵌

Vue 应用构建到 `web/dist`;通过 `tools/sync-web-dist` 同步到 `internal/webui/dist`,
再由 Go `embed` 内嵌到二进制(`//go:embed dist/*`)。`internal/webui/dist` 被
`.gitignore` 忽略,不提交到仓库。

admin server 路由:
- `/api/*` → admin API handler(优先匹配)
- `/` → webui handler(静态资源 + SPA fallback)

开发模式下使用 Vite dev server(端口 5173),`/api` 代理到 Go admin server(`:8080`)。
构建单二进制前需先执行前端构建:

```sh
cd web && npm install && npm run build && cd ..
go run ./tools/sync-web-dist
go build ./cmd/frp-warden
```

## 单二进制打包

刻意避免 CGO,从而使 `GOOS`/`GOARCH` 交叉编译能够为所有目标平台生成静态二进制,
无需 C 工具链。前端被嵌入而非单独分发。
参见 [DECISIONS/0001-use-go-single-binary.md](DECISIONS/0001-use-go-single-binary.md)。

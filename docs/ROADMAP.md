# 路线图

frp-warden 按阶段推进。每个阶段刻意保持窄范围,使工作可评审、文档可同步。
**不要提前实现后续阶段的功能。**

| 阶段 | 标题                                           | 状态          |
|------|------------------------------------------------|---------------|
| 0    | 项目初始化与文档                                | ✅ 已完成      |
| 1    | 配置加载与自动初始化                            | ✅ 已完成      |
| 2    | 数据库连接、迁移框架与管理员初始化              | ✅ 已完成      |
| 3    | tenant / resource / grant / proxy 数据模型      | ✅ 已完成      |
| 4    | frps plugin 的 Login/NewProxy/Ping 强制校验     | ✅ 已完成      |
| 5    | 管理 API                                        | ✅ 已完成      |
| 6    | Vue 3 管理后台                                  | ✅ 已完成      |
| 7    | 将前端内嵌进 Go 二进制                          | ✅ 已完成      |
| 8    | frpc 配置生成                                   | ✅ 已完成      |
| 9    | GitHub Actions 多平台发布构建(VERSION 控制)     | ✅ 已完成      |
| 10   | 测试、文档与发布准备                            | ✅ 已完成      |

## 阶段详情

### Phase 0 — 项目初始化与文档(已完成)
模块、带 fail-closed 占位的最小骨架、完整文档体系、`.gitignore`、CI 构建草案、
前端脚手架说明。**无业务逻辑。**

### Phase 1 — 配置加载与自动初始化(已完成)
实现真实 YAML 配置加载;首次运行自动生成默认 `config.yaml`;`session_secret` 为空时
用 `crypto/rand` 生成并写回;根据 SQLite DSN 创建数据目录;基础配置校验;`-c`/`-config`
命令行参数。已接入 `cmd/frp-warden` 替换原内存默认值。本阶段不连接数据库、不引入
SQLite 驱动、不真正启动 HTTP 服务。

### Phase 2 — 数据库连接、迁移框架与管理员初始化(已完成)
基于配置打开数据库(SQLite 用纯 Go 驱动 `modernc.org/sqlite`,无 CGO);实现轻量
迁移框架(`schema_migrations` 控制每条迁移只执行一次,失败 fail fast);创建基础表
`admins`/`settings`/`sessions`/`audit_logs`;首次启动在空库上创建默认管理员
(密码强随机生成、一次性打印、仅存 bcrypt hash,`must_change_password=true`)。
MySQL/MariaDB/PostgreSQL 仅完成驱动与连接骨架,迁移尚未实现(fail fast)。
本阶段不实现登录 API、不实现 tenant/resource/grant/proxy 业务。

### Phase 3 — tenant / resource / grant / proxy 数据模型(已完成)
新增迁移 v2,创建 `tenants`/`domain_zones`/`resources`/`resource_grants`/`proxies`
五张表,并落实唯一性约束:`tenants.code` 唯一、`resources(type,value)` 唯一、
**`resource_grants.resource_id` 唯一(一个资源只属于一个租户)**、`proxies(tenant_id,name)` 唯一、
`domain_zones.zone` 唯一。在 `internal/store` 提供 repository/service 层(tenant、
domain_zone、resource、grant、proxy)与明确的领域错误,实现 subdomain 资源依赖 enabled
domain_zone、proxy 类型须与资源类型匹配、proxy 资源须已授权给本租户等业务约束。
本阶段不实现 Web API、不实现 frps plugin 真实鉴权。

### Phase 4 — frps plugin 强制校验(已完成)
在 `internal/plugin` 实现 `/plugin/frp` 的 `Login`/`NewProxy`/`Ping` 真实鉴权:
- **Login**:解析 tenant code + metas.token,`GetTenantByCode` + `security.VerifySecret`
  bcrypt 校验,检查 `status==enabled`,否则 reject。绝不在原因中泄露 token/hash。
- **NewProxy**(核心强制点):再次校验租户与 token;按 proxy_name 取后台
  `ProxyAuthContext`(proxy + resource + grant);强制:proxy 存在且 enabled、proxy_type
  与请求一致、resource available、grant 存在且属于本租户且 enabled、请求的
  subdomain/remote_port 与后台资源严格一致;禁止 custom_domains。客户端无法靠改
  subdomain/remote_port/custom_domains 越权。
- **Ping**:校验身份与 token,更新 `last_seen_at`;disabled tenant 或 token 错误则拒绝。
- 默认 fail-closed;业务拒绝返回 HTTP 200 + `{"reject":true,"reject_reason":"..."}` 以
  兼容 frps plugin 行为;未知 op/JSON 解析错误/超大请求体各返回明确的拒绝或错误码。
- 在 `internal/store` 新增 `UpdateTenantLastSeen` 与 `GetProxyAuthContext`(聚合查询)。
- `internal/security` 新增 `HashSecret`/`VerifySecret` 语义别名(token 用)。
- CloseProxy 暂不启用(ops 目标为 Login/NewProxy/Ping),后续可用于审计/状态更新。
- 26 项表驱动测试覆盖:Login(5 项)、NewProxy http(6 项+3 项 disabled chain)、
  NewProxy tcp/udp(5 项)、Ping(3 项)、HTTP/解析层(4 项)。

### Phase 5 — 管理后台 REST API 与管理员 session(已完成)
在 `internal/admin` 实现管理后台 REST API:
- **认证**:管理员登录(`POST /api/auth/login`,bcrypt 校验)、退出(撤销 session + 清除
  cookie)、获取当前信息(`GET /api/auth/me`)、修改密码(`POST /api/auth/change-password`,
  旧密码校验 + 新密码长度 ≥10 + `must_change_password=false`)。
- **session**:基于安全 cookie(`HttpOnly` + `SameSiteLax`)的 session 认证;session token
  明文只存 cookie,数据库只存 SHA-256 哈希;默认有效期 24 小时。
- **管理 API**:tenant CRUD(创建时自动生成 token、明文只返回一次、数据库只存 bcrypt hash)、
  domain_zone CRUD、resource CRUD(subdomain/tcp_port/udp_port)、grant CRUD(一个资源只能
  授权给一个 tenant)、proxy CRUD(复用 Phase 3 服务层约束)、审计日志查询。
- **审计日志**:登录成功/失败、登出、修改密码、创建/启用/禁用 tenant/resource/grant/proxy 等;
  绝不写入密码/明文 token/session token。
- 默认启动 HTTP 服务(admin + plugin);`-version` 打印版本后退出。
- 27 项测试覆盖:Auth(8 项)、Tenant CRUD(6 项)、Resource/Grant/Proxy API(9 项)、
  Audit(2 项)、HTTP 解析层(4 项)。
- 新增 ADR 0011。

### Phase 6 — Vue 3 管理后台(已完成)
在 `web/` 实现基于 Vue 3 + Vite + TypeScript + Naive UI 的管理后台前端:
- **登录页**:简洁的登录表单,产品名 frp-warden + 中文定位。
- **管理布局**:左侧导航 + 顶部栏 + 内容区;菜单:仪表盘、租户管理、域名区域、资源池、
  授权管理、映射管理、审计日志、账号设置。
- **仪表盘**:当前管理员信息、`must_change_password` 提醒、部署模型说明、快捷操作入口。
- **租户管理**:列表(含状态标签)、创建(弹窗)、禁用/启用(确认)、重置 token(弹窗显示
  plain_token 一次性)。
- **域名区域**:列表、创建、禁用/启用。
- **资源池**:列表(含类型标签)、创建(subdomain/tcp_port/udp_port,subdomain 需选域名区域)。
- **授权管理**:按租户筛选、创建(选租户+资源)、禁用/启用。
- **映射管理**:按租户筛选、创建(选租户+资源+类型+名称+本地地址端口)、禁用/启用。
- **审计日志**:列表(含操作标签)。
- **账号设置**:显示管理员信息、修改密码(旧密码+新密码+确认)。
- **API 客户端**:`fetch` + `credentials:include`;401 自动跳转登录页;错误显示后端 message。
- **路由守卫**:未登录跳转 `/login`;已登录跳转 `/dashboard`;刷新恢复登录状态。
- **安全**:前端不保存 password/token 到 localStorage;plain_token 只通过弹窗一次性展示。
- `web/dist` 已生成(Phase 7 内嵌到 Go 二进制)。
- 新增 ADR 0012。

### Phase 7 — 将前端内嵌进 Go 二进制(已完成)
使用 Go `embed` 将 `web/dist` 内嵌到二进制,实现单一自包含可执行文件:
- `internal/webui`:内嵌前端静态资源(`//go:embed dist/*`)与 SPA fallback handler。
- `tools/sync-web-dist`:跨平台 Go 工具,将 `web/dist` 同步到 `internal/webui/dist`
  (供 embed 使用;该目录被 `.gitignore` 忽略,不提交)。
- admin server 路由:`/api/*` 优先匹配 admin API handler;`/` fallback 到 webui handler
  (静态资源 + SPA fallback)。`/plugin/frp` 不在 admin server 上,由独立的 plugin server
  处理。
- CI workflow 更新:前端构建(npm ci + type-check + build)→ sync-web-dist → Go test →
  多平台 build(内嵌前端)。
- webui 测试:7 项(首页/SPA fallback/静态资源 JS+CSS/空 FS 降级/API 路径不被吞/Content-Type)。
- 新增 ADR 0013。

### Phase 8 — frpc 配置生成(已完成)
根据 tenant 的授权资源(proxy/resource/grant)生成 frpc.toml 配置:
- `internal/frpcconfig`:frpc.toml 生成器;根据 Config 结构生成 TOML,正确转义字符串;
  http/https 使用 subdomain,tcp/udp 使用 remotePort;只生成 enabled proxy/available
  resource/enabled grant;无 proxy 时生成基础配置+中文注释提示。
- **API**:`GET /api/tenants/{id}/frpc-config`(token 占位符模板)、
  `GET /api/tenants/{id}/frpc-config/download`(TOML 文件下载)。
- **创建 tenant / 重置 token**:响应新增 `frpc_config` 字段(含真实 token 的完整配置,
  仅在此刻一次性返回)。
- **前端**:tenant 管理页面新增"frpc 配置"按钮(查看占位符模板)、创建/重置 token 弹窗
  展示 token + 完整 frpc.toml(两个 tab)、复制功能。
- **安全**:数据库不保存明文 token;平时只能生成 token 占位符模板;plain_token 和
  frpc_config 绝不写入审计日志。
- 新增 ADR 0014。

### Phase 9 — GitHub Actions 多平台发布构建(已完成)
- **CI workflow**(`.github/workflows/build.yml`):push 到 dev/main 和 PR 到 main 时运行;
  前端构建 → sync-web-dist → gofmt/vet/test → 多平台编译验证。
- **Release workflow**(`.github/workflows/release.yml`):打 tag `v*` 时自动触发;
  前端构建 → sync-web-dist → 6 平台交叉编译 → 打包(zip/tar.gz) → 生成 checksums.txt
  → 创建 GitHub Release 并上传产物。
- **支持平台**:windows/amd64、windows/386、linux/amd64、linux/386、linux/arm64、linux/arm v7。
- **产物命名**:`frp-warden_${version}_${os}_${arch}.zip`(Windows)/`.tar.gz`(Linux)。
- **版本注入**:ldflags 注入 `Version`/`Commit`/`BuildDate`;`frp-warden -version` 显示。
- **CGO_ENABLED=0**:全部平台无 CGO,纯 Go 交叉编译。
- 新增 ADR 0015。

### Phase 10 — 测试、文档与发布准备(已完成)
- README 面向新用户重写:快速开始、架构图、部署模型、安全提醒、文档导航、License。
- 安装部署文档(docs/INSTALL.md):下载、安装、首次启动、配置、备份。
- frps 配置指南(docs/FRPS_SETUP.md):frps.toml 示例、httpPlugins、DNS 泛域名、常见错误。
- systemd 部署示例(docs/SYSTEMD.md):用户/目录/unit 文件/管理命令。
- 反向代理建议(docs/REVERSE_PROXY.md):Caddy/Nginx 示例、plugin 端口不暴露。
- 端到端冒烟测试文档(docs/SMOKE_TEST.md):完整 14 步测试流程。
- 常见问题(docs/TROUBLESHOOTING.md):忘记密码/token、NewProxy 拒绝、子域名不通等。
- GitHub 仓库与发布说明(docs/GITHUB.md):分支保护、tag 命名、Release 流程。
- 配置示例(config.example.yaml)、frps/frpc 示例(examples/)。
- MIT License、根目录 SECURITY.md。
- 新增 ADR 0016。

## 明确暂不做的事项

- 客户端自选自定义域名 / `custom_domains`。
- 超出 subdomain / TCP 端口 / UDP 端口之外的资源类型。
- 替换或 fork frp。
- frp-warden 自身的高可用 / 集群。

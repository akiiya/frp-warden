# API

frp-warden 暴露两个 HTTP 接口:**管理 API**(供控制台使用)和 **frps plugin
端点**(供 frps 使用)。本文档描述的是预期的形态;其中大部分尚未实现(参见
[ROADMAP.md](ROADMAP.md))。

## 约定

- 管理 API 是位于 `/api` 下、基于 HTTP 的 JSON 接口。
- 管理 API 的认证基于 session-cookie(Phase 2)。除登录外,所有 `/api/*` 路由
  都要求一个已认证的 session。
- plugin 端点遵循 frps server-plugin 协议(而非管理 API 的约定)。参见
  [FRP_PLUGIN.md](FRP_PLUGIN.md)。

## 管理端点(计划中)

| Method | Path                          | Purpose                               | Phase |
|--------|-------------------------------|---------------------------------------|-------|
| GET    | `/healthz`                    | 存活探针(无需认证)。                 | 0 ✅  |
| POST   | `/api/auth/login`             | 管理员登录,设置 session cookie。      | 2     |
| POST   | `/api/auth/logout`            | 使 session 失效。                     | 2     |
| GET    | `/api/auth/me`                | 当前管理员信息。                      | 2     |
| POST   | `/api/auth/password`          | 修改自己的密码。                      | 2     |
| GET    | `/api/tenants`                | 列出 tenant。                         | 5     |
| POST   | `/api/tenants`                | 创建 tenant(仅返回一次 token)。     | 5     |
| GET    | `/api/tenants/{id}`           | tenant 详情。                         | 5     |
| PATCH  | `/api/tenants/{id}`           | 更新(启用/禁用、重命名)。           | 5     |
| POST   | `/api/tenants/{id}/token`     | 轮换 token(仅返回一次 token)。      | 5     |
| GET    | `/api/resources`              | 列出资源。                            | 5     |
| POST   | `/api/resources`              | 创建一个资源。                        | 5     |
| DELETE | `/api/resources/{id}`         | 删除一个资源。                        | 5     |
| GET    | `/api/grants`                 | 列出 grant。                          | 5     |
| POST   | `/api/grants`                 | 将一个资源授予某个 tenant。           | 5     |
| DELETE | `/api/grants/{id}`            | 撤销一个 grant。                      | 5     |
| GET    | `/api/tenants/{id}/frpc`      | 为某个 tenant 生成 frpc 配置。        | 8     |
| GET    | `/api/audit`                  | 列出审计日志条目。                    | 5+    |

说明:

- token 只在创建/轮换时返回**一次**,之后不可再次获取(只存储其哈希)。
- 如果资源已被授予,则创建 grant 必须失败(`resource_id` 唯一性约束——参见
  [DATA_MODEL.md](DATA_MODEL.md))。

## plugin 端点

| Method | Path           | Purpose                                          | Phase |
|--------|----------------|--------------------------------------------------|-------|
| POST   | `/plugin/frp`  | frps 钩子:`Login` / `NewProxy` / `Ping`。       | 4     |

请求/响应遵循 frps server-plugin 信封;参见 [FRP_PLUGIN.md](FRP_PLUGIN.md)。
**已实现真实鉴权**(Phase 4,见 `internal/plugin`):默认 fail-closed,业务拒绝返回
HTTP 200 + `{"reject": true, "reject_reason": "..."}` 以兼容 frps。

## 状态

- `GET /healthz` 已实现(返回服务名称和版本)。
- `POST /plugin/frp` 已实现真实鉴权(Login/NewProxy/Ping)。
- 管理 API 已实现(Phase 5):认证(tenant CRUD、domain_zone、resource、grant、proxy、
  审计日志);基于 session cookie 认证;统一 JSON 响应格式。详见 [ROADMAP.md](ROADMAP.md)。

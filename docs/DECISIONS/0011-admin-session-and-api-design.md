# 0011 — 管理后台 session 与 API 设计

- 状态:已接受
- 日期:2026-06-23

## 背景

Phase 5 需要实现管理后台 REST API 与管理员 session 认证。需要在轻量、安全、可测试
之间取得平衡。

## 决策与理由

### 为什么使用 session cookie 而不是 JWT

JWT 是无状态的,但管理后台更需要可控的会话管理:logout 能即时撤销、过期时间可精确
控制、不需要在每个请求中携带完整的用户信息。session cookie + 数据库 session 表更
适合管理后台的"登录→操作→登出"模式。JWT 适合 API-to-API 的无状态调用,不适合
管理后台的交互式操作。

### 为什么 session token 明文只存 cookie,数据库只存 hash

session token 是高熵随机值,明文只存在于浏览器 cookie 中(通过 HttpOnly 保护),
数据库只保存 SHA-256 哈希。即使数据库泄露,攻击者也无法直接使用 session token。
与密码的 bcrypt 区分:session token 是高熵随机值,SHA-256 足够且比 bcrypt 快得多;
密码低熵必须用 bcrypt。

### 为什么 cookie 必须 HttpOnly + SameSite

HttpOnly 防止 JavaScript 访问 cookie(XSS 防护);SameSite=Lax 防止跨站请求伪造
(CSRF 防护)。两者都是标准的 cookie 安全属性,不需要额外的 CSRF token 机制。

### 为什么 API 不返回 hash/secret

密码哈希、session_secret、token_hash 都是敏感信息,不应暴露给前端。即使前端不需要
这些值,泄露也会增加攻击面。API 响应只包含前端需要的最小信息。

### 为什么 tenant token 明文只显示一次

tenant token 是设备连接 frps 的凭据。创建/重置时明文只返回一次,之后只存 hash。
这样即使数据库泄露,攻击者也无法获取 token 明文;即使前端被入侵,历史 token 也不可获取。

### 为什么本轮只做 REST API,不做前端页面

分阶段推进。Phase 5 先把 API 做扎实(认证、授权、审计),Phase 6 再用 Vue 3 + Naive UI
消费这些 API。先有可靠的 API,再有 UI,避免两者同时变动带来的风险。

### 为什么仍保持标准库 net/http,不引入 Web 框架

当前路由需求简单(一组 RESTful 端点),标准库的 `http.ServeMux` 足够。引入 Gin/Echo
等框架会增加依赖、增加学习成本,且对当前规模的 API 没有明显收益。如果后续路由复杂度
显著增加,可以再评估引入轻量 router。

## 影响

- `internal/admin` 包含全部管理 API handler;`NewHandler(store)` 接收 `*store.Store`。
- `internal/server.New` 同时挂载 admin 和 plugin handler;默认启动两个 HTTP 服务。
- `cmd/frp-warden` 移除 `-serve` 标志,默认启动服务;保留 `-version`。
- session cookie 名称 `fw_session`;SHA-256 哈希存储;默认 24 小时过期。
- 统一 JSON 响应格式:`{"ok":true,"data":{}}` / `{"ok":false,"error":{"code":"...","message":"..."}}`。

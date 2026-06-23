# 0010 — frps plugin 鉴权流程

- 状态:已接受
- 日期:2026-06-12

## 背景

Phase 4 需要实现 frps server plugin 的 `/plugin/frp` 端点的 `Login`/`NewProxy`/`Ping`
真实鉴权,把 Phase 3 落地的 tenant/resource/grant 数据模型转化为真正的授权强制。这是
frp-warden 从"数据地基"到"授权闭环"的关键一步。

## 决策与理由

### 为什么 Login 只做身份校验

Login 的职责是"这个人是谁、密码对不对、是否被禁用",不做资源授权判断。资源授权是
NewProxy 的职责——Login 与 NewProxy 职责分离,各自校验点清晰,也符合 frps 的操作语义。
Login 校验 tenant code + metas.token(bcrypt 常量时间比对)+ status==enabled 即可。

### 为什么 NewProxy 是最关键的资源授权校验点

frpc Login 成功不代表它可以随意创建 proxy。NewProxy 才是"某个 proxy 能不能跑起来"的
唯一关卡。它必须再次校验租户与 token(不信任 Login 的结果),并强制:proxy 必须在后台
登记且 enabled、proxy_type 与请求一致、所用资源已 available 授权给本租户且 grant
enabled、请求的 subdomain/remote_port 与后台资源严格一致。这些条件缺一不可。

### 为什么 frpc 配置不可信

frpc 运行在设备端,配置文件可被随意修改。客户端可以请求任意 proxy_name、subdomain、
remote_port、custom_domains。因此真正的权限边界绝不能放在客户端,而必须由服务端数据库
(Phase 3 的 grant 关系)与 frps plugin 校验(Phase 4)决定。客户端靠修改这些字段越权
是被禁止的——NewProxy 会逐字段与后台资源比对。

### 为什么 NewProxy 需要再次校验 token

即使 Login 成功,攻击者也可能在 Login 之后篡改后续请求。因此 NewProxy 必须重新执行
完整的身份校验(tenant code + token + status),确保每个 proxy 创建请求都是经过验证的。

### 为什么禁止 custom_domains

custom_domains 允许客户端声明任意域名,会绕过"资源池授权"模型:如果客户端可以自由
指定域名,它就不需要通过 grant 机制获取资源,整个授权闭环就失效了。因此前期一律拒绝
custom_domains,直到该特性被明确设计并纳入资源管理。

### 为什么 Ping 要更新 last_seen_at 并处理 disabled tenant

Ping 是 frps 周期性的心跳。更新 last_seen_at 让运维知道哪些设备在线;更重要的是,
Ping 会重新校验 tenant 的 enabled 状态——如果管理员在设备连接期间禁用了该 tenant,
Ping 拒绝会让 frps 尽快断开连接,实现"禁用即时生效"而无需等待重连。

### 为什么业务拒绝使用 reject=true 而不是 HTTP 500

frps server plugin 的协议设计是:HTTP 200 + `{"reject": true/ false}` 来表示业务结果,
HTTP 500/400 只用于解析/传输层错误。业务拒绝(如凭据错误、资源未授权)不是服务器故障,
而是鉴权的正常结果,应返回 200 + reject=true。这样 frps 可以正确区分"插件出错"与
"请求被拒绝"。

### 为什么 plugin 默认 fail-closed

任何未明确放行的请求都应被拒绝。未知 op、JSON 解析失败、内部错误——所有这些情况
都返回 reject 而非 allow。宁可误拒也不能误放。Phase 0 的占位处理器就遵循这一原则,
Phase 4 的真实鉴权继承并强化了它。

### 为什么 CloseProxy 暂不启用

当前 frps.toml 的 ops 配置为 `["Login", "NewProxy", "Ping"]`,不包含 CloseProxy。
CloseProxy 在当前授权闭环中不是必须的——proxy 的生命周期由 NewProxy 开启、由
frps 连接断开自然结束。CloseProxy 后续可用于审计(记录 proxy 关闭事件)或状态更新,
但不是 Phase 4 的范围。

## 影响

- `internal/plugin` 从占位拒绝处理器升级为真实鉴权处理器;`NewHandler` 接收
  `*store.Store`。
- `internal/store` 新增 `UpdateTenantLastSeen` 与 `GetProxyAuthContext`(聚合查询)。
- `internal/security` 新增 `HashSecret`/`VerifySecret` 语义别名。
- `internal/server.New` 签名变更(新增 `*store.Store` 参数)。
- 业务拒绝以 HTTP 200 + reject=true 返回;解析错误以 HTTP 400 返回;请求体过大以
  HTTP 413 返回;reject_reason 绝不包含 token/hash/secret。
- 26 项表驱动测试覆盖全部鉴权路径与边界情况。

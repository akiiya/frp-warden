# 安全

frp-warden 是一个授权控制平面,因此它自身的安全态势也是产品的一部分。下列规则
是不变量——代码与评审都必须遵守它们。

## 认证与密钥

- **不得硬编码任何凭据。**不存在 `admin/admin`。初始管理员账户使用一个配置的
  用户名(默认 `admin`)和一个在首次初始化时**随机生成的密码**。
- **初始密码只展示一次。**它在初始化时打印到 stdout/日志,且从不以明文存储。
  运维人员应当在此时记录下来。
- **首次登录后请修改密码。**控制台应在首次登录时提示/要求修改密码。
- **密码仅以哈希形式存储。**当前使用 `bcrypt`(`bcrypt.DefaultCost`),封装在
  `internal/security`,为将来切换 argon2id 预留空间。绝不存储或记录明文密码。
- 新建默认管理员的 `must_change_password` 为 `true`,要求其首次登录后修改密码。
- **tenant token 仅以哈希形式存储。**token 的明文只在创建/轮换时展示一次
  (以便填入 frpc 配置),之后不可再次获取。
- **session secret** 在首次启动时若为空则生成,并持久化到配置文件中(`0600` 权限)。
  请将配置文件视为敏感信息。
- **不在日志/启动摘要中泄露敏感信息。**启动摘要与日志绝不打印 `session_secret`、
  `password_hash`、`token_hash` 或数据库密码;默认管理员明文密码只在创建时一次性打印,
  不写入配置文件或日志文件。

## 管理控制台

- 管理控制台需要登录;所有管理端点都位于认证之后(`/api/healthz` 和 `POST /api/auth/login`
  是仅有的公开路由)。
- **session**:基于安全 cookie(`HttpOnly` + `SameSiteLax`);session token 明文只存
  cookie,数据库只存 SHA-256 哈希;默认有效期 24 小时;logout 撤销 session。
- **密码修改**:必须提供旧密码;新密码长度 ≥10;修改后 `must_change_password=false`。
- **tenant token**:创建/重置时明文只返回一次,数据库只存 bcrypt hash。
- **审计日志**:记录登录成功/失败、登出、修改密码、创建/启用/禁用 tenant/resource/grant/proxy;
  绝不写入密码/明文 token/session token。
- 适当地保护管理监听地址(LAN/VPN、带 TLS 的反向代理)。尽管它可以绑定
  `0.0.0.0`,但不建议在没有 TLS/防护的情况下将其公开暴露。

## plugin 端点

- frps plugin 端点**默认监听 `127.0.0.1:9000`**,**不得**暴露到公网。任何
  能够访问它的人都可以影响为 frps 作出的授权决策。
- 如果 frps 运行在另一台主机上,请通过可信的私有网络或隧道连接——绝不要使用
  公网绑定。

## 授权模型

- **所有授权都在服务端强制执行**,即在 frp-warden 中针对数据库执行。决策从不
  委托给客户端。
- **frpc 客户端配置不可信。**客户端可以请求任意 proxy/subdomain/端口;只有一个
  明确的 grant 才能为其授权。
- **`NewProxy` 是关键的强制执行点。**每一个被请求的 subdomain 或公网端口都必须
  映射到一个已授予发起请求的 tenant 的资源。任何未授予的一律拒绝。(参见
  [FRP_PLUGIN.md](FRP_PLUGIN.md)。)
- **资源是单一归属的。**一个资源只能授予一个 tenant,从而防止端点冲突和跨 tenant 劫持
  (参见 [DATA_MODEL.md](DATA_MODEL.md))。这一点由数据库层的唯一约束强制:Phase 3 已落实
  `resource_grants.resource_id` 唯一约束,即使业务代码出错也无法把同一资源授权给两个 tenant;
  `internal/store` 同时做主动校验并返回明确的领域错误(如"资源已被授权")。
- **proxy 创建受服务端约束。**`internal/store` 在创建 proxy 时强制:所用资源必须已 enabled
  授权给该 tenant、proxy 类型必须与资源类型匹配(http/https↔subdomain、tcp↔tcp_port、
  udp↔udp_port)、同一 tenant 内 proxy 名唯一、`local_port` 在 1-65535。这些是 Phase 4 plugin
  鉴权的数据基础。
- **禁用 tenant 会快速生效。**`Login` 和 `Ping` 会重新检查 tenant 的启用状态,
  使得被禁用 tenant 的现有连接被断开。

## Plugin 鉴权(Phase 4,已实现)

`internal/plugin` 已实现 `/plugin/frp` 的真实鉴权,默认 fail-closed:

- **Login**:tenant code + token 身份校验(密码学常量时间 bcrypt 比对);user/token 缺失、
  租户不存在、token 错误均返回模糊提示(不区分"不存在"与"错误");disabled tenant 明确告知。
- **NewProxy**:再次校验租户与 token(不信任 Login 结果);强制:proxy 存在且 enabled、
  proxy_type 与请求一致、resource available、grant 存在且属于本租户且 enabled;
  请求的 subdomain/remote_port 必须与后台资源严格一致;**禁止 custom_domains**。
  客户端无法靠修改 subdomain/remote_port/custom_domains 越权。
- **Ping**:校验身份与 token→更新 `last_seen_at`;disabled tenant 拒绝。
- **CloseProxy**:暂不启用(ops 目标为 Login/NewProxy/Ping)。
- `reject_reason` 以中文为主,**绝不包含 token / hash / session_secret**。
- 业务拒绝返回 HTTP 200 + `{"reject": true}` 以兼容 frps plugin 行为;解析级错误返回
  HTTP 400;请求体过大返回 HTTP 413。

## 失败即拒绝(Fail closed)

存疑时,一律拒绝。plugin 处理器默认拒绝,因此一个未完整接线的部署不会意外授权任何东西。
即使 Login 成功,NewProxy 仍会再次校验租户与 token。

## 构建 / 供应链

- 无 CGO、依赖最小化(更小的攻击面、可复现的交叉构建)。在加入新依赖前先对其
  进行审查。

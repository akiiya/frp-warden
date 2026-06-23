# frps server plugin 集成

frp-warden 通过 **frps server plugin** 机制与 frps 集成:
frps 在约定的操作节点向 frp-warden 发起 HTTP 调用,frp-warden 据此回答是否
允许、拒绝或改写该操作。frps **不会**把 frp-warden 作为 Go 插件加载——两者是
通过 HTTP 通信的独立进程。

参考:frp 的 `[[httpPlugins]]` / server-plugin 管理钩子。

## frps 配置

在 `frps.toml` 中加入:

```toml
[[httpPlugins]]
name = "frp-warden"
addr = "127.0.0.1:9000"
path = "/plugin/frp"
ops = ["Login", "NewProxy", "Ping"]
```

- `addr` 必须指向 frp-warden 的 plugin 监听地址(`server.plugin_addr`)。
- `path` 必须与 frp-warden 提供的路径一致(`/plugin/frp`)。
- `ops` 列出 frps 会转发的操作。

frps 会向该路径 POST 一个 JSON 信封 `{"version", "op", "content"}`。插件回复
`{"reject", "reject_reason", "unchange"}`(并可选地返回改写后的 `content`)。

## 强制执行的操作

### Login

当一个 frpc 客户端连接时触发。

- 针对数据库校验所提供的身份(tenant 的 `user` / `metas`)和 **token**
  (token 按哈希比对)。
- 如果 tenant 未知、token 不匹配,或 tenant 处于 `disabled` 状态,则拒绝。

### NewProxy —— 关键的强制执行点

每次客户端尝试注册一个 proxy 时触发。**授权真正在此处强制执行。**针对发起请求的
tenant,校验:

- `proxy_name` —— 该 tenant 是否被允许使用(且在该 tenant 内唯一)。
- `proxy_type` —— 是否为被允许的类型。
- `subdomain` —— 如果是 HTTP/HTTPS proxy,请求的 subdomain 必须是**已授予该
  tenant**的资源。
- `remote_port` —— 如果是 TCP/UDP proxy,请求的公网端口必须是**已授予该
  tenant**的资源。

凡是未明确授予的一律拒绝。由于 frpc 客户端配置不可信,客户端可以请求任何东西
——只有 grant 检查才能作出裁决。

### Ping

由已连接的客户端周期性触发。

- 更新该 tenant 的 `last_seen`。
- 重新检查该 tenant 是否仍处于启用状态;通过拒绝来让**禁用某个 tenant 对已经
  连接的客户端立即生效**,而无需等待重连。

## 自定义域名

早期阶段**不**允许客户端使用 `custom_domains`。在该特性被明确设计并加入之前,
NewProxy 必须拒绝自定义域名请求(目前它是一个非目标——参见
[PROJECT_CONTEXT.md](PROJECT_CONTEXT.md))。

## 部署与网络暴露

frps 与 frp-warden 都部署在同一台公网服务器 / VPS(或同一内网可达),frps 通过本机
回环调用 frp-warden 的 plugin 接口。因此该 plugin 端点**默认仅监听 `127.0.0.1:9000`**,
**绝不能暴露到公网**——任何能访问它的对象都会参与授权决策。随身 WiFi / Debian 等设备
只运行 frpc 客户端,不运行 frps、也不运行 frp-warden,因此不会、也不应直接访问 plugin
接口。参见 [SECURITY.md](SECURITY.md) 与 [PROJECT_CONTEXT.md](PROJECT_CONTEXT.md) 的部署拓扑。

## 状态

`/plugin/frp` 上的处理器已实现**真实鉴权**(Phase 4,见 `internal/plugin`)。默认
fail-closed:任何解析失败、未知 op、内部错误都放行;业务拒绝返回 HTTP 200 +
`{"reject": true, "reject_reason": "..."}` 以兼容 frps plugin 行为。`reject_reason`
以中文为主,绝不包含 token / hash / session_secret。

- **Login**:tenant code + token 身份校验。
- **NewProxy**:再次校验租户与 token,强制 proxy 存在且 enabled、类型一致、资源已授权
  给本租户且 available、subdomain/remote_port 与后台资源严格一致;禁止 custom_domains。
- **Ping**:校验身份与 token,更新 `last_seen_at`;disabled tenant 拒绝。
- **CloseProxy**:暂不启用(ops 目标为 Login/NewProxy/Ping),后续可用于审计/状态更新。
- 审计日志:Login/NewProxy/Ping 成功/失败的审计写入计划在后续阶段实现(见
  [ROADMAP.md](ROADMAP.md)),绝不写入 token 明文。

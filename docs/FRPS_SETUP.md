# frps 配置指南

本文档说明如何配置 frps 以配合 frp-warden 使用。

## 部署模型

frps 与 frp-warden 部署在**同一台公网服务器/VPS** 上(或同一内网可达)。
随身 WiFi / Debian 等设备**只运行 frpc 客户端**,不运行 frps。

## frps.toml 示例

完整示例见 [examples/frps.toml](../examples/frps.toml)。

```toml
# frps 控制端口,frpc 客户端连接此端口。
bindPort = 7000

# HTTP 虚拟主机端口。
vhostHTTPPort = 80

# HTTPS 虚拟主机端口(可选)。
# vhostHTTPSPort = 443

# subdomain 泛域名根。必须与 frp-warden 配置的 frp.subdomain_host 一致。
subdomainHost = "frp.example.com"

# frps server plugin 配置。
[[httpPlugins]]
name = "frp-warden"
addr = "127.0.0.1:9000"
path = "/plugin/frp"
ops = ["Login", "NewProxy", "Ping"]
```

## 关键配置说明

### bindPort

frps 控制端口,frpc 客户端通过此端口与 frps 建立连接。默认 7000。

### vhostHTTPPort

HTTP 虚拟主机端口。当访问 `http://xxx.frp.example.com` 时,请求通过此端口进入 frps,
再转发给对应的 frpc 客户端。

- 如果服务器 80 端口空闲,直接用 80。
- 如果前面有 Nginx/Caddy 反代,可以用非标准端口(如 8080),由反代转发到此端口。

### vhostHTTPSPort

HTTPS 虚拟主机端口(可选)。如果需要 HTTPS 访问子域名,取消注释并配置。

### subdomainHost

泛域名根。必须与 frp-warden 配置中的 `frp.subdomain_host` 一致。

例如设置为 `frp.example.com`,则 tenant 可使用 `xxx.frp.example.com` 形式的子域名。

### httpPlugins

frps server plugin 配置,让 frps 在关键操作节点调用 frp-warden 进行授权校验。

| 字段 | 值 | 说明 |
|---|---|---|
| `name` | `frp-warden` | 插件名称 |
| `addr` | `127.0.0.1:9000` | frp-warden plugin 接口地址(仅回环) |
| `path` | `/plugin/frp` | 请求路径 |
| `ops` | `["Login", "NewProxy", "Ping"]` | 转发的操作 |

**为什么 `addr` 建议 `127.0.0.1`**:plugin 接口参与授权决策,暴露到公网会导致安全风险。
frps 与 frp-warden 同机部署时,回环地址即可。

## DNS 泛域名解析

需要为 `subdomainHost` 配置 DNS 泛域名解析:

```
*.frp.example.com  →  你的公网服务器 IP
```

在 DNS 服务商后台添加:

| 类型 | 主机记录 | 记录值 |
|---|---|---|
| A | `*` | `你的服务器IP` |

或使用 CNAME:

| 类型 | 主机记录 | 记录值 |
|---|---|---|
| CNAME | `*` | `frp.example.com` |

## 为什么不配置公共 auth.token

frps 支持配置 `auth.token` 用于客户端认证。但 frp-warden 通过 plugin 机制为每个 tenant
独立校验 token,不需要在 frps 层面配置公共 token。如果同时配置了 frps 的 `auth.token`,
frpc 也需要配置相同的 token,会增加配置复杂度。

## 常见错误

### plugin 地址连不上

```
[plugin.go:xxx] http plugin frp-warden connect error: dial tcp 127.0.0.1:9000: connectex: ...
```

原因:frp-warden 未启动,或 plugin_addr 不匹配。

解决:确认 frp-warden 已启动,且 `server.plugin_addr` 与 frps.toml 的 `httpPlugins.addr` 一致。

### subdomainHost 没配

```
[NewProxy] subdomain xxx not allowed
```

原因:frps.toml 未配置 `subdomainHost`,或与 frp-warden 的 `frp.subdomain_host` 不一致。

解决:在 frps.toml 中添加 `subdomainHost = "frp.example.com"`,并确保与 frp-warden 配置一致。

### DNS 泛域名没配

访问 `xxx.frp.example.com` 无法解析。

原因:DNS 未配置泛域名解析。

解决:在 DNS 后台添加 `*.frp.example.com` 的 A 记录或 CNAME。

### token 错误

```
[plugin.go:xxx] Login failed: 租户不存在或凭据无效
```

原因:frpc 配置中的 `user` 或 `metadatas.token` 不正确。

解决:在 WebUI 中确认 tenant code,并重置 token 获取新的 frpc 配置。

### NewProxy 被拒绝

```
[plugin.go:xxx] NewProxy rejected: 该资源未授权给此租户
```

原因:frpc 请求的 subdomain 或 remotePort 未在 frp-warden 中授权给该 tenant。

解决:在 WebUI 中为该 tenant 授权对应的资源,并创建 proxy 映射。

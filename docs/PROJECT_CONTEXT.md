# 项目背景

## frp-warden 是什么

frp-warden 是一个**面向 frp 的多 tenant 授权控制平面**。它并不替代
[frp](https://github.com/fatedier/frp);它运行在 frps 旁边,负责管控*谁被允许暴露什么*。
frps 继续负责隧道转发;frp-warden 掌管策略:tenant、资源以及授权。

## 为什么需要它

普通的 frps 部署共用一个 token(或一组静态 token),并且会信任客户端请求的
任意 subdomain/端口。当面对大量彼此独立的设备/用户、且每个都必须隔离并限制到
特定资源时,这种方式无法扩展。frp-warden 引入了一个控制平面,使得:

- frps 只需部署一次,即可被所有人复用,
- 每个设备拥有自己的身份和 token,
- 每个设备只能使用管理员授予的那些 subdomain/端口,
- 同一个 subdomain/端口绝不会被分配给两个设备,
- 接入一个新设备只需几次点击(创建 tenant → 授予资源 →
  生成 frpc 配置),而不必修改服务端配置再重新部署。

## 核心模型

- **1 个设备 = 1 个用户 = 1 个 tenant。** 在本系统中这三者是同一个实体。
  tenant 是身份与隔离的基本单位。
- **每个 tenant 拥有自己的 token。** 该 token 用于把设备的 frpc 认证到
  frps(通过 frp-warden 校验)。token 仅以哈希形式存储。
- **tenant 只能使用被授权的资源。** 默认不允许任何访问;只有存在显式授权时才允许访问。

### Tenant

一个 tenant 代表一台设备/一个用户。它拥有唯一的 `code`(用作 frp 用户身份)、
一个 token(仅哈希)、一个启用/禁用状态,以及一个最后在线时间戳。

### 资源(Resource)

资源是 frp 可以暴露的、可分配且可授权的单元。类型包括:

- **subdomain** —— 一个 HTTP/HTTPS subdomain(frp 的 `subdomain`),
- **TCP 端口** —— 一个对外的远程 TCP 端口,
- **UDP 端口** —— 一个对外的远程 UDP 端口,
- *(未来)* **自定义域名** —— 一个完整的自定义域名,早期阶段刻意不纳入范围。

资源由其 `(type, value)` 组合唯一标识。

### 授权(Grant)

一条授权把**一个资源绑定到一个 tenant**。它就是授权记录。如果某个 tenant
对某资源没有授权,该 tenant 便无法使用它。

### 为什么资源不能授予多个 tenant

一个对外的 subdomain 或端口在 frps 上是全局唯一的路由目标。如果同一个
subdomain/端口被授予两个 tenant,它们的隧道就会冲突、争抢同一个对外端点 ——
轻则路由结果不确定,重则一个 tenant 劫持另一个 tenant 的流量。为了让路由保持
明确、并维持 tenant 隔离,**每个资源最多只有一个归属的 tenant**。这一点通过对
授权记录 `resource_id` 的唯一性约束来强制实现(参见
[DATA_MODEL.md](DATA_MODEL.md))。

## 职责划分

- **frps** 只负责转发。它部署一次,除了每次请求中获知的信息外,并不感知
  tenant、资源或授权。
- **frp-warden** 掌管授权。frps 通过 HTTP 向它询问每一次
  `Login` / `NewProxy` / `Ping` 是否被允许。
- **frps 不会以插件库的形式加载 frp-warden**。两者的集成走的是 frps 的
  **server-plugin** HTTP 机制:frps 向 frp-warden 的 plugin 端点发起 HTTP 调用。
  它们是相互独立的进程。参见 [FRP_PLUGIN.md](FRP_PLUGIN.md)。

## 部署拓扑

frp-warden 与 frps 都部署在**正规的公网服务器 / VPS** 上;随身 WiFi / Debian 设备
**只作为客户端运行 frpc**,既不运行 frps,也不运行 frp-warden。

- **公网服务器 / VPS**:同时运行 frps(对外提供控制端口与公网入口)与 frp-warden
  (管理后台 + plugin 接口)。两者在同一台主机、或同一内网可达;frps 通过 HTTP 调用
  frp-warden 的 plugin 接口。
- frp-warden 的 `plugin_addr` **默认 `127.0.0.1:9000`,只允许本机 frps 调用**,不暴露公网。
- **随身 WiFi / Debian 设备(客户端)**:只运行 frpc。运维在 frp-warden 管理页面创建
  tenant、授权资源、生成 frpc 配置,再把该配置复制到设备上启动;设备凭 tenant 的 token
  连接 frps。

也就是说:策略与授权集中在公网服务器上的 frp-warden,设备端只是一个携带 token 的 frpc
客户端,其本地配置不可信(真正的权限边界由服务端数据库与 frps plugin 校验决定)。

## 非目标(当前阶段)

- 为客户端提供自定义域名(推迟)。
- 允许客户端在 frpc 配置中自由选择 `custom_domains`。
- 替代或 fork frp。

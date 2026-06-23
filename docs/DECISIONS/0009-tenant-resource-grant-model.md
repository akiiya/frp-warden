# 0009 — tenant / resource / grant 核心授权模型

- 状态:已接受
- 日期:2026-06-12

## 背景

frp-warden 的本质是多租户授权控制面:它必须明确"哪个设备能用哪些公网资源",
并在数据层把这种关系表达清楚、约束牢固,供后续 Web API 与 frps plugin 校验复用。
本 ADR 记录 Phase 3 在数据模型与服务约束上的关键决策。

## 决策与理由

### 为什么"1 设备 = 1 用户 = 1 租户"

每台随身 WiFi / Debian 设备对应一个 `tenant`,tenant 是身份与隔离的基本单位,持有
独立的 token(对应 frpc 的 user)。把设备、用户、租户三者合一,模型最简单、隔离边界最清晰:
授权、配额、启用/禁用都以 tenant 为粒度,不需要额外的用户↔设备多对多关系。

### 为什么资源必须独立建模

公网 subdomain、TCP 端口、UDP 端口是稀缺且全局唯一的路由目标。把它们建模为独立的
`resources`,而不是直接写在 proxy 或 tenant 上,才能:统一表达"系统里存在哪些可分配资源"、
用 `(type, value)` 唯一保证不重复登记、并让"授权"成为一条可独立管理与审计的关系记录。

### 为什么 `resource_grants.resource_id` 必须唯一

这是多租户资源隔离的关键。一个公网 subdomain/端口若被授权给两个 tenant,其隧道会在 frps
上争抢同一对外端点,导致路由不确定甚至跨租户流量劫持。因此"一个资源只能授权给一个 tenant"
必须由**数据库唯一约束**强制(`resource_grants.resource_id` UNIQUE),而不能只靠业务代码判断——
业务校验可能因并发或代码缺陷失效,数据库约束是最终兜底。服务层同时主动校验并返回明确错误。

### 为什么 subdomain 资源依赖 domain_zones

subdomain 资源只有挂在一个真实可路由的泛域名区域(如 `*.frp.example.com`)下才有意义。
用 `domain_zones` 显式声明系统允许使用哪些区域,并要求创建 subdomain 资源前必须存在
enabled 的区域、且资源绑定到具体 zone,可避免凭空创建无法落地的子域名,也为后续生成完整
访问域名(subdomain 前缀 + 去掉 `*.` 的 zone)提供依据。

### 为什么 frpc 配置不可信

frpc 运行在设备端,其配置文件可被随意修改。客户端可以请求任意 proxy 名、subdomain、端口。
因此真正的权限边界绝不能放在客户端,而必须由服务端数据库(本轮的 grant 关系)与 frps plugin
校验(Phase 4)决定。Phase 3 的服务层已对 proxy 创建强制"资源须已授权给本租户、类型须匹配"。

### 为什么本轮只实现数据模型,不实现 plugin 鉴权

分阶段推进、保持每轮可评审。Phase 3 先把数据模型、唯一约束与服务层约束做扎实并测试覆盖;
Phase 4 的 frps plugin Login/NewProxy/Ping 真实鉴权将直接复用这些数据与校验逻辑。先有可信的
数据地基,再在其上做鉴权,避免鉴权逻辑与数据模型同时变动带来的风险。

### 为什么 frps / frp-warden 部署在公网服务器,随身 WiFi 只运行 frpc

frps 需要公网可达才能对外提供入口;frp-warden 负责授权决策,必须能被 frps 通过本机回环
低延迟、可信地调用。因此二者部署在同一台公网服务器 / VPS(或同一内网),frp-warden 的
`plugin_addr` 默认仅监听 `127.0.0.1`、不暴露公网。随身 WiFi / Debian 设备只是携带 token 的
frpc 客户端,不运行 frps、也不运行 frp-warden——既降低设备端复杂度,也把授权集中在受控的
服务器侧。

## 影响

- 五张表(tenants / domain_zones / resources / resource_grants / proxies)与其唯一约束
  写入 Phase 3 迁移 v2;新增迁移只能追加版本号。
- 服务层(`internal/store`)返回明确的领域错误,供 Phase 4 plugin 与 Phase 5 API 复用。
- 端口资源按 `tcp_port` / `udp_port` 区分;若未来要让 TCP/UDP 端口完全互斥,可再演进为统一
  port 类型,当前以 `(type, value)` 唯一保证同类型不重复。

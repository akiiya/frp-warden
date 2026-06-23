# 0014 — frpc 配置生成

- 状态:已接受
- 日期:2026-06-23

## 背景

Phase 8 需要根据 tenant 的授权资源(proxy/resource/grant)生成 frpc.toml 配置,
方便运维复制到设备上启动 frpc。需要在用户体验与安全之间取得平衡。

## 决策与理由

### 为什么 frpc 配置由服务端生成

frpc 配置涉及多个实体的聚合:tenant code、proxy 类型与本地地址、resource 类型与值、
grant 状态。手动编写容易出错(如 subdomain 拼错、端口写反),且无法保证与数据库一致。
服务端生成可保证配置与授权状态完全一致,减少人工错误。

### 为什么数据库不保存明文 tenant token

tenant token 是设备连接 frps 的凭据。如果数据库保存明文,数据库泄露即意味着所有
tenant 的 token 泄露。只保存 bcrypt hash,即使数据库泄露也无法反推 token 明文。
这是安全红线,不为任何便利性妥协。

### 为什么平时只能生成 token 占位符模板

因为数据库不保存明文 token,服务端无法从 hash 反推明文。因此平时查看 frpc 配置时,
只能生成 token 占位符模板(如 `metadatas.token = "请粘贴创建/重置时显示的一次性 token"`)。
这是数据库只存 hash 的必然结果。

### 为什么创建/重置 token 时可以返回完整 frpc.toml

创建 tenant 或重置 token 时,明文 token 刚刚生成,仍在内存中。此时可以将其注入
frpc.toml 并一次性返回给管理员。这是唯一可以获取含真实 token 的完整配置的时机。
之后该 token 不可再获取。

### 为什么不生成 customDomains

custom_domains 允许客户端声明任意域名,会绕过"资源池授权"模型。当前阶段一律禁止
custom_domains(见 [FRP_PLUGIN.md](FRP_PLUGIN.md)),frpc 配置中也不生成该字段。

### 为什么只生成 enabled proxy / enabled grant / available resource

禁用的 proxy、禁用的授权、禁用的资源不应出现在 frpc 配置中。如果配置中包含已禁用的
条目,frpc 启动后会尝试使用它们,但 frps 的 NewProxy 校验会拒绝,导致混乱。只生成
当前有效的条目,保证配置可直接使用。

### 为什么本轮不做 Release workflow

Release 自动化属于 Phase 9,需要多平台构建矩阵、产物上传、GitHub Releases 集成。
这与 frpc 配置生成是独立的功能,不应混在同一个阶段。

### 为什么随身 WiFi 只拿 frpc 配置,不运行 frps/frp-warden

frps 和 frp-warden 部署在公网服务器/VPS 上,负责隧道转发和授权控制。随身 WiFi / Debian
设备是客户端,只运行 frpc,携带 token 连接 frps。运维在管理页面创建 tenant、授权资源、
生成 frpc 配置后,复制到设备上启动即可。

## 影响

- `internal/frpcconfig`:frpc.toml 生成器,不访问数据库,方便单元测试。
- `internal/store`:新增 `ListEnabledProxyAuthContextsByTenant`。
- `internal/admin`:新增 frpc config API 端点;创建/重置 token 响应新增 `frpc_config`。
- 前端 TenantsView.vue:新增"frpc 配置"按钮、创建/重置弹窗展示 token + 完整配置。
- 安全:plain_token 和 frpc_config 绝不写入审计日志、数据库、localStorage。

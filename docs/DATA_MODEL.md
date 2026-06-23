# 数据模型

本文档描述各实体、关系,以及必须成立的唯一性约束。

落地状态:全部实体均已真实建表(见 `internal/db/migrate.go`):`admins`、`settings`、
`sessions`、`audit_logs` 来自 **Phase 2** 迁移 v1;`tenants`、`domain_zones`、`resources`、
`resource_grants`、`proxies` 来自 **Phase 3** 迁移 v2。下文字段与迁移保持一致。
领域类型定义在 `internal/model`,repository/service 在 `internal/store`。

## 实体

### admins

管理控制台的操作者。

- `id`
- `username` —— **唯一**
- `password_hash` —— bcrypt 哈希(未来可换 argon2id);**非空**、**绝不**明文
- `must_change_password` —— 默认 `true`,要求首次登录后修改密码
- `status` —— **非空**,`enabled` / `disabled`
- `created_at`, `updated_at`
- `last_login_at` —— 最近登录时间(可空)

### tenants

一个 tenant = 一台设备 = 一个用户。每台随身 WiFi / Debian 设备对应一个 tenant。

- `id`
- `code` —— 短而稳定的标识符,对应 frpc 的 user;**唯一**、**非空**
- `name` —— 人类可读的标签,**非空**
- `token_hash` —— tenant token 的哈希;**非空**、**绝不**明文(明文只在创建/重置时显示一次)
- `status` —— **非空**,`enabled` / `disabled`
- `description` —— 描述(默认空串)
- `created_at`, `updated_at`
- `last_seen_at` —— 由 Ping 更新(可空)

### domain_zones

顶级域名区域,声明系统允许使用的泛域名区域(例如 `*.frp.example.com`)。
只有存在 `enabled` 的区域,才允许创建 subdomain 资源。

- `id`
- `name` —— 区域名称,**非空**
- `zone` —— 泛域名区域(如 `*.frp.example.com`);**唯一**、**非空**
- `status` —— **非空**,`enabled` / `disabled`
- `created_at`, `updated_at`

### resources

frp 可以暴露的、可分配且可授权的单元。

- `id`
- `type` —— **非空**,`subdomain` | `tcp_port` | `udp_port`(未来:`custom_domain`)
- `value` —— **非空**;subdomain 仅保存前缀(如 `ufi001`),端口类型保存端口号字符串(如 `61001`)
- `domain_zone_id` —— subdomain 资源指向所属 `domain_zones`;端口类型为空
- `status` —— **非空**,`available` / `disabled`
- `created_at`, `updated_at`
- 约束:`(type, value)` **唯一**;subdomain 必须绑定 enabled 的 `domain_zone`

### resource_grants

授权记录。每条把一个资源授权给一个 tenant。

- `id`
- `tenant_id` —— 引用 `tenants`,**非空**
- `resource_id` —— 引用 `resources`,**唯一**、**非空**
- `status` —— **非空**,`enabled` / `disabled`
- `created_at`, `updated_at`
- 约束:`resource_id` **唯一**(一个资源只属于一个 tenant);`(tenant_id, resource_id)` 唯一

### proxies

某个 tenant 的映射配置:用某个已授权资源暴露某个本地服务(即客户端实际运行的内容)。

- `id`
- `tenant_id` —— 引用 `tenants`,**非空**
- `resource_id` —— 该 proxy 使用的资源,引用 `resources`,**非空**
- `name` —— proxy 名称,**非空**
- `proxy_type` —— **非空**,`http` | `https` | `tcp` | `udp`
- `local_ip` —— **非空**,默认 `127.0.0.1`
- `local_port` —— **非空**,取值 1-65535(业务层校验)
- `status` —— **非空**,`enabled` / `disabled`
- `created_at`, `updated_at`
- 约束:`(tenant_id, name)` **唯一**;proxy_type 必须与资源 type 匹配;资源必须已授权给该 tenant

### sessions

已登录的管理员会话。本轮(Phase 2)仅建表,不实现登录逻辑。

- `id` —— 不透明的 session id
- `admin_id` —— 引用 `admins`
- `token_hash` —— session token 的 hash;**绝不**明文(未来登录时填充)
- `expires_at`
- `created_at`
- `revoked_at` —— 撤销时间(可空)

### audit_logs

安全相关的操作轨迹。本轮(Phase 2)仅建表,为后续审计做准备。

- `id`
- `actor_type` —— **非空**,行为主体类型(如 `admin`、`tenant`、`system`)
- `actor_id` —— 行为主体 id(可空)
- `action` —— **非空**,例如 `tenant.create`、`grant.create`、`plugin.newproxy.reject`
- `target_type` —— 目标类型(可空)
- `target_id` —— 目标 id(可空)
- `message` —— 描述信息(可空)
- `ip` —— 来源 IP(可空)
- `created_at`

### settings

持久化的运行时键/值设置(如初始化状态 `initial_admin_created_at`)。

- `key` —— **唯一**(主键)
- `value`
- `created_at`
- `updated_at`

## 关系

```
admins  1 ──< sessions

tenants 1 ──< proxies
tenants 1 ──< resource_grants >── 1 resources
domain_zones 1 ──< resources (subdomain type)
```

- 一个 tenant 拥有多个 proxy 和多条授权。
- 一个资源最多属于一条授权(一个归属的 tenant)。
- 一个 subdomain 资源属于一个 domain zone。

## 唯一性约束(必须强制)

| Constraint                                   | 含义                                               |
|----------------------------------------------|----------------------------------------------------|
| `tenants.code` unique                        | tenant code 全局唯一。                             |
| `resources (type, value)` unique             | 给定的 subdomain/端口只存在一份。                  |
| `resource_grants.resource_id` unique         | 一个资源**只能**授予一个 tenant。                  |
| `proxies (tenant_id, name)` unique           | proxy 名称在**同一** tenant 内唯一。               |
| `resource_grants (tenant_id, resource_id)` unique | 同一 tenant 与 resource 的授权关系不重复。    |
| `domain_zones.zone` unique                   | 每个顶级域名区域只注册一次。                       |
| `settings.key` unique                        | 每个设置键对应一个值。                             |

`resource_grants.resource_id` 的唯一性是对
[PROJECT_CONTEXT.md](PROJECT_CONTEXT.md) 中"一个资源 → 一个 tenant"规则的技术
层面强制实现。它保证不会有两个 tenant 声明同一个对外的 subdomain 或端口。

// Package model 定义 frp-warden 的核心领域实体与状态常量。
//
// 这些类型由 internal/store 的 repository/service 层使用并作为返回值,
// 字段与 internal/db 的迁移保持一致;数据模型说明见 docs/DATA_MODEL.md。
//
// 安全不变量(由 store 业务校验 + 数据库唯一约束共同保证,见 docs/SECURITY.md):
//   - Tenant.Code 全局唯一。
//   - Resource 按 (Type, Value) 唯一。
//   - 一个 Resource 至多授权给一个 Tenant(ResourceGrant.ResourceID 唯一)。
//   - Proxy.Name 在同一 Tenant 内唯一。
//   - 密码与 token 仅以 hash 存储,绝不存明文。
package model

import "time"

// 资源类型(对应 resources.type)。
const (
	ResourceTypeSubdomain = "subdomain"
	ResourceTypeTCPPort   = "tcp_port"
	ResourceTypeUDPPort   = "udp_port"
)

// proxy 类型(对应 proxies.proxy_type)。
const (
	ProxyTypeHTTP  = "http"
	ProxyTypeHTTPS = "https"
	ProxyTypeTCP   = "tcp"
	ProxyTypeUDP   = "udp"
)

// 通用启用状态,用于 tenants / domain_zones / resource_grants / proxies。
const (
	StatusEnabled  = "enabled"
	StatusDisabled = "disabled"
)

// 资源状态,用于 resources。
const (
	ResourceStatusAvailable = "available"
	ResourceStatusDisabled  = "disabled"
)

// DefaultLocalIP 是 proxy 未显式指定本地 IP 时的默认值。
const DefaultLocalIP = "127.0.0.1"

// Admin 是管理后台操作员,仅存储密码 hash。
type Admin struct {
	ID                 int64
	Username           string
	PasswordHash       string
	MustChangePassword bool
	Status             string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	LastLoginAt        *time.Time
}

// Tenant 与设备/用户 1:1 对应。每个 tenant 拥有独立 token,仅以 hash 存储。
type Tenant struct {
	ID          int64
	Code        string // 全局唯一,对应 frpc 的 user
	Name        string
	TokenHash   string // 仅存 hash,绝不存明文 token
	Status      string // enabled / disabled
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	LastSeenAt  *time.Time
}

// DomainZone 是顶级域名区域(如 *.frp.example.com),用于分配 subdomain 资源。
type DomainZone struct {
	ID        int64
	Name      string
	Zone      string // 唯一,例如 *.frp.example.com
	Status    string // enabled / disabled
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Resource 是可分配、可授权的公网资源(subdomain 或公网端口),按 (Type, Value) 唯一。
type Resource struct {
	ID           int64
	Type         string // subdomain / tcp_port / udp_port
	Value        string // subdomain 前缀,或端口号字符串
	DomainZoneID *int64 // subdomain 资源指向所属顶级域名区域;端口类型为空
	Status       string // available / disabled
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ResourceGrant 将一个 Resource 授权给一个 Tenant。ResourceID 唯一,保证一个资源只属于一个租户。
type ResourceGrant struct {
	ID         int64
	TenantID   int64
	ResourceID int64 // 唯一
	Status     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Proxy 是租户的映射配置:用某个已授权资源暴露某个本地服务。Name 在同一 tenant 内唯一。
type Proxy struct {
	ID         int64
	TenantID   int64
	ResourceID int64
	Name       string
	ProxyType  string // http / https / tcp / udp
	LocalIP    string
	LocalPort  int
	Status     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Session 是已登录的管理员会话(本轮仅建表,未实现登录)。
type Session struct {
	ID        string
	AdminID   int64
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
	RevokedAt *time.Time
}

// AuditLog 记录与安全相关的操作,供后续审查。
type AuditLog struct {
	ID         int64
	ActorType  string
	ActorID    *int64
	Action     string
	TargetType string
	TargetID   string
	Message    string
	IP         string
	CreatedAt  time.Time
}

// Setting 是持久化在数据库中的单个键值运行时设置。
type Setting struct {
	Key       string
	Value     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

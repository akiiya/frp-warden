package store

import "errors"

// 领域错误。错误信息以中文为主,技术字段名保留英文。
// 这些错误供测试、Web API 与 frps plugin 校验复用,避免向上层抛出含糊的数据库错误。
var (
	// ErrEmptyField 表示必填字段为空。
	ErrEmptyField = errors.New("必填字段不能为空")

	// 管理员相关。
	ErrAdminNotFound = errors.New("管理员不存在")

	// session 相关。
	ErrSessionNotFound = errors.New("session 不存在或已失效")

	// 租户相关。
	ErrTenantNotFound = errors.New("租户不存在")
	ErrTenantDisabled = errors.New("租户已禁用")
	ErrDuplicateCode  = errors.New("租户 code 已存在")

	// 顶级域名区域相关。
	ErrDomainZoneNotFound  = errors.New("顶级域名区域不存在")
	ErrDuplicateZone       = errors.New("顶级域名区域 zone 已存在")
	ErrNoEnabledDomainZone = errors.New("没有可用的顶级域名区域（domain_zone），无法创建 subdomain 资源")
	ErrSubdomainNeedsZone  = errors.New("subdomain 资源必须绑定一个 enabled 的顶级域名区域")
	ErrDomainZoneDisabled  = errors.New("顶级域名区域已禁用")

	// 资源相关。
	ErrResourceNotFound    = errors.New("资源不存在")
	ErrResourceDisabled    = errors.New("资源已禁用")
	ErrDuplicateResource   = errors.New("相同 type 与 value 的资源已存在")
	ErrInvalidPort         = errors.New("端口范围非法（应在 1-65535）")
	ErrInvalidResourceType = errors.New("非法的资源类型")

	// 授权相关。
	ErrResourceAlreadyGranted = errors.New("该资源已被授权给某个租户，不能重复授权")
	ErrGrantNotFound          = errors.New("授权记录不存在")

	// 映射(proxy)相关。
	ErrProxyNotFound              = errors.New("proxy 不存在")
	ErrInvalidProxyType           = errors.New("非法的 proxy 类型")
	ErrProxyTypeResourceMismatch  = errors.New("proxy 类型与资源类型不匹配")
	ErrResourceNotGrantedToTenant = errors.New("该资源未授权给此租户")
	ErrDuplicateProxyName         = errors.New("同一租户下已存在同名 proxy")
)

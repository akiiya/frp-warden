// Package store 实现 frp-warden 的核心授权数据层:租户(tenant)、顶级域名区域
// (domain_zone)、公网资源(resource)、资源授权(resource_grant)与映射配置(proxy)
// 的 repository/service 逻辑。
//
// 设计:基于标准库 database/sql,不使用 ORM(见
// docs/DECISIONS/0007-database-migrations.md);本轮(Phase 3)只实现数据模型与服务
// 约束,不实现 Web API、不实现 frps plugin 真实鉴权(留待 Phase 4)。
//
// 关键安全约束既在数据库层用唯一约束强制(见 internal/db 的迁移),也在服务层做主动
// 校验并返回明确错误(见 errors.go)。SQL 占位符使用 "?",当前面向 SQLite。
package store

import (
	"database/sql"
	"strings"
)

// Store 封装底层数据库连接,聚合各实体的数据访问与服务方法。
type Store struct {
	db *sql.DB
}

// New 用给定的 *sql.DB 构造 Store。调用方负责该连接的生命周期。
func New(db *sql.DB) *Store {
	return &Store{db: db}
}

// isUniqueViolation 判断错误是否为 SQLite 唯一约束冲突,用于把底层错误转换为明确的领域错误。
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// validPort 校验端口是否在合法范围 1-65535。
func validPort(port int) bool {
	return port >= 1 && port <= 65535
}

// Package db 负责按配置打开数据库连接、判定方言并执行轻量迁移。
//
// 设计原则:不使用 ORM,基于标准库 database/sql + 手写迁移即可(见
// docs/DECISIONS/0007-database-migrations.md)。
//
// 数据库支持成熟度(本轮 Phase 2):
//   - sqlite:完整可用(纯 Go 驱动 modernc.org/sqlite,无 CGO),迁移已实现,测试覆盖。
//   - mysql / mariadb:仅连接骨架,迁移尚未实现,执行迁移时 fail fast 报错。
//   - postgresql:仅连接骨架,迁移尚未实现,执行迁移时 fail fast 报错。
//
// 之所以注册了 mysql/postgresql 驱动却暂不实现其迁移,是为了让"按 driver 选择
// 连接"这一骨架先就位;真正的多数据库迁移留待后续阶段,绝不写"看似支持实则不支持"
// 的半成品(见 docs/DECISIONS/0007)。
package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/fengheasia/frp-warden/internal/config"

	// 数据库驱动(匿名导入以完成 database/sql 注册)。
	_ "github.com/go-sql-driver/mysql" // 注册名 "mysql"(mysql / mariadb 共用)
	_ "github.com/jackc/pgx/v5/stdlib" // 注册名 "pgx"(postgresql)
	_ "modernc.org/sqlite"             // 注册名 "sqlite"(纯 Go,无 CGO)
)

// Dialect 表示数据库方言,用于在迁移与 SQL 语法层面区分不同数据库。
type Dialect string

const (
	// DialectSQLite 为 SQLite,本轮完整实现。
	DialectSQLite Dialect = "sqlite"
	// DialectMySQL 为 MySQL/MariaDB,本轮仅连接骨架。
	DialectMySQL Dialect = "mysql"
	// DialectPostgres 为 PostgreSQL,本轮仅连接骨架。
	DialectPostgres Dialect = "postgresql"
)

// ResolveDialect 将配置中的 driver 名映射为内部方言。
// mariadb 复用 mysql 方言与驱动。未知 driver 返回错误。
func ResolveDialect(driver string) (Dialect, error) {
	switch driver {
	case "sqlite":
		return DialectSQLite, nil
	case "mysql", "mariadb":
		return DialectMySQL, nil
	case "postgresql":
		return DialectPostgres, nil
	default:
		return "", fmt.Errorf("db: 不支持的数据库 driver %q（允许 sqlite/mysql/mariadb/postgresql）", driver)
	}
}

// Open 根据配置打开数据库连接,完成连通性检查(Ping),并返回连接与方言。
//
// 调用方负责在使用完毕后 Close 返回的 *sql.DB。
func Open(cfg config.DatabaseConfig) (*sql.DB, Dialect, error) {
	dialect, err := ResolveDialect(cfg.Driver)
	if err != nil {
		return nil, "", err
	}

	var driverName, dsn string
	switch dialect {
	case DialectSQLite:
		driverName, dsn = "sqlite", sqliteDSN(cfg.DSN)
	case DialectMySQL:
		driverName, dsn = "mysql", cfg.DSN
	case DialectPostgres:
		driverName, dsn = "pgx", cfg.DSN
	}

	sdb, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, "", fmt.Errorf("db: 打开数据库失败（driver=%s）: %w", cfg.Driver, err)
	}

	// SQLite 是单文件,并发写会触发 "database is locked"。
	// 将最大打开连接数限制为 1,以串行化写入、提升稳健性。
	if dialect == DialectSQLite {
		sdb.SetMaxOpenConns(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sdb.PingContext(ctx); err != nil {
		_ = sdb.Close()
		return nil, "", fmt.Errorf("db: 连接数据库失败（driver=%s）: %w", cfg.Driver, err)
	}

	return sdb, dialect, nil
}

// sqliteDSN 处理 SQLite 的 DSN:
//   - 若调用方已自带查询参数(含 "?"),原样返回,尊重其显式配置;
//   - 否则追加默认 pragma:开启外键约束,并设置忙等待,降低锁冲突。
func sqliteDSN(dsn string) string {
	dsn = strings.TrimSpace(dsn)
	if strings.Contains(dsn, "?") {
		return dsn
	}
	return dsn + "?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
}

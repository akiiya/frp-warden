package db

import (
	"context"
	"database/sql"
	"fmt"
)

// migration 表示一条带版本号的迁移。Statements 中的语句会在同一事务内按序执行。
type migration struct {
	Version    int
	Name       string
	Statements []string
}

// sqliteMigrations 是 SQLite 的迁移列表,按 Version 升序。新增迁移请追加新的版本号,
// 不要修改既有迁移(其是否已执行由 schema_migrations 控制)。
var sqliteMigrations = []migration{
	{
		Version: 1,
		Name:    "initial schema",
		Statements: []string{
			// admins:管理员表。username 唯一;password_hash 非空(仅存 hash);
			// status 非空(enabled/disabled);must_change_password 默认 1(true)。
			`CREATE TABLE IF NOT EXISTS admins (
				id                   INTEGER PRIMARY KEY AUTOINCREMENT,
				username             TEXT    NOT NULL UNIQUE,
				password_hash        TEXT    NOT NULL,
				must_change_password INTEGER NOT NULL DEFAULT 1,
				status               TEXT    NOT NULL DEFAULT 'enabled',
				created_at           TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at           TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				last_login_at        TIMESTAMP
			)`,

			// settings:系统设置键值表,用于保存初始化状态及未来系统级设置。
			`CREATE TABLE IF NOT EXISTS settings (
				key        TEXT PRIMARY KEY,
				value      TEXT NOT NULL,
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
			)`,

			// sessions:管理员会话表。本轮仅建表,不实现登录逻辑。
			// token_hash 未来保存 session token 的 hash,绝不存明文。
			`CREATE TABLE IF NOT EXISTS sessions (
				id         TEXT PRIMARY KEY,
				admin_id   INTEGER NOT NULL,
				token_hash TEXT    NOT NULL,
				expires_at TIMESTAMP NOT NULL,
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				revoked_at TIMESTAMP,
				FOREIGN KEY (admin_id) REFERENCES admins(id)
			)`,

			// audit_logs:审计日志表。本轮仅建表,为后续审计做准备。
			`CREATE TABLE IF NOT EXISTS audit_logs (
				id          INTEGER PRIMARY KEY AUTOINCREMENT,
				actor_type  TEXT NOT NULL,
				actor_id    INTEGER,
				action      TEXT NOT NULL,
				target_type TEXT,
				target_id   TEXT,
				message     TEXT,
				ip          TEXT,
				created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
			)`,
		},
	},
	{
		Version: 2,
		Name:    "tenant resource grant proxy",
		Statements: []string{
			// tenants:租户表。每台随身 WiFi / Debian 设备对应一个租户。
			// code 唯一(对应 frpc 的 user);token_hash 仅存 hash,绝不存明文 token;
			// status 非空(enabled/disabled)。
			`CREATE TABLE IF NOT EXISTS tenants (
				id           INTEGER PRIMARY KEY AUTOINCREMENT,
				code         TEXT NOT NULL UNIQUE,
				name         TEXT NOT NULL,
				token_hash   TEXT NOT NULL,
				status       TEXT NOT NULL DEFAULT 'enabled',
				description  TEXT NOT NULL DEFAULT '',
				created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				last_seen_at DATETIME
			)`,

			// domain_zones:顶级域名区域表,声明系统允许使用的泛域名区域(如 *.frp.example.com)。
			// zone 唯一;只有存在 enabled 的区域,才允许创建 subdomain 资源。
			`CREATE TABLE IF NOT EXISTS domain_zones (
				id         INTEGER PRIMARY KEY AUTOINCREMENT,
				name       TEXT NOT NULL,
				zone       TEXT NOT NULL UNIQUE,
				status     TEXT NOT NULL DEFAULT 'enabled',
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			)`,

			// resources:公网资源表。type ∈ {subdomain, tcp_port, udp_port}。
			// (type, value) 唯一;subdomain 必须绑定 domain_zone_id,端口类型则为空。
			`CREATE TABLE IF NOT EXISTS resources (
				id             INTEGER PRIMARY KEY AUTOINCREMENT,
				type           TEXT NOT NULL,
				value          TEXT NOT NULL,
				domain_zone_id INTEGER,
				status         TEXT NOT NULL DEFAULT 'available',
				created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				UNIQUE (type, value),
				FOREIGN KEY (domain_zone_id) REFERENCES domain_zones(id)
			)`,

			// resource_grants:资源授权表。核心约束:resource_id 唯一 ——
			// 一个资源只能授权给一个租户(多租户资源隔离的关键,由数据库强制)。
			`CREATE TABLE IF NOT EXISTS resource_grants (
				id          INTEGER PRIMARY KEY AUTOINCREMENT,
				tenant_id   INTEGER NOT NULL,
				resource_id INTEGER NOT NULL UNIQUE,
				status      TEXT NOT NULL DEFAULT 'enabled',
				created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				UNIQUE (tenant_id, resource_id),
				FOREIGN KEY (tenant_id)   REFERENCES tenants(id),
				FOREIGN KEY (resource_id) REFERENCES resources(id)
			)`,

			// proxies:映射配置表。某租户用某个已授权资源暴露某个本地服务。
			// 同一 tenant_id 下 name 唯一;local_port 取值 1-65535(业务层校验)。
			`CREATE TABLE IF NOT EXISTS proxies (
				id          INTEGER PRIMARY KEY AUTOINCREMENT,
				tenant_id   INTEGER NOT NULL,
				resource_id INTEGER NOT NULL,
				name        TEXT NOT NULL,
				proxy_type  TEXT NOT NULL,
				local_ip    TEXT NOT NULL DEFAULT '127.0.0.1',
				local_port  INTEGER NOT NULL,
				status      TEXT NOT NULL DEFAULT 'enabled',
				created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				UNIQUE (tenant_id, name),
				FOREIGN KEY (tenant_id)   REFERENCES tenants(id),
				FOREIGN KEY (resource_id) REFERENCES resources(id)
			)`,
		},
	},
}

// Migrate 按顺序执行尚未应用的迁移,返回本次实际应用的迁移条数。
//
// 行为:
//   - 先确保 schema_migrations 表存在;
//   - 已应用的版本(记录在 schema_migrations 中)会被跳过,因此重复执行安全;
//   - 每条迁移在独立事务中执行,成功后写入 schema_migrations 再提交;
//   - 任意一条迁移失败都会回滚并立即返回错误(fail fast,绝不静默跳过)。
//
// 本轮仅实现 SQLite 的迁移;其它方言会 fail fast 报错(见包文档与
// docs/DECISIONS/0007-database-migrations.md)。
func Migrate(ctx context.Context, sdb *sql.DB, dialect Dialect) (int, error) {
	if dialect != DialectSQLite {
		return 0, fmt.Errorf("db: 当前仅实现 SQLite 的迁移,方言 %q 尚未支持（见 docs/DECISIONS/0007-database-migrations.md）", dialect)
	}

	if _, err := sdb.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    INTEGER PRIMARY KEY,
		name       TEXT NOT NULL,
		applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return 0, fmt.Errorf("db: 创建 schema_migrations 表失败: %w", err)
	}

	applied, err := appliedVersions(ctx, sdb)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, m := range sqliteMigrations {
		if applied[m.Version] {
			continue
		}
		if err := applyMigration(ctx, sdb, m); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// appliedVersions 读取已应用的迁移版本集合。
func appliedVersions(ctx context.Context, sdb *sql.DB) (map[int]bool, error) {
	rows, err := sdb.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("db: 查询 schema_migrations 失败: %w", err)
	}
	defer rows.Close()

	applied := make(map[int]bool)
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("db: 读取迁移版本失败: %w", err)
		}
		applied[v] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("db: 遍历迁移版本失败: %w", err)
	}
	return applied, nil
}

// applyMigration 在单个事务内执行一条迁移并记录到 schema_migrations。
func applyMigration(ctx context.Context, sdb *sql.DB, m migration) error {
	tx, err := sdb.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("db: 开启迁移事务失败（版本 %d %s）: %w", m.Version, m.Name, err)
	}

	for _, stmt := range m.Statements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("db: 迁移失败（版本 %d %s）: %w", m.Version, m.Name, err)
		}
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (version, name) VALUES (?, ?)`, m.Version, m.Name); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("db: 记录迁移失败（版本 %d %s）: %w", m.Version, m.Name, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("db: 提交迁移失败（版本 %d %s）: %w", m.Version, m.Name, err)
	}
	return nil
}

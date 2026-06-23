package db

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/fengheasia/frp-warden/internal/config"
)

func TestResolveDialect(t *testing.T) {
	cases := map[string]Dialect{
		"sqlite":     DialectSQLite,
		"mysql":      DialectMySQL,
		"mariadb":    DialectMySQL,
		"postgresql": DialectPostgres,
	}
	for driver, want := range cases {
		got, err := ResolveDialect(driver)
		if err != nil {
			t.Errorf("ResolveDialect(%q) 返回错误: %v", driver, err)
		}
		if got != want {
			t.Errorf("ResolveDialect(%q) = %q, 期望 %q", driver, got, want)
		}
	}
	if _, err := ResolveDialect("oracle"); err == nil {
		t.Error("非法 driver 应返回错误")
	}
}

func TestOpenAndPingSQLite(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "ping.db")
	sdb, dialect, err := Open(config.DatabaseConfig{Driver: "sqlite", DSN: dsn})
	if err != nil {
		t.Fatalf("Open 失败: %v", err)
	}
	defer sdb.Close()

	if dialect != DialectSQLite {
		t.Errorf("方言 = %q, 期望 sqlite", dialect)
	}
	if err := sdb.PingContext(context.Background()); err != nil {
		t.Errorf("Ping 失败: %v", err)
	}
}

func TestOpenInvalidDriver(t *testing.T) {
	if _, _, err := Open(config.DatabaseConfig{Driver: "oracle", DSN: "x"}); err == nil {
		t.Error("非法 driver 的 Open 应返回错误")
	}
}

func TestMigrateCreatesTables(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "migrate.db")
	sdb, dialect, err := Open(config.DatabaseConfig{Driver: "sqlite", DSN: dsn})
	if err != nil {
		t.Fatalf("Open 失败: %v", err)
	}
	defer sdb.Close()

	n, err := Migrate(context.Background(), sdb, dialect)
	if err != nil {
		t.Fatalf("Migrate 失败: %v", err)
	}
	if n == 0 {
		t.Error("空库首次迁移应至少应用一条迁移")
	}

	wantTables := []string{"schema_migrations", "admins", "settings", "sessions", "audit_logs"}
	for _, tbl := range wantTables {
		var name string
		err := sdb.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, tbl).Scan(&name)
		if err != nil {
			t.Errorf("期望存在表 %q,但查询失败: %v", tbl, err)
		}
	}
}

func TestMigrateIdempotent(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "idem.db")
	sdb, dialect, err := Open(config.DatabaseConfig{Driver: "sqlite", DSN: dsn})
	if err != nil {
		t.Fatalf("Open 失败: %v", err)
	}
	defer sdb.Close()

	ctx := context.Background()
	n1, err := Migrate(ctx, sdb, dialect)
	if err != nil {
		t.Fatalf("首次 Migrate 失败: %v", err)
	}
	n2, err := Migrate(ctx, sdb, dialect)
	if err != nil {
		t.Fatalf("二次 Migrate 失败: %v", err)
	}
	if n1 == 0 {
		t.Error("首次迁移应应用迁移")
	}
	if n2 != 0 {
		t.Errorf("二次迁移不应重复应用,实际应用 %d 条", n2)
	}
}

func TestMigrateNonSQLiteFailsFast(t *testing.T) {
	// 仅验证非 SQLite 方言会 fail fast,不建立真实连接。
	if _, err := Migrate(context.Background(), nil, DialectPostgres); err == nil {
		t.Error("非 SQLite 方言的迁移应 fail fast 报错")
	}
}

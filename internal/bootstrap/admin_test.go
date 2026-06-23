package bootstrap

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/fengheasia/frp-warden/internal/config"
	"github.com/fengheasia/frp-warden/internal/db"
	"github.com/fengheasia/frp-warden/internal/security"
)

// newMigratedDB 在临时目录打开并迁移一个全新的 SQLite 库,供测试使用。
func newMigratedDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "bootstrap.db")
	sdb, dialect, err := db.Open(config.DatabaseConfig{Driver: "sqlite", DSN: dsn})
	if err != nil {
		t.Fatalf("打开数据库失败: %v", err)
	}
	t.Cleanup(func() { _ = sdb.Close() })
	if _, err := db.Migrate(context.Background(), sdb, dialect); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	return sdb
}

func TestEnsureInitialAdminCreates(t *testing.T) {
	sdb := newMigratedDB(t)
	ctx := context.Background()

	res, err := EnsureInitialAdmin(ctx, sdb, "admin")
	if err != nil {
		t.Fatalf("初始化管理员失败: %v", err)
	}
	if !res.Created {
		t.Fatal("空库首次初始化应创建管理员")
	}
	if res.Username != "admin" {
		t.Errorf("用户名 = %q, 期望 admin", res.Username)
	}
	if res.Password == "" {
		t.Error("创建管理员应返回明文密码用于一次性打印")
	}

	// 数据库中应存有一条管理员,且 must_change_password 为 true。
	var (
		username           string
		passwordHash       string
		mustChangePassword int
		status             string
	)
	err = sdb.QueryRow(
		`SELECT username, password_hash, must_change_password, status FROM admins WHERE username=?`, "admin").
		Scan(&username, &passwordHash, &mustChangePassword, &status)
	if err != nil {
		t.Fatalf("查询管理员失败: %v", err)
	}
	if mustChangePassword != 1 {
		t.Errorf("must_change_password = %d, 期望 1(true)", mustChangePassword)
	}
	if status != "enabled" {
		t.Errorf("status = %q, 期望 enabled", status)
	}
	// password_hash 不应是明文密码。
	if passwordHash == res.Password {
		t.Error("password_hash 不应等于明文密码")
	}
	// 存库的 hash 必须能通过 bcrypt 校验返回的明文密码。
	if !security.VerifyPassword(passwordHash, res.Password) {
		t.Error("存库的 password_hash 应能通过 bcrypt 校验明文密码")
	}
}

func TestEnsureInitialAdminSkipsWhenExists(t *testing.T) {
	sdb := newMigratedDB(t)
	ctx := context.Background()

	first, err := EnsureInitialAdmin(ctx, sdb, "admin")
	if err != nil {
		t.Fatalf("首次初始化失败: %v", err)
	}
	if !first.Created {
		t.Fatal("首次应创建管理员")
	}

	second, err := EnsureInitialAdmin(ctx, sdb, "admin")
	if err != nil {
		t.Fatalf("二次初始化失败: %v", err)
	}
	if second.Created {
		t.Error("已存在管理员时不应重复创建")
	}
	if second.Password != "" {
		t.Error("未创建管理员时不应返回密码")
	}

	// 管理员数量应仍为 1。
	var count int
	if err := sdb.QueryRow(`SELECT COUNT(*) FROM admins`).Scan(&count); err != nil {
		t.Fatalf("统计管理员失败: %v", err)
	}
	if count != 1 {
		t.Errorf("管理员数量 = %d, 期望 1", count)
	}
}

package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/fengheasia/frp-warden/internal/config"
	"github.com/fengheasia/frp-warden/internal/db"
	"github.com/fengheasia/frp-warden/internal/model"
)

// newTestStore 在临时目录打开并迁移一个全新的 SQLite 库,返回 Store,避免污染仓库。
func newTestStore(t *testing.T) *Store {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "store.db")
	sdb, dialect, err := db.Open(config.DatabaseConfig{Driver: "sqlite", DSN: dsn})
	if err != nil {
		t.Fatalf("打开数据库失败: %v", err)
	}
	t.Cleanup(func() { _ = sdb.Close() })
	if _, err := db.Migrate(context.Background(), sdb, dialect); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	return New(sdb)
}

// 迁移测试:Phase 3 的 5 张表均被创建,且重复迁移不报错。
func TestMigrationCreatesPhase3Tables(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "m.db")
	sdb, dialect, err := db.Open(config.DatabaseConfig{Driver: "sqlite", DSN: dsn})
	if err != nil {
		t.Fatalf("打开数据库失败: %v", err)
	}
	defer sdb.Close()

	if _, err := db.Migrate(context.Background(), sdb, dialect); err != nil {
		t.Fatalf("首次迁移失败: %v", err)
	}
	// 重复迁移应不报错且不再应用。
	n, err := db.Migrate(context.Background(), sdb, dialect)
	if err != nil {
		t.Fatalf("二次迁移失败: %v", err)
	}
	if n != 0 {
		t.Errorf("二次迁移应用了 %d 条,期望 0", n)
	}

	for _, tbl := range []string{"tenants", "domain_zones", "resources", "resource_grants", "proxies"} {
		var name string
		if err := sdb.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, tbl).Scan(&name); err != nil {
			t.Errorf("期望存在表 %q,查询失败: %v", tbl, err)
		}
	}
}

// 租户:创建、code 唯一、禁用。
func TestTenantCRUD(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	tn, err := s.CreateTenant(ctx, TenantInput{Code: "wifi001", Name: "随身WiFi-001", TokenHash: "hash-1"})
	if err != nil {
		t.Fatalf("创建租户失败: %v", err)
	}
	if tn.ID == 0 || tn.Status != model.StatusEnabled {
		t.Errorf("租户创建结果异常: %+v", tn)
	}

	// code 唯一约束。
	if _, err := s.CreateTenant(ctx, TenantInput{Code: "wifi001", Name: "重复", TokenHash: "hash-2"}); !errors.Is(err, ErrDuplicateCode) {
		t.Errorf("重复 code 应返回 ErrDuplicateCode,实际 %v", err)
	}

	// 必填校验。
	if _, err := s.CreateTenant(ctx, TenantInput{Code: "", Name: "x", TokenHash: "h"}); !errors.Is(err, ErrEmptyField) {
		t.Errorf("空 code 应返回 ErrEmptyField,实际 %v", err)
	}

	// 禁用。
	if err := s.UpdateTenantStatus(ctx, tn.ID, model.StatusDisabled); err != nil {
		t.Fatalf("禁用租户失败: %v", err)
	}
	got, err := s.GetTenantByCode(ctx, "wifi001")
	if err != nil {
		t.Fatalf("按 code 查询失败: %v", err)
	}
	if got.Status != model.StatusDisabled {
		t.Errorf("租户状态 = %q,期望 disabled", got.Status)
	}

	// 不存在。
	if _, err := s.GetTenantByID(ctx, 99999); !errors.Is(err, ErrTenantNotFound) {
		t.Errorf("不存在租户应返回 ErrTenantNotFound,实际 %v", err)
	}
}

// 顶级域名区域:创建、zone 唯一、无 enabled 区域时禁止创建 subdomain 资源。
func TestDomainZone(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	// 还没有任何 enabled 区域时,创建 subdomain 资源应失败。
	if _, err := s.CreateSubdomainResource(ctx, "ufi001", 1); !errors.Is(err, ErrNoEnabledDomainZone) {
		t.Errorf("无 enabled 区域应返回 ErrNoEnabledDomainZone,实际 %v", err)
	}

	z, err := s.CreateDomainZone(ctx, "默认区域", "*.frp.example.com")
	if err != nil {
		t.Fatalf("创建顶级域名区域失败: %v", err)
	}
	if z.Status != model.StatusEnabled {
		t.Errorf("区域状态 = %q,期望 enabled", z.Status)
	}

	// zone 唯一约束。
	if _, err := s.CreateDomainZone(ctx, "重复", "*.frp.example.com"); !errors.Is(err, ErrDuplicateZone) {
		t.Errorf("重复 zone 应返回 ErrDuplicateZone,实际 %v", err)
	}

	has, err := s.HasEnabledDomainZone(ctx)
	if err != nil || !has {
		t.Errorf("应存在 enabled 区域,has=%v err=%v", has, err)
	}
}

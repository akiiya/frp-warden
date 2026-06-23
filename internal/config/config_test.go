package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDefaultValid 验证默认配置合法，且 plugin 默认仅监听回环地址。
func TestDefaultValid(t *testing.T) {
	cfg := Default()

	if err := cfg.Validate(); err != nil {
		t.Fatalf("默认配置应当合法，却返回错误: %v", err)
	}
	// plugin 监听地址必须默认仅绑定回环（见 docs/SECURITY.md）。
	if got := cfg.Server.PluginAddr; got != "127.0.0.1:9000" {
		t.Errorf("plugin_addr 默认值 = %q，期望回环地址 127.0.0.1:9000", got)
	}
	if got := cfg.Database.Driver; got != "sqlite" {
		t.Errorf("database.driver 默认值 = %q，期望 sqlite", got)
	}
	if got := cfg.Database.DSN; got != "./data/frp-warden.db" {
		t.Errorf("database.dsn 默认值 = %q，期望 ./data/frp-warden.db", got)
	}
	// 默认配置不得内置 session_secret。
	if cfg.Security.SessionSecret != "" {
		t.Errorf("session_secret 默认值必须为空，实际为 %q", cfg.Security.SessionSecret)
	}
}

// TestLoadGeneratesDefaultAndSecret 验证配置文件不存在时会生成默认配置，
// 且 session_secret 会被自动生成并写回；再次加载时密钥保持稳定。
func TestLoadGeneratesDefaultAndSecret(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")

	res, err := Load(path)
	if err != nil {
		t.Fatalf("首次 Load 失败: %v", err)
	}
	if !res.Generated {
		t.Error("配置文件不存在时，Generated 应为 true")
	}
	if !res.SecretGenerated {
		t.Error("session_secret 为空时，SecretGenerated 应为 true")
	}
	if res.Config.Security.SessionSecret == "" {
		t.Fatal("加载后 session_secret 不应为空")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("默认配置文件应已写入磁盘: %v", err)
	}

	// 文件中应实际写入了非空 session_secret。
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取生成的配置文件失败: %v", err)
	}
	if strings.Contains(string(raw), "session_secret: \"\"") {
		t.Error("写回的配置文件中 session_secret 不应为空")
	}

	// 再次加载：不应再生成文件或密钥，且密钥保持不变。
	res2, err := Load(path)
	if err != nil {
		t.Fatalf("二次 Load 失败: %v", err)
	}
	if res2.Generated || res2.SecretGenerated {
		t.Error("二次加载不应再生成配置文件或 session_secret")
	}
	if res2.Config.Security.SessionSecret != res.Config.Security.SessionSecret {
		t.Error("二次加载 session_secret 应与首次一致")
	}
}

// TestLoadInvalidDriver 验证非法 database.driver 会报错。
func TestLoadInvalidDriver(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	content := `
server:
  admin_addr: "0.0.0.0:8080"
  plugin_addr: "127.0.0.1:9000"
database:
  driver: "oracle"
  dsn: "./data/frp-warden.db"
security:
  session_secret: "x"
  initial_admin_username: "admin"
frp:
  server_addr: "127.0.0.1"
  server_port: 7000
log:
  level: "info"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("写入测试配置失败: %v", err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("非法 database.driver 应当返回错误")
	}
}

// TestLoadInvalidLogLevel 验证非法 log.level 会报错。
func TestLoadInvalidLogLevel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	content := `
server:
  admin_addr: "0.0.0.0:8080"
  plugin_addr: "127.0.0.1:9000"
database:
  driver: "sqlite"
  dsn: "./data/frp-warden.db"
security:
  session_secret: "x"
  initial_admin_username: "admin"
frp:
  server_addr: "127.0.0.1"
  server_port: 7000
log:
  level: "trace"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("写入测试配置失败: %v", err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("非法 log.level 应当返回错误")
	}
}

// TestLoadInvalidYAML 验证配置文件格式错误时会明确报错，而非静默忽略。
func TestLoadInvalidYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("server: [this is not valid"), 0o600); err != nil {
		t.Fatalf("写入测试配置失败: %v", err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("YAML 格式错误应当返回错误")
	}
}

// TestEnsureDataDir 验证 SQLite DSN 对应的数据目录会被创建。
func TestEnsureDataDir(t *testing.T) {
	base := t.TempDir()
	cfg := Default()
	cfg.Database.DSN = filepath.Join(base, "data", "frp-warden.db")

	if err := EnsureDataDir(cfg); err != nil {
		t.Fatalf("EnsureDataDir 失败: %v", err)
	}
	dir := filepath.Join(base, "data")
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("数据目录应被创建: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("%s 应为目录", dir)
	}
}

// TestEnsureDataDirNonSQLite 验证非 sqlite 驱动时不创建本地目录。
func TestEnsureDataDirNonSQLite(t *testing.T) {
	cfg := Default()
	cfg.Database.Driver = "postgresql"
	cfg.Database.DSN = "postgres://user:pass@127.0.0.1:5432/frpwarden"

	if err := EnsureDataDir(cfg); err != nil {
		t.Fatalf("非 sqlite 驱动时 EnsureDataDir 不应报错: %v", err)
	}
}

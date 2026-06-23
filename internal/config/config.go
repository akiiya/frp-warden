// Package config 定义 frp-warden 的配置结构、默认值与校验逻辑。
//
// 配置以单个 YAML 文件承载，结构体上的 yaml tag 是配置键名的唯一来源，
// 必须与 docs/CONFIGURATION.md 保持同步：任何字段变更都要同时更新该文档。
//
// 本包负责：默认配置、YAML 加载/生成（见 load.go）、配置校验、
// 以及根据 SQLite DSN 创建数据目录。数据库连接与迁移属于 Phase 2，本包不涉及。
package config

import (
	"errors"
	"fmt"
	"strings"
)

// Config 是 frp-warden 的顶层配置。
//
// 其 YAML 布局见 docs/CONFIGURATION.md，两者必须保持一致。
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Security SecurityConfig `yaml:"security"`
	FRP      FRPConfig      `yaml:"frp"`
	Log      LogConfig      `yaml:"log"`
}

// ServerConfig 控制两个 HTTP 监听地址。
//
// AdminAddr 提供管理后台 UI/API，可在局域网内暴露；
// PluginAddr 提供 frps server plugin 接口，默认必须仅监听回环地址
// （见 docs/SECURITY.md）。
type ServerConfig struct {
	AdminAddr  string `yaml:"admin_addr"`
	PluginAddr string `yaml:"plugin_addr"`
}

// DatabaseConfig 选择并配置后端存储。
//
// Driver 取值之一：sqlite（默认）、mysql、mariadb、postgresql。
// 早期阶段仅实现 sqlite，其余为后续阶段预留。
type DatabaseConfig struct {
	Driver string `yaml:"driver"`
	DSN    string `yaml:"dsn"`
}

// SecurityConfig 保存鉴权与会话相关设置。
//
// SessionSecret 为空时，首次启动会自动生成强随机值并写回配置文件（见 load.go）。
// InitialAdminUsername 仅是用户名；初始管理员密码在初始化时随机生成、绝不写入配置
// （仅以 hash 存入数据库，见 docs/SECURITY.md）。
type SecurityConfig struct {
	SessionSecret        string `yaml:"session_secret"`
	InitialAdminUsername string `yaml:"initial_admin_username"`
}

// FRPConfig 保存生成 frpc 客户端配置时使用的参数。
type FRPConfig struct {
	ServerAddr    string `yaml:"server_addr"`
	ServerPort    int    `yaml:"server_port"`
	SubdomainHost string `yaml:"subdomain_host"`
}

// LogConfig 控制日志行为。
type LogConfig struct {
	Level string `yaml:"level"`
}

// Default 返回带有文档约定默认值的 Config。
//
// 这些默认值刻意与 docs/CONFIGURATION.md 保持一致。注意 plugin 监听地址
// 默认仅绑定回环地址。
func Default() Config {
	return Config{
		Server: ServerConfig{
			AdminAddr:  "0.0.0.0:8080",
			PluginAddr: "127.0.0.1:9000",
		},
		Database: DatabaseConfig{
			Driver: "sqlite",
			DSN:    "./data/frp-warden.db",
		},
		Security: SecurityConfig{
			SessionSecret:        "",
			InitialAdminUsername: "admin",
		},
		FRP: FRPConfig{
			ServerAddr:    "127.0.0.1",
			ServerPort:    7000,
			SubdomainHost: "",
		},
		Log: LogConfig{
			Level: "info",
		},
	}
}

// allowedDrivers 是当前允许的数据库驱动集合。
var allowedDrivers = map[string]bool{
	"sqlite":     true,
	"mysql":      true,
	"mariadb":    true,
	"postgresql": true,
}

// allowedLogLevels 是当前允许的日志级别集合。
var allowedLogLevels = map[string]bool{
	"debug": true,
	"info":  true,
	"warn":  true,
	"error": true,
}

// Validate 对配置做基础合法性校验。配置非法时返回明确的错误，启动应据此中止。
func (c Config) Validate() error {
	if strings.TrimSpace(c.Server.AdminAddr) == "" {
		return errors.New("config: server.admin_addr 不能为空")
	}
	if strings.TrimSpace(c.Server.PluginAddr) == "" {
		return errors.New("config: server.plugin_addr 不能为空")
	}
	if strings.TrimSpace(c.Database.Driver) == "" {
		return errors.New("config: database.driver 不能为空")
	}
	if !allowedDrivers[c.Database.Driver] {
		return fmt.Errorf("config: database.driver 非法 %q，允许的取值为 sqlite/mysql/mariadb/postgresql", c.Database.Driver)
	}
	if strings.TrimSpace(c.Database.DSN) == "" {
		return errors.New("config: database.dsn 不能为空")
	}
	if strings.TrimSpace(c.Security.InitialAdminUsername) == "" {
		return errors.New("config: security.initial_admin_username 不能为空")
	}
	if strings.TrimSpace(c.FRP.ServerAddr) == "" {
		return errors.New("config: frp.server_addr 不能为空")
	}
	if c.FRP.ServerPort <= 0 {
		return fmt.Errorf("config: frp.server_port 必须大于 0，当前为 %d", c.FRP.ServerPort)
	}
	if !allowedLogLevels[c.Log.Level] {
		return fmt.Errorf("config: log.level 非法 %q，允许的取值为 debug/info/warn/error", c.Log.Level)
	}
	return nil
}

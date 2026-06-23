package config

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// DefaultConfigPath 是未显式指定时使用的默认配置文件路径。
const DefaultConfigPath = "./config.yaml"

// sessionSecretBytes 是自动生成 session_secret 时使用的随机字节数。
// 至少 32 字节熵，使用 crypto/rand 生成（绝不使用 math/rand）。
const sessionSecretBytes = 32

// configFileHeader 写入自动生成配置文件顶部的提示注释。
const configFileHeader = "# frp-warden 配置文件（由程序自动生成，可手动编辑）\n" +
	"# 字段说明见 docs/CONFIGURATION.md\n" +
	"# 注意：plugin_addr 默认仅监听 127.0.0.1，请勿暴露到公网。\n\n"

// LoadResult 描述一次配置加载的结果，便于启动时输出清晰的日志。
type LoadResult struct {
	Config Config // 最终生效的配置
	Path   string // 实际使用的配置文件路径

	// Generated 表示本次因配置文件不存在而自动生成了默认配置文件。
	Generated bool
	// SecretGenerated 表示本次自动生成了 session_secret 并写回了配置文件。
	SecretGenerated bool
}

// Load 从 path 加载配置，实现 frp-warden 的零配置首启动逻辑：
//
//  1. 文件不存在：使用默认配置，并在补全 session_secret 后写入该文件；
//  2. 文件存在：读取并解析，缺省字段沿用 Default() 的默认值；
//  3. 解析失败：返回明确错误（不静默忽略）；
//  4. session_secret 为空：使用 crypto/rand 生成强随机值并写回文件；
//  5. 最终配置经过 Validate 校验，非法时返回错误。
func Load(path string) (*LoadResult, error) {
	if strings.TrimSpace(path) == "" {
		path = DefaultConfigPath
	}

	res := &LoadResult{Path: path}
	cfg := Default()

	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		// 以默认值为底进行反序列化，文件中未出现的键沿用默认值。
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("config: 解析配置文件 %s 失败: %w", path, err)
		}
	case errors.Is(err, fs.ErrNotExist):
		// 文件不存在：标记为自动生成，后续写回到磁盘。
		res.Generated = true
	default:
		return nil, fmt.Errorf("config: 读取配置文件 %s 失败: %w", path, err)
	}

	// session_secret 为空则自动生成强随机值。
	if strings.TrimSpace(cfg.Security.SessionSecret) == "" {
		secret, err := generateSessionSecret()
		if err != nil {
			return nil, err
		}
		cfg.Security.SessionSecret = secret
		res.SecretGenerated = true
	}

	// 在写回磁盘之前完成校验，避免把非法配置落盘。
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// 配置文件新建，或本次生成了 session_secret，都需要写回磁盘。
	if res.Generated || res.SecretGenerated {
		if err := Save(cfg, path); err != nil {
			return nil, err
		}
	}

	res.Config = cfg
	return res, nil
}

// Save 将配置以 YAML 写入 path，并在文件顶部附带说明注释。
//
// 配置可能包含 session_secret，属于敏感信息，因此文件权限设为 0600。
func Save(cfg Config, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("config: 序列化配置失败: %w", err)
	}

	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("config: 创建配置目录 %s 失败: %w", dir, err)
		}
	}

	out := append([]byte(configFileHeader), data...)
	if err := os.WriteFile(path, out, 0o600); err != nil {
		return fmt.Errorf("config: 写入配置文件 %s 失败: %w", path, err)
	}
	return nil
}

// EnsureDataDir 在使用 sqlite 时，根据 DSN 推断并创建数据目录。
//
// 本轮（Phase 1）只创建目录，不连接数据库、不做迁移、不引入 SQLite 驱动，
// 这些都属于 Phase 2。
func EnsureDataDir(cfg Config) error {
	if cfg.Database.Driver != "sqlite" {
		return nil
	}
	dir := filepath.Dir(sqliteFilePath(cfg.Database.DSN))
	if dir == "" || dir == "." {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("config: 创建数据目录 %s 失败: %w", dir, err)
	}
	return nil
}

// generateSessionSecret 使用 crypto/rand 生成 base64 编码的强随机密钥。
func generateSessionSecret() (string, error) {
	buf := make([]byte, sessionSecretBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("config: 生成 session_secret 失败: %w", err)
	}
	return base64.StdEncoding.EncodeToString(buf), nil
}

// sqliteFilePath 从 sqlite DSN 中提取实际文件路径，
// 忽略可能存在的 "file:" 前缀与 "?" 之后的查询参数。
func sqliteFilePath(dsn string) string {
	s := strings.TrimPrefix(strings.TrimSpace(dsn), "file:")
	if i := strings.IndexByte(s, '?'); i >= 0 {
		s = s[:i]
	}
	return s
}

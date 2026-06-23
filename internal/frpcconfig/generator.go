// Package frpcconfig 根据 tenant、proxy、resource、grant 数据生成 frpc.toml 配置。
//
// 安全原则(见 docs/SECURITY.md):
//   - tenant token 明文只在创建 tenant 或重置 token 时出现一次。
//   - 数据库只保存 token_hash,不能反推明文。
//   - 平时查看 frpc 配置只能生成 token 占位符模板。
//   - 创建/重置时可生成含真实 token 的完整配置(因 plain_token 仍在内存中)。
//   - 绝不将 plain_token 写入数据库、审计日志、localStorage。
//
// 生成的 TOML 面向 frp 新版配置风格(serverAddr/serverPort/user/metadatas.token/[[proxies]])。
package frpcconfig

import (
	"fmt"
	"strconv"
	"strings"
)

// Config 是 frpc 配置的结构化表示。
type Config struct {
	ServerAddr string
	ServerPort int
	User       string
	Token      string // 明文 token 或占位符
	Proxies    []ProxyEntry
}

// ProxyEntry 是一个 proxy 条目。
type ProxyEntry struct {
	Name       string
	Type       string // http / https / tcp / udp
	LocalIP    string
	LocalPort  int
	Subdomain  string // http/https 使用
	RemotePort int    // tcp/udp 使用
}

// Generate 根据 Config 生成 frpc.toml 字符串。
//
// 安全:token 由调用方传入,本函数不访问数据库。
// 如果 proxies 为空,生成基础配置并附中文注释提示。
func Generate(cfg Config) (string, error) {
	var b strings.Builder

	// 头部注释。
	b.WriteString("# frpc 客户端配置 —— 由 frp-warden 自动生成\n")
	b.WriteString("# 文档:https://github.com/fatedier/frp\n\n")

	// 基础配置。
	b.WriteString(fmt.Sprintf("serverAddr = %s\n", tomlString(cfg.ServerAddr)))
	b.WriteString(fmt.Sprintf("serverPort = %d\n\n", cfg.ServerPort))
	b.WriteString(fmt.Sprintf("user = %s\n\n", tomlString(cfg.User)))
	b.WriteString(fmt.Sprintf("metadatas.token = %s\n", tomlString(cfg.Token)))

	if len(cfg.Proxies) == 0 {
		b.WriteString("\n# 尚未配置映射(proxy)。请先在管理后台创建映射后再生成配置。\n")
		return b.String(), nil
	}

	// 生成每个 proxy。
	for _, p := range cfg.Proxies {
		b.WriteString("\n[[proxies]]\n")
		b.WriteString(fmt.Sprintf("name = %s\n", tomlString(p.Name)))
		b.WriteString(fmt.Sprintf("type = %s\n", tomlString(p.Type)))
		b.WriteString(fmt.Sprintf("localIP = %s\n", tomlString(p.LocalIP)))
		b.WriteString(fmt.Sprintf("localPort = %d\n", p.LocalPort))

		switch p.Type {
		case "http", "https":
			if p.Subdomain == "" {
				return "", fmt.Errorf("frpcconfig: http/https proxy %q 缺少 subdomain", p.Name)
			}
			b.WriteString(fmt.Sprintf("subdomain = %s\n", tomlString(p.Subdomain)))
		case "tcp", "udp":
			if p.RemotePort <= 0 || p.RemotePort > 65535 {
				return "", fmt.Errorf("frpcconfig: %s proxy %q 的 remotePort 非法(%d)", p.Type, p.Name, p.RemotePort)
			}
			b.WriteString(fmt.Sprintf("remotePort = %d\n", p.RemotePort))
		default:
			return "", fmt.Errorf("frpcconfig: 不支持的 proxy 类型 %q", p.Type)
		}
	}

	return b.String(), nil
}

// tomlString 将字符串转为 TOML 双引号字符串,正确转义特殊字符。
func tomlString(s string) string {
	// TOML 双引号字符串需要转义:反斜杠、双引号、换行、回车、制表等。
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return `"` + s + `"`
}

// ValidatePort 校验端口是否在合法范围 1-65535。
func ValidatePort(port int) bool {
	return port >= 1 && port <= 65535
}

// ParsePort 将字符串解析为端口号,校验范围。
func ParsePort(s string) (int, error) {
	port, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0, fmt.Errorf("frpcconfig: 端口 %q 不是合法数字", s)
	}
	if !ValidatePort(port) {
		return 0, fmt.Errorf("frpcconfig: 端口 %d 超出范围(1-65535)", port)
	}
	return port, nil
}

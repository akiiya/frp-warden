package frpcconfig

import (
	"strings"
	"testing"
)

func TestGenerateBasic(t *testing.T) {
	cfg := Config{
		ServerAddr: "frp.example.com",
		ServerPort: 7000,
		User:       "ufi001",
		Token:      "test-token-123",
		Proxies: []ProxyEntry{
			{Name: "web", Type: "http", LocalIP: "127.0.0.1", LocalPort: 8080, Subdomain: "ufi001"},
		},
	}
	out, err := Generate(cfg)
	if err != nil {
		t.Fatalf("生成失败: %v", err)
	}
	if !strings.Contains(out, `serverAddr = "frp.example.com"`) {
		t.Error("缺少 serverAddr")
	}
	if !strings.Contains(out, `serverPort = 7000`) {
		t.Error("缺少 serverPort")
	}
	if !strings.Contains(out, `user = "ufi001"`) {
		t.Error("缺少 user")
	}
	if !strings.Contains(out, `metadatas.token = "test-token-123"`) {
		t.Error("缺少 metadatas.token")
	}
}

func TestGenerateHTTPProxy(t *testing.T) {
	cfg := Config{
		ServerAddr: "frp.example.com", ServerPort: 7000, User: "u", Token: "t",
		Proxies: []ProxyEntry{
			{Name: "web", Type: "http", LocalIP: "127.0.0.1", LocalPort: 8080, Subdomain: "ufi001"},
			{Name: "web2", Type: "https", LocalIP: "127.0.0.1", LocalPort: 8443, Subdomain: "ufi001"},
		},
	}
	out, err := Generate(cfg)
	if err != nil {
		t.Fatalf("生成失败: %v", err)
	}
	if !strings.Contains(out, `subdomain = "ufi001"`) {
		t.Error("http proxy 应包含 subdomain")
	}
	if strings.Contains(out, "remotePort") {
		t.Error("http proxy 不应包含 remotePort")
	}
}

func TestGenerateTCPUDPProxy(t *testing.T) {
	cfg := Config{
		ServerAddr: "frp.example.com", ServerPort: 7000, User: "u", Token: "t",
		Proxies: []ProxyEntry{
			{Name: "ssh", Type: "tcp", LocalIP: "127.0.0.1", LocalPort: 22, RemotePort: 61001},
			{Name: "dns", Type: "udp", LocalIP: "127.0.0.1", LocalPort: 53, RemotePort: 62001},
		},
	}
	out, err := Generate(cfg)
	if err != nil {
		t.Fatalf("生成失败: %v", err)
	}
	if !strings.Contains(out, `remotePort = 61001`) {
		t.Error("tcp proxy 应包含 remotePort 61001")
	}
	if !strings.Contains(out, `remotePort = 62001`) {
		t.Error("udp proxy 应包含 remotePort 62001")
	}
	if strings.Contains(out, "subdomain") {
		t.Error("tcp/udp proxy 不应包含 subdomain")
	}
}

func TestGenerateEmptyProxies(t *testing.T) {
	cfg := Config{
		ServerAddr: "frp.example.com", ServerPort: 7000, User: "u", Token: "t",
		Proxies: nil,
	}
	out, err := Generate(cfg)
	if err != nil {
		t.Fatalf("生成失败: %v", err)
	}
	if !strings.Contains(out, "尚未配置映射") {
		t.Error("无 proxy 时应包含提示注释")
	}
}

func TestGenerateInvalidRemotePort(t *testing.T) {
	cfg := Config{
		ServerAddr: "frp.example.com", ServerPort: 7000, User: "u", Token: "t",
		Proxies: []ProxyEntry{
			{Name: "ssh", Type: "tcp", LocalIP: "127.0.0.1", LocalPort: 22, RemotePort: 0},
		},
	}
	if _, err := Generate(cfg); err == nil {
		t.Error("非法 remotePort 应返回错误")
	}
}

func TestGenerateHTTPEmptySubdomain(t *testing.T) {
	cfg := Config{
		ServerAddr: "frp.example.com", ServerPort: 7000, User: "u", Token: "t",
		Proxies: []ProxyEntry{
			{Name: "web", Type: "http", LocalIP: "127.0.0.1", LocalPort: 8080, Subdomain: ""},
		},
	}
	if _, err := Generate(cfg); err == nil {
		t.Error("http proxy 空 subdomain 应返回错误")
	}
}

func TestTOMLStringEscape(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`hello`, `"hello"`},
		{`he"llo`, `"he\"llo"`},
		{`he\llo`, `"he\\llo"`},
		{"he\nllo", `"he\nllo"`},
	}
	for _, tt := range tests {
		got := tomlString(tt.input)
		if got != tt.want {
			t.Errorf("tomlString(%q) = %q,期望 %q", tt.input, got, tt.want)
		}
	}
}

func TestValidatePort(t *testing.T) {
	if !ValidatePort(1) || !ValidatePort(65535) || !ValidatePort(8080) {
		t.Error("合法端口应返回 true")
	}
	if ValidatePort(0) || ValidatePort(65536) || ValidatePort(-1) {
		t.Error("非法端口应返回 false")
	}
}

func TestParsePort(t *testing.T) {
	if _, err := ParsePort("abc"); err == nil {
		t.Error("非数字应返回错误")
	}
	if _, err := ParsePort("0"); err == nil {
		t.Error("端口 0 应返回错误")
	}
	if _, err := ParsePort("70000"); err == nil {
		t.Error("端口 70000 应返回错误")
	}
	port, err := ParsePort("8080")
	if err != nil || port != 8080 {
		t.Errorf("ParsePort(8080) = %d,%v", port, err)
	}
}

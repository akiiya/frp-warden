# frp-warden

> Multi-tenant authorization control plane for frp.

frp-warden 是 [frp](https://github.com/fatedier/frp) 的多租户授权控制面板。
它决定**哪个设备可以使用哪个子域名或公网端口**,并通过 frps server plugin 机制强制执行。

**状态:v0.1.0-rc 准备中。** 已完成 Phase 0~10,首个候选发布版本准备就绪。

## 核心能力

- **多租户**:每台设备对应一个独立租户(tenant)。
- **独立 token**:每个租户拥有独立的 frp token,数据库只存 hash。
- **资源授权**:子域名/TCP 端口/UDP 端口按租户授权,资源唯一归属。
- **服务端强制校验**:frps plugin Login/NewProxy/Ping 全部由 frp-warden 授权判断。
- **单二进制 + 内嵌 WebUI**:下载一个文件即可运行,管理页面内嵌其中。
- **frpc 配置生成**:创建租户/重置 token 时一次性返回完整 frpc.toml。
- **零配置启动**:首次运行自动生成配置文件、数据库和管理员账号。

## 架构图

```
公网服务器 / VPS
┌──────────────────────────────────────────────┐
│  frps(:7000)         frp-warden(:8080+:9000) │
│    │                    │                     │
│    │ HTTP plugin hooks  │ 管理后台 WebUI      │
│    └────────────────────┘                     │
│         127.0.0.1:9000                        │
└──────────────────────────────────────────────┘
         ▲
         │ frpc(携带 tenant token)
         │
┌────────┴─────────────┐
│ 随身 WiFi / Debian   │
│ 只运行 frpc          │
└──────────────────────┘
```

## 部署模型

- **frps + frp-warden** 部署在同一台公网服务器/VPS 上。
- **随身 WiFi / Debian 设备只运行 frpc**,不运行 frps、不运行 frp-warden。
- 运维在 WebUI 创建租户、授权资源、生成 frpc 配置,复制到设备启动。

## 快速开始

### 1. 下载

从 [GitHub Releases](https://github.com/fengheasia/frp-warden/releases) 下载对应平台的二进制。

### 2. 首次启动

```sh
# Linux
chmod +x frp-warden
./frp-warden

# Windows
frp-warden.exe
```

首次启动会自动:

- 生成 `config.yaml`(含自动生成的 session_secret)。
- 创建 SQLite 数据库 `data/frp-warden.db`。
- 创建默认管理员,**密码随机生成并一次性显示在控制台**。

### 3. 访问 WebUI

浏览器访问 `http://服务器IP:8080`,用初始管理员账号登录。

### 4. 配置 frps

在 frps.toml 中添加:

```toml
bindPort = 7000
vhostHTTPPort = 80
subdomainHost = "frp.example.com"

[[httpPlugins]]
name = "frp-warden"
addr = "127.0.0.1:9000"
path = "/plugin/frp"
ops = ["Login", "NewProxy", "Ping"]
```

### 5. 创建租户和生成 frpc 配置

1. 在 WebUI 创建域名区域(如 `*.frp.example.com`)。
2. 创建租户(会生成 token 并一次性显示)。
3. 创建 subdomain 资源。
4. 授权资源给租户。
5. 创建 http proxy 映射。
6. 复制 frpc.toml 到设备运行。

## 安全注意事项

- 初始管理员密码只显示一次,请立即记录。
- tenant token 只在创建/重置时显示一次,数据库只存 hash。
- **frps plugin 端口默认 `127.0.0.1:9000`,切勿暴露公网。**
- 管理后台建议用 Nginx/Caddy 反代并启用 HTTPS。

## 文档

- [安装部署](docs/INSTALL.md)
- [frps 配置](docs/FRPS_SETUP.md)
- [systemd 部署](docs/SYSTEMD.md)
- [反向代理](docs/REVERSE_PROXY.md)
- [端到端冒烟测试](docs/SMOKE_TEST.md)
- [常见问题](docs/TROUBLESHOOTING.md)
- [配置说明](docs/CONFIGURATION.md)
- [安全模型](docs/SECURITY.md)
- [GitHub 与发布](docs/GITHUB.md)
- [架构](docs/ARCHITECTURE.md)
- [API](docs/API.md)
- [数据模型](docs/DATA_MODEL.md)
- [路线图](docs/ROADMAP.md)
- [决策记录](docs/DECISIONS/)

## 开发

```sh
# 后端
go build -o frp-warden ./cmd/frp-warden

# 前端
cd web && npm install && npm run dev

# 构建单二进制(含内嵌 WebUI)
cd web && npm install && npm run build && cd ..
go run ./tools/sync-web-dist
go build -o frp-warden ./cmd/frp-warden
```

详见 [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md)。

## License

[MIT](LICENSE)

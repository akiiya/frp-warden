# 安装部署指南

本文档说明如何下载、安装和首次运行 frp-warden。

## 下载

从 [GitHub Releases](https://github.com/fengheasia/frp-warden/releases) 下载对应平台的压缩包。

| 平台 | 文件名 |
|---|---|
| Windows 64位 | `frp-warden_*_windows_amd64.zip` |
| Windows 32位 | `frp-warden_*_windows_386.zip` |
| Linux 64位 | `frp-warden_*_linux_amd64.tar.gz` |
| Linux 32位 | `frp-warden_*_linux_386.tar.gz` |
| Linux ARM64 | `frp-warden_*_linux_arm64.tar.gz` |
| Linux ARMv7 | `frp-warden_*_linux_armv7.tar.gz` |

下载后校验 SHA-256:

```sh
# Linux/macOS
sha256sum -c checksums.txt

# Windows PowerShell
Get-FileHash frp-warden_*.zip -Algorithm SHA256
```

## 安装

### Linux

```sh
# 解压
tar xzf frp-warden_*_linux_amd64.tar.gz

# 移动到系统目录(可选)
sudo mv frp-warden /usr/local/bin/

# 或直接在当前目录运行
chmod +x frp-warden
```

### Windows

解压 zip 文件,得到 `frp-warden.exe`。

## 首次启动

```sh
# Linux
./frp-warden

# Windows
frp-warden.exe
```

首次启动时,frp-warden 会自动:

1. 生成默认配置文件 `config.yaml`(当前目录)。
2. 创建 SQLite 数据库目录 `data/` 和数据库文件 `data/frp-warden.db`。
3. 执行数据库迁移。
4. 创建默认管理员,密码**随机生成并一次性显示在控制台**。

```
============================================================
frp-warden 已创建默认管理员账号

用户名: admin
密码: xxxxxxxxxxxxxxxxxx

请立即登录管理后台修改默认密码。
该密码只会显示一次，请妥善保存。
============================================================
```

**请立即记录密码**,关闭窗口后无法再次查看。

## 配置文件

默认配置文件路径:当前目录下的 `config.yaml`。

可通过 `-c` 参数指定:

```sh
./frp-warden -c /etc/frp-warden/config.yaml
```

配置示例见 [config.example.yaml](../config.example.yaml),完整说明见 [CONFIGURATION.md](CONFIGURATION.md)。

### 关键配置项

| 配置项 | 默认值 | 说明 |
|---|---|---|
| `server.admin_addr` | `0.0.0.0:8080` | 管理后台监听地址 |
| `server.plugin_addr` | `127.0.0.1:9000` | frps plugin 接口(仅回环) |
| `database.driver` | `sqlite` | 数据库驱动 |
| `database.dsn` | `./data/frp-warden.db` | SQLite 文件路径 |
| `frp.server_addr` | `127.0.0.1` | frps 地址(客户端连接用) |
| `frp.server_port` | `7000` | frps 控制端口 |
| `frp.subdomain_host` | `""` | 泛域名根(需与 frps 一致) |

### 修改监听地址

编辑 `config.yaml`:

```yaml
server:
  admin_addr: "0.0.0.0:8080"    # 管理后台
  plugin_addr: "127.0.0.1:9000" # frps plugin(不要改)
```

### 设置 frps 连接信息

```yaml
frp:
  server_addr: "frp.example.com"  # 你的公网服务器地址
  server_port: 7000
  subdomain_host: "frp.example.com"  # 与 frps.toml 的 subdomainHost 一致
```

## 访问 WebUI

启动后,在浏览器访问:

```
http://服务器IP:8080
```

用初始管理员账号登录。建议首次登录后立即修改密码。

## 备份

需要备份的文件:

- `config.yaml` — 配置文件(含 session_secret)
- `data/frp-warden.db` — SQLite 数据库(含所有租户、资源、授权数据)

```sh
# 备份示例
cp config.yaml config.yaml.bak
cp data/frp-warden.db data/frp-warden.db.bak
```

## 命令行参数

| 参数 | 说明 |
|---|---|
| `-version` | 打印版本信息后退出 |
| `-c <path>` | 指定配置文件路径 |
| `-config <path>` | 同 `-c` |

## 下一步

- 配置 frps: [FRPS_SETUP.md](FRPS_SETUP.md)
- systemd 部署: [SYSTEMD.md](SYSTEMD.md)
- 反向代理: [REVERSE_PROXY.md](REVERSE_PROXY.md)
- 端到端测试: [SMOKE_TEST.md](SMOKE_TEST.md)

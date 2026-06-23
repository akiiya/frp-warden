# 常见问题

## 忘记默认管理员密码怎么办

初始管理员密码只在首次启动时显示一次。如果忘记,需要重置数据库:

1. 停止 frp-warden。
2. 删除 `data/frp-warden.db`(会丢失所有数据)。
3. 重新启动 frp-warden,会重新生成管理员并显示新密码。

如果不想丢失数据,可以手动操作 SQLite 数据库:

```sh
sqlite3 data/frp-warden.db "DELETE FROM admins WHERE username='admin';"
```

重启后会自动创建新的管理员。

## 忘记 tenant token 怎么办

tenant token 明文只在创建/重置时显示一次,数据库只保存 hash,无法恢复。

解决:在 WebUI 中"重置 Token",会生成新的 token 并一次性显示。

## 为什么查看 frpc 配置时 token 是占位符

数据库只保存 token hash,无法反推明文。因此平时查看 frpc 配置时只能显示占位符模板。

获取含真实 token 的完整配置的唯一方式:创建租户或重置 token 时的一次性弹窗。

## 为什么 NewProxy 被拒绝

常见原因:

- 资源未授权给该 tenant。
- proxy 类型与资源类型不匹配(http 需要 subdomain 资源,tcp 需要 tcp_port 资源)。
- 请求的 subdomain 或 remotePort 与授权资源不一致。
- 资源或授权被禁用。
- 使用了 custom_domains(当前不允许)。

在 WebUI 中检查:租户状态、资源状态、授权状态、proxy 映射配置。

## 为什么子域名访问不到

检查以下几点:

1. DNS 泛域名解析是否配置(`*.frp.example.com → 服务器IP`)。
2. frps.toml 的 `subdomainHost` 是否配置正确。
3. frp-warden 配置的 `frp.subdomain_host` 是否与 frps 一致。
4. 资源是否已授权给 tenant。
5. proxy 映射是否创建。
6. frpc 是否正常运行(proxy 状态为 `start proxy success`)。
7. 本地服务是否在运行。

## 为什么 WebUI 能打开但 frps plugin 不工作

- 检查 frp-warden 是否启动了 plugin 接口(默认 `127.0.0.1:9000`)。
- 检查 frps.toml 的 `httpPlugins.addr` 是否与 frp-warden 的 `plugin_addr` 一致。
- 检查 frp-warden 和 frps 是否在同一台机器上(或网络可达)。

## 为什么 plugin 端口不能暴露公网

plugin 端口参与授权决策。如果暴露到公网,任何人都可以发送伪造的 Login/NewProxy 请求,
绕过授权控制。默认 `127.0.0.1:9000` 只允许本机 frps 调用。

## SQLite 数据库在哪里

默认路径:`./data/frp-warden.db`。

可在 `config.yaml` 的 `database.dsn` 中修改。

## 如何备份和迁移

备份两个文件:

- `config.yaml` — 配置(含 session_secret)
- `data/frp-warden.db` — 数据库

迁移:将这两个文件复制到新服务器,运行 frp-warden 即可。

## MySQL/PostgreSQL 现在是否完整支持

当前仅 SQLite 完整可用(迁移已实现)。MySQL/MariaDB/PostgreSQL 仅连接骨架,
迁移尚未实现,使用时会在迁移阶段 fail fast 报错。

## GitHub Release 产物怎么选

| 系统 | 架构 | 选择 |
|---|---|---|
| Windows 64位 | x86_64 | `windows_amd64.zip` |
| Windows 32位 | x86 | `windows_386.zip` |
| Linux 64位 | x86_64 | `linux_amd64.tar.gz` |
| Linux 32位 | x86 | `linux_386.tar.gz` |
| Linux ARM64 | aarch64 | `linux_arm64.tar.gz` |
| Linux ARMv7 | armv7l | `linux_armv7.tar.gz` |

不确定架构时,Linux 可运行 `uname -m` 查看。

## Windows/Linux 如何启动

```sh
# Linux
chmod +x frp-warden
./frp-warden

# Windows CMD
frp-warden.exe

# Windows PowerShell
.\frp-warden.exe
```

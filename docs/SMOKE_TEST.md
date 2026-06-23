# 端到端冒烟测试

本文档提供一个完整的端到端测试流程,验证 frp-warden + frps + frpc 的完整链路。

## 前提条件

- 一台公网服务器/VPS,已安装 frps 和 frp-warden。
- 一台客户端设备(随身 WiFi / Debian / 任意 Linux),已安装 frpc。
- 域名已配置泛域名解析(`*.frp.example.com → 服务器IP`)。

## 测试流程

### 1. 启动 frp-warden

```sh
./frp-warden
```

记录初始管理员密码(仅显示一次)。

### 2. 启动 frps

```sh
frps -c frps.toml
```

确认 frps 日志中出现:

```
[plugin.go:xxx] plugin [frp-warden] address: 127.0.0.1:9000
```

### 3. 登录 WebUI

浏览器访问 `http://服务器IP:8080`(或反代后的域名),用初始管理员账号登录。

首次登录后建议修改密码。

### 4. 创建域名区域

在"域名区域"页面创建:

- 名称:默认区域
- Zone: `*.frp.example.com`

### 5. 创建租户

在"租户管理"页面创建:

- Code: `test001`
- 名称:测试设备

创建成功后,**立即复制保存 token 和 frpc.toml**。

### 6. 创建资源

在"资源池"页面创建 subdomain 资源:

- 类型: subdomain
- 域名区域: 选择刚才创建的区域
- Value: `test001`

### 7. 授权资源

在"授权管理"页面将资源授权给租户 `test001`。

### 8. 创建映射

在"映射管理"页面创建:

- 租户: test001
- 资源: 选择刚才创建的 subdomain 资源
- 名称: web
- 类型: http
- 本地 IP: 127.0.0.1
- 本地端口: 8080(或任意本地服务端口)

### 9. 在客户端运行 frpc

将步骤 5 中获取的 frpc.toml 复制到客户端设备,运行:

```sh
frpc -c frpc.toml
```

确认 frpc 日志中出现:

```
[xxx.go:xxx] [web] start proxy success
```

### 10. 访问子域名

在浏览器访问:

```
http://test001.frp.example.com
```

如果本地 8080 端口有服务,应该能看到响应。

### 11. 验证未授权资源被拒绝

尝试修改 frpc.toml,将 subdomain 改为未授权的值:

```toml
subdomain = "unauthorized001"
```

重启 frpc,确认日志中出现:

```
[xxx.go:xxx] [web] start proxy failed: [NewProxy] subdomain unauthorized001 not allowed
```

恢复正确的 subdomain。

### 12. 禁用租户

在 WebUI 中禁用租户 `test001`。

等待 frpc 下次 Ping(约 30 秒),确认 frpc 日志中出现连接被拒绝。

### 13. 查看审计日志

在 WebUI 的"审计日志"页面查看操作记录,确认:

- 租户创建、资源创建、授权创建等操作有记录。
- 日志中不包含 token 明文或密码。

### 14. 恢复

重新启用租户 `test001`,确认 frpc 可以重新连接。

## 预期结果

| 步骤 | 预期 |
|---|---|
| 启动 frp-warden | 生成配置、数据库、管理员 |
| 启动 frps | plugin 连接成功 |
| 登录 WebUI | 成功 |
| 创建域名区域 | 成功 |
| 创建租户 | 成功,显示 token |
| 创建资源 | 成功 |
| 授权资源 | 成功 |
| 创建映射 | 成功 |
| 运行 frpc | proxy 启动成功 |
| 访问子域名 | 看到本地服务 |
| 未授权子域名 | 被拒绝 |
| 禁用租户 | frpc 连接被拒绝 |
| 审计日志 | 有记录,无敏感信息 |

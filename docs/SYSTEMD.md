# systemd 部署指南

本文档说明如何使用 systemd 管理 frp-warden 服务。

## 目录结构建议

```
/opt/frp-warden/                  # 程序目录
/opt/frp-warden/frp-warden        # 二进制
/etc/frp-warden/config.yaml       # 配置文件
/var/lib/frp-warden/              # 数据目录(data/frp-warden.db)
```

## 创建用户

```sh
sudo useradd -r -s /usr/sbin/nologin frpwarden
```

## 安装

```sh
# 创建目录
sudo mkdir -p /opt/frp-warden /etc/frp-warden /var/lib/frp-warden

# 复制二进制
sudo cp frp-warden /opt/frp-warden/
sudo chmod +x /opt/frp-warden/frp-warden

# 复制配置(首次运行后会自动生成,或手动复制示例)
sudo cp config.example.yaml /etc/frp-warden/config.yaml

# 设置权限
sudo chown -R frpwarden:frpwarden /opt/frp-warden /etc/frp-warden /var/lib/frp-warden
```

修改配置文件中的数据库路径:

```yaml
database:
  driver: "sqlite"
  dsn: "/var/lib/frp-warden/frp-warden.db"
```

## systemd unit 文件

创建 `/etc/systemd/system/frp-warden.service`:

```ini
[Unit]
Description=frp-warden - frp 多租户授权控制面板
Documentation=https://github.com/fengheasia/frp-warden
After=network.target
# 如果 frps 也运行在同一台机器上,可以添加:
# Wants=frps.service
# After=network.target frps.service

[Service]
Type=simple
User=frpwarden
Group=frpwarden
WorkingDirectory=/opt/frp-warden
ExecStart=/opt/frp-warden/frp-warden -c /etc/frp-warden/config.yaml
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal

# 安全加固
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/frp-warden /etc/frp-warden
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

## 管理命令

```sh
# 重新加载 systemd 配置
sudo systemctl daemon-reload

# 启动
sudo systemctl start frp-warden

# 停止
sudo systemctl stop frp-warden

# 重启
sudo systemctl restart frp-warden

# 开机自启
sudo systemctl enable frp-warden

# 查看状态
sudo systemctl status frp-warden

# 查看日志
sudo journalctl -u frp-warden -f
```

## 查看初始管理员密码

首次启动后,初始管理员密码会打印到日志:

```sh
sudo journalctl -u frp-warden | grep -A 5 "已创建默认管理员"
```

## frps 与 frp-warden 同机部署

如果 frps 也运行在同一台机器上,建议:

1. frp-warden 先启动(等待数据库初始化完成)。
2. frps 后启动(依赖 frp-warden 的 plugin 接口)。

在 frps 的 systemd unit 中添加:

```ini
[Unit]
After=network.target frp-warden.service
Wants=frp-warden.service
```

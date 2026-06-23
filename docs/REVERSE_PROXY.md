# 反向代理建议

本文档说明如何用 Nginx 或 Caddy 为 frp-warden 管理后台配置反向代理和 HTTPS。

## 为什么要反代

- 管理后台(`admin_addr`)直接暴露到公网不安全(HTTP 明文)。
- 反代可以启用 HTTPS,保护登录凭据和 session cookie。
- 反代可以绑定域名,方便访问。

## 重要提醒

- **plugin 端口(`127.0.0.1:9000`)不要反代到公网。**该端口只允许本机 frps 调用。
- **WebUI 反代到 `admin_addr`(默认 `127.0.0.1:8080`)。**
- **frps 的 `vhostHTTPPort`/`vhostHTTPSPort` 与 WebUI 反代端口不要混淆。**
  frps 的 vhost 端口用于转发 frpc 客户端的 HTTP/HTTPS 流量,WebUI 反代用于管理页面。

## Caddy 示例

Caddy 自动申请 HTTPS 证书,配置最简单。

```Caddyfile
# frp-warden 管理后台
frp-admin.example.com {
    reverse_proxy 127.0.0.1:8080
}
```

如果 frps 的 vhostHTTPPort 也需要反代(如 80 端口被 Caddy 占用):

```Caddyfile
# frps HTTP 虚拟主机(vhostHTTPPort 假设为 8080)
*.frp.example.com {
    reverse_proxy 127.0.0.1:8080
}

# frp-warden 管理后台(用独立子域名)
frp-admin.example.com {
    reverse_proxy 127.0.0.1:8080
}
```

注意:如果 Caddy 和 frps 都监听 80 端口会冲突。建议 frps 的 `vhostHTTPPort` 改为
非标准端口(如 8081),由 Caddy 反代到该端口。

## Nginx 示例

```nginx
# frp-warden 管理后台
server {
    listen 80;
    server_name frp-admin.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name frp-admin.example.com;

    ssl_certificate /etc/letsencrypt/live/frp-admin.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/frp-admin.example.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

如果需要反代 frps HTTP 虚拟主机:

```nginx
server {
    listen 80;
    server_name *.frp.example.com;

    location / {
        proxy_pass http://127.0.0.1:8081;  # frps vhostHTTPPort
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## 注意事项

- frp-warden 管理后台和 frps vhost 可以共用同一台服务器的 80/443 端口,但需要用
  不同的域名(或子域名)区分。
- 如果 Nginx/Caddy 占用了 80 端口,frps 的 `vhostHTTPPort` 需要改为其他端口(如 8081)。
- plugin 端口(`127.0.0.1:9000`)始终不要暴露。

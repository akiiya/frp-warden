# 安全政策

## 报告安全问题

如果发现安全漏洞,请**不要**在公开 Issue 中提交。请通过私信或邮件联系维护者。

## 安全模型简述

frp-warden 是 frp 的多租户授权控制面。核心安全原则:

- **所有授权在服务端强制执行**,客户端 frpc 配置不可信。
- **密码和 token 仅以 hash 存储**,绝不保存明文。
- **frps plugin 端口默认仅监听 `127.0.0.1`**,不得暴露公网。
- 管理员初始密码随机生成,仅显示一次。
- tenant token 明文仅在创建/重置时返回一次。

详细安全规则见 [docs/SECURITY.md](docs/SECURITY.md)。

## 注意事项

- 不要在公开场合分享 config.yaml(含 session_secret)。
- 不要在公开场合分享 tenant token 明文。
- 不要把 frps plugin 端口暴露到公网。
- 定期备份 `config.yaml` 和 `data/frp-warden.db`。

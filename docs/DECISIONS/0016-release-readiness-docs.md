# 0016 — 发布前文档与准备工作

- 状态:已接受
- 日期:2026-06-23

## 背景

Phase 10 是首个候选发布版本(v0.1.0-rc1)前的最后准备阶段。需要补齐安装/部署/测试文档,
让新用户能够顺利上手,让运维能够安全部署。

## 决策与理由

### 为什么发布前必须补齐安装/部署/冒烟测试文档

没有文档的软件无法被正确使用。安装文档让新用户知道如何下载和运行;部署文档让运维知道
如何配置 frps、systemd、反向代理;冒烟测试文档让开发者和运维验证完整链路。这些是
软件成熟度的基本要求。

### 为什么需要 config.example.yaml

config.example.yaml 提供一个带中文注释的配置参考,让新用户快速理解每个配置项的作用。
它不包含真实 secret(session_secret 留空),可以安全地提交到仓库。

### 为什么需要 frps/frpc 示例

frps 和 frpc 的配置格式对新用户来说不够直观。示例文件提供一个可以直接参考的起点,
减少配置错误。frpc 示例使用占位符 token,明确说明真实配置应从 WebUI 生成。

### 为什么 WebUI 建议反代 HTTPS

管理后台处理登录凭据和 session cookie。如果直接通过 HTTP 访问,这些敏感信息会以明文
传输。反向代理(Nginx/Caddy)可以启用 HTTPS,保护传输安全。

### 为什么 plugin 端口不能暴露公网

plugin 端口参与授权决策(Login/NewProxy/Ping)。如果暴露到公网,攻击者可以发送伪造的
请求绕过授权。默认 `127.0.0.1:9000` 只允许本机 frps 调用。

### 为什么默认使用 MIT License

MIT 是最宽松、最广泛使用的开源许可证之一。它允许自由使用、修改和分发,对商业友好。
对于一个工具型项目,MIT 是合适的选择。

### 为什么本轮不做 Docker/SBOM/签名

Docker 化、SBOM 和签名都是有价值的,但不是首个候选版本的必要条件。当前单二进制已经
足够简单,Docker 可以后续按需添加。SBOM 和签名在供应链安全成熟后再引入。

### 为什么 README 需要面向新用户重写

之前的 README 更多是开发者的内部文档。面向 GitHub 用户的 README 应该让新用户在 5 分钟内
理解项目是什么、怎么跑起来。快速开始、架构图、部署模型、安全提醒是必要的。

## 影响

- 新增 13 个文件:LICENSE、SECURITY.md、config.example.yaml、examples/frps.toml、
  examples/frpc.example.toml、docs/INSTALL.md、docs/FRPS_SETUP.md、docs/SYSTEMD.md、
  docs/REVERSE_PROXY.md、docs/SMOKE_TEST.md、docs/TROUBLESHOOTING.md、docs/GITHUB.md、
  docs/DECISIONS/0016-release-readiness-docs.md。
- README.md 面向新用户重写。
- CHANGELOG.md 整理为 v0.1.0-rc1 格式。
- CLAUDE.md、AGENTS.md、ROADMAP.md 更新到 Phase 10 完成状态。

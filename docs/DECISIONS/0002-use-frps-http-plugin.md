# 0002 — 通过 frps HTTP server 插件机制集成

- 状态:已接受
- 日期:2026-06-12

## 背景

frp-warden 需要在 frps 做出信任决策的各个环节(客户端登录、proxy 注册、
保活)实施按 tenant 的授权。我们必须在不 fork frp、也不将 frp-warden 的
发布周期与 frp 内部实现耦合的前提下做到这一点。

## 决策

使用 **frps 的 server-plugin HTTP 机制**。frps 通过一个
`[[httpPlugins]]` 条目配置,指向 frp-warden 的插件端点
(`/plugin/frp`),并转发 `Login`、`NewProxy` 和 `Ping` 操作。
frp-warden 通过 HTTP 返回允许/拒绝(并可进行改写)。frps 与 frp-warden
保持为**独立的进程**;frps 不会以库的形式加载 frp-warden。

## 影响

- frp-warden 可与原版 frps 配合工作 —— 无需 fork,无需重新编译 frps。
- 插件端点对安全至关重要,默认必须绑定到 loopback
  (参见 [../SECURITY.md](../SECURITY.md) 与 [../FRP_PLUGIN.md](../FRP_PLUGIN.md))。
- `NewProxy` 成为集中的授权实施点。
- 集成依赖于 frp 的插件协议;必须持续跟踪 frp 中
  协议/版本的变更。
- 通信基于 HTTP,因此在需要时,frps 与 frp-warden 可以通过受信任的
  私有链路运行在不同的主机上。

# 0001 — 使用 Go 并交付单一二进制

- 状态:已接受
- 日期:2026-06-12

## 背景

frp-warden 必须能够在各种主机上(Debian 系与 CentOS 系 Linux、Windows、
多种 CPU 架构)方便地与 frps 部署在一起。运维人员不应被迫安装
运行时、维护单独的 Web 服务器或处理原生依赖。

## 决策

用 **Go** 实现后端,并将 frp-warden 以**单一、自包含的二进制**形式分发。
避免使用 CGO,使交叉编译仅靠 `GOOS`/`GOARCH` 即可完成,无需 C 工具链,
产出静态二进制。通过 Go `embed` 将构建好的
Web 前端嵌入到二进制中(参见
[0003](0003-use-vue3-vite-frontend.md) 与 [../ROADMAP.md](../ROADMAP.md) 中的第 7 阶段)。

## 影响

- 每个平台只需交付一个产物;没有外部资源或运行时。
- 易于为 windows/amd64、windows/386、linux/amd64、
  linux/386、linux/arm64、linux/arm/v7 进行交叉编译。
- SQLite 驱动必须是无 CGO 的(参见 [0005](0005-use-sqlite-by-default.md))。
- Linux 构建保持广泛兼容(不会因 CGO 而与 glibc 版本耦合),
  有助于 Debian/CentOS 的可移植性。
- 发布时,前端构建是后端构建的前置条件。

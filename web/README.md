# frp-warden web（管理后台）

基于 Vue 3 + Vite + TypeScript + [Naive UI](https://www.naiveui.com/) 的管理后台前端。
构建输出到 `web/dist`，后续 Phase 7 通过 Go `embed` 内嵌到二进制。

## 状态

Phase 6 已完成：登录页、管理布局（左侧导航 + 顶部栏）、仪表盘、租户管理（含 frpc 配置查看/复制）、
域名区域、资源池、授权管理、映射管理、审计日志、账号设置。前端调用 Phase 5 REST API。

## 快速开始

```sh
cd web

# 安装依赖
npm install

# 本地开发（需要先启动后端 frp-warden，前端通过 Vite proxy 调用 /api）
npm run dev

# 类型检查
npm run type-check

# 生产构建 → web/dist
npm run build
```

> 本机 Go 位于 `C:\go_v1.26\bin\go.exe`。后端启动命令：
> `C:\go_v1.26\bin\go.exe build -o frp-warden.exe ./cmd/frp-warden && ./frp-warden.exe`

## 目录结构

```
src/
  api/           API 客户端与各模块接口
  components/    公共组件（预留）
  layouts/       管理后台布局（AdminLayout.vue）
  router/        Vue Router 路由与守卫
  stores/        Pinia 状态管理（auth store）
  views/         页面视图
  types/         TypeScript 类型定义
  utils/         工具函数（预留）
```

## 开发说明

- Vite dev server 默认端口 5173，`/api` 代理到 `http://127.0.0.1:8080`（Go admin server）。
- 前端使用 `fetch` + `credentials:include` 调用后端 session cookie 认证 API。
- 401 时自动跳转登录页。
- `web/node_modules/` 和 `web/dist/` 已被 `.gitignore` 忽略。

## 安全

- 前端不保存 password / plain_token / session token 到 localStorage。
- plain_token 只在创建/重置租户时通过弹窗一次性展示，提醒用户复制保存。
- 不在 console 打印敏感信息。

# 0012 — Vue 3 + Naive UI 管理后台前端

- 状态:已接受
- 日期:2026-06-23

## 背景

Phase 6 需要实现 frp-warden 的管理后台前端,消费 Phase 5 的 REST API。需要选择
合适的前端技术栈,并在轻量、可维护、现代化之间取得平衡。

## 决策与理由

### 为什么使用 Vue 3

Vue 3 是国内前端社区最流行的框架之一,生态成熟、文档完善、学习曲线平缓。项目团队
对 Vue 更熟悉,开发效率更高。Composition API 提供了更好的逻辑复用与类型推导。

### 为什么使用 Vite

Vite 是 Vue 作者开发的下一代构建工具,开发服务器启动极快(HMR 毫秒级),生产构建
基于 Rollup,输出体积小。相比 Webpack,开发体验显著提升,且配置更简洁。

### 为什么使用 TypeScript

TypeScript 提供静态类型检查,在编译时发现类型错误,减少运行时 bug。对于管理后台
这种表单密集、API 调用频繁的应用,类型安全尤为重要。Vue 3 + Vite + TS 是官方推荐
的组合。

### 为什么使用 Naive UI

Naive UI 是一个 Vue 3 原生的组件库,API 设计现代、主题可定制、文档完善。相比
Element Plus / Ant Design Vue,Naive UI 更轻量、更 TypeScript 友好、主题系统更灵活。
项目不需要重型组件库,Naive UI 的 DataTable、Form、Modal 等组件足够覆盖管理后台需求。

### 为什么本轮只做前端页面,不做 Go embed

分阶段推进。Phase 6 先把前端页面做扎实(登录、管理、API 对接),Phase 7 再把 `web/dist`
通过 Go `embed` 内嵌到二进制。先有可靠的 UI,再做内嵌,避免两者同时变动带来的风险。

### 为什么前端不保存 password/token 到 localStorage

localStorage 没有 HttpOnly 保护,容易被 XSS 攻击读取。password 和 token 都是敏感信息,
不应存在 localStorage 中。session cookie 有 HttpOnly 保护,更安全。plain_token 只通过
弹窗一次性展示,提醒用户复制保存。

### 为什么 tenant plain_token 只通过弹窗一次性展示

tenant token 是设备连接 frps 的凭据。创建/重置时明文只在弹窗中显示一次,之后不可再获取。
这样即使前端被入侵,历史 token 也不可获取。弹窗中有明确的"请复制保存"提示。

### 为什么使用 Vite proxy 调试后端 API

开发模式下,前端(Vite dev server,端口 5173)和后端(Go admin server,端口 8080)是
两个独立进程。Vite 的 `server.proxy` 配置将 `/api` 请求代理到后端,避免跨域问题,
且不需要后端配置 CORS。生产环境通过 Go embed 同源部署,也不需要 CORS。

## 影响

- `web/` 目录包含完整的 Vue 3 + Vite + TypeScript + Naive UI 前端项目。
- `web/dist` 已生成,后续 Phase 7 通过 Go `embed` 内嵌。
- 前端使用 `fetch` + `credentials:include` 调用后端 session cookie 认证 API。
- 路由使用 Vue Router(hash 模式);状态管理使用 Pinia。
- 前端代码注释与页面文案以中文为主。

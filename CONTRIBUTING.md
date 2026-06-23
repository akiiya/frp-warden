# 为 frp-warden 贡献

感谢你参与 frp-warden 的开发。本项目是**文档驱动**的:
文档与代码同步演进,每次改动都应让两者保持一致。

## 分支

- 所有开发都在 **`dev`** 分支上进行。不要直接向 `main` 提交。
- `main` 仅保留经过评审、可发布的状态。
- 如果愿意,可以从 `dev` 拉出分支来开发较大的特性(`feature/...`),完成后再合并
  回 `dev`。

## 工作流程

1. 阅读 [CLAUDE.md](CLAUDE.md) / [AGENTS.md](AGENTS.md) 以及你所涉及领域的文档
   (先从 [docs/PROJECT_CONTEXT.md](docs/PROJECT_CONTEXT.md)
   和 [docs/ROADMAP.md](docs/ROADMAP.md) 开始)。
2. 在 [docs/ROADMAP.md](docs/ROADMAP.md) 中确认当前所处的阶段,并把你的
   改动控制在该范围内。
3. 进行改动,**更新相关文档与 `CHANGELOG.md`**。
4. 运行下面的验证命令。
5. 总结你改动了哪些内容(代码文件、文档文件、运行的命令、后续事项)。

## 验证

先检测 Go(它可能位于 `C:\go_v1.26\bin\go.exe`,且未加入 `PATH`)。

```sh
go fmt ./...
go vet ./...
go test ./...
go build ./cmd/frp-warden
```

前端(在 `web/` 下存在后):

```sh
npm install
npm run build
```

## 代码风格

- Go:保持标准的 `gofmt`/`go vet` 无告警。包要小而内聚;与周边代码风格保持一致。
  为导出的符号编写文档。
- 保持依赖精简且无 CGO(单一静态二进制是目标之一 —— 参见
  [docs/DECISIONS/0001-use-go-single-binary.md](docs/DECISIONS/0001-use-go-single-binary.md))。
- 前端:TypeScript、Vue 3 单文件组件(SFC)、Naive UI 组件。

## 安全

在改动鉴权、token、配置或插件端点之前,请先查阅 [docs/SECURITY.md](docs/SECURITY.md)。
切勿以明文存储密钥,切勿将插件端口默认绑定到公网接口,切勿绕过服务端的授权检查。

## 架构决策

重要的决策以 ADR 的形式记录在 [docs/DECISIONS/](docs/DECISIONS/) 中。
当你做出值得记录的决策时,请新增一个带编号的文件。

## 提交

除非明确要求,否则不要代替维护者提交或推送。当你确实需要提交时,
请写清晰的提交信息,并在相关处引用所属阶段。

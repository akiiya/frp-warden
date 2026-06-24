# 0017 — VERSION 文件控制版本与 main 自动发布

- 状态:已接受
- 日期:2026-06-23

## 背景

Phase 9.1 需要调整发布流程:版本号由 VERSION 文件统一控制,dev 合并到 main 后自动
完成测试、构建、创建 tag、创建 Release、上传资产,不再需要人工打 tag。

## 决策与理由

### 为什么使用 VERSION 文件控制版本

版本号是项目的核心元信息,应该有一个明确的来源(单一事实来源)。VERSION 文件让版本号
可审计、可追踪(在 git 历史中可见变更),且不依赖 CI 变量或手动输入。

### 为什么 VERSION 不带 v 前缀

VERSION 文件只存储语义化版本号(`0.1.0-rc1`),tag 名由 workflow 自动加 `v` 前缀
(`v0.1.0-rc1`)。这样 VERSION 文件内容更干净,tag 命名规则统一由 workflow 控制。

### 为什么 dev 合并 main 后自动发布

自动化发布减少人工操作,避免忘记打 tag 或打错 tag。PR 合并到 main 时,CI 已验证代码
质量,Release workflow 接续完成构建和发布,流程无缝衔接。

### 为什么创建 tag 和 Release 必须在同一个 workflow 内完成

使用 `GITHUB_TOKEN` 创建 tag 时,通常不会再次触发另一个 workflow(防止递归触发)。
因此不能设计成"workflow A 创建 tag → 依赖 tag push 触发 workflow B"。正确做法是:
同一个 workflow 内读取 VERSION → 构建 → 创建 tag → 创建 Release → 上传资产。

### 为什么不依赖 workflow 创建 tag 后再触发 tag workflow

如上所述,GITHUB_TOKEN 创建的 tag push 不会触发另一个 workflow。这种设计会导致
Release 流程断裂。单 workflow 内完成所有步骤更可靠。

### 为什么 tag 已存在时要失败,而不是覆盖

覆盖已有 tag 是危险操作:已有 Release 可能已被用户下载,覆盖会导致版本混乱。
明确失败并提示"请更新 VERSION"更安全,也更清晰。

### 为什么正式发布仍通过 PR 修改 VERSION 控制

VERSION 文件是代码的一部分,修改它应该经过正常的代码审查流程(dev → PR → main)。
这保证了每个发布版本都有对应的 PR 记录和 CI 验证。

### 为什么 main 必须继续受保护,只能 PR 合并

main 是发布分支,直接 push 可能引入未经 CI 验证的代码。PR 合并保证代码经过 CI 验证
和 review,Release 产物的质量有保障。

## 影响

- 新增 `VERSION` 文件(初始内容 `0.1.0-rc1`)。
- `.github/workflows/release.yml`:触发条件改为 `push: branches: [main]`,流程为
  读取 VERSION → 测试 → 构建 → 创建 tag → 创建 Release。
- `.github/workflows/build.yml`:CI 验证(push dev / PR main),不创建 tag/Release。
- `docs/GITHUB.md`:更新发布流程说明。
- tag 名格式: `v${VERSION}`。
- tag 已存在时 workflow 明确失败。

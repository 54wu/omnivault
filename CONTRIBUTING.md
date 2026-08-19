# 贡献指南 Contributing

感谢你有兴趣为 **OmniVault · 万象档案袋** 做出贡献。无论你是提交 Issue、修正文档、修复缺陷、添加特性，还是撰写测试，我们都欢迎。项目遵循 [贡献者公约](./CODE_OF_CONDUCT.md)，请确保你的行为合乎约定。

如果只是提问或讨论，优先去 [Discussions](https://github.com/54wu/omnivault/discussions)；如果要报告明确的 Bug 或新功能建议，使用 [Issues](https://github.com/54wu/omnivault/issues)。

## 报告 Bug（Issue）

提交 Issue 前，请先用搜索功能确认没有重复项。

一个高质量的 Bug 报告应包含：

- **问题概述**：发生了什么 / 期望怎样
- **复现步骤**：最小可复现的操作序列
- **运行环境**：操作系统与版本、Go 版本（`go version`）、OmniVault 版本（`omnivault status` 或 Release 标签）
- **日志或报错**：命令行完整输出、故障截图
- **安全相关的问题请勿公开发布**，见 [SECURITY.md](./SECURITY.md)

## 功能建议

在 Issue 中说明你的使用场景、期望行为，以及为什么现有命令/接口无法满足。贴上相关文档链接会很有帮助。

## 环境准备

- **Go 1.26+** 与 git。（Windows 下可选安装 go-winres；`build.ps1` 会在缺失时自动安装）
- 克隆后先在仓库根目录运行 `go build ./...`，确认能编译。
- 运行全部测试：`make test`（等价于 `go test -race ./...`）。

## 分支与开发流程

1. **Fork** 本仓库，从 `main` 拉出一条功能分支：`git checkout -b feat/your-feature main`（Bug 修复用 `fix/xxx`，文档用 `docs/xxx`）。
2. 在分支上开发，保持改动小且聚焦（一个 PR 解决一个问题）。
3. 遵循代码规范（见下），为新逻辑补充单元测试。
4. 本地验证：`make test`、`go vet ./...` 全部通过。
5. **推送分支并提交 Pull Request**，指向 `main`。

## 代码风格与规范

- Go 代码保持 `gofmt` 格式（`gofmt -l .` 输出为空）。
- 通过 `go vet ./...` 无告警；新代码连同 `go test -race ./...` 通过。
- 遵循包内既有命名与结构；`internal/` 下的分层（`crypto` / `store` / `vault` / `api` / `merge` / `dpapi`）请勿随意打散。
- 面向用户的新增/变更，同步更新本 README 与 `README_EN.md` 的对应章节。
- 隐私与安全优先：任何改动都不得把明文密钥或密码写入磁盘、日志或网络。

## 提交信息规范

采用语义化提交（Conventional Commits），格式为 `类型(可选作用域): 描述`：

- `feat: ...` 新功能
- `fix: ...` 缺陷修复
- `docs: ...` 文档
- `test: ...` 测试
- `refactor: ...` 重构
- `chore: ...` 构建/工具/杂务
- `perf: ...` 性能

## Pull Request 流程

1. 标题概括改动；描述中说明**动机**（为什么）与**影响**（改了哪些行为/接口）。
2. 关联相关 Issue（`Closes #123`）。
3. 等待 CI（`go test -race`）通过；必要时补齐因新增接口而失败的测试。
4. 维护者 review 后合入。若需发布新版本，由维护者按「发布（Release）」章节打标签。
5. 如果 PR 长期无人响应，欢迎在评论区友善提醒。

## 安全检查清单

因为本项目核心是**加密与密钥管理**，每次改动请自问：

- [ ] 是否引入任何将密钥/密码落盘、落日志、落网络的路径？
- [ ] 是否绕过或削弱了原有的 0-knowledge 边界？
- [ ] 加密层的改动是否有对应测试？
- [ ] 是否尝试过从攻击者视角（如越权读取他人字段、令牌滥用）审视？

如对你负责的部分不确定，在 PR 描述中说明你的思考即可。
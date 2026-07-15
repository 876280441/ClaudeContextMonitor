# 贡献指南

欢迎贡献！无论是 Bug 反馈、功能建议、文档改进还是代码 PR，都非常欢迎。

## 提交 Issue

- 请先搜索是否已有相同问题。
- 描述清楚：操作系统、Go 版本、复现步骤、期望与实际行为。
- 最好附上命令输出（注意脱敏，Session 内容可能含敏感信息）。

## 提交 PR

1. Fork 仓库并新建分支：`git checkout -b feat/your-feature`。
2. 保证代码通过静态检查：

   ```bash
   gofmt -w .
   go vet ./...
   go build ./...
   ```

3. 风格上：
   - **核心命令（CLI / Web）仅用 Go 标准库，不引入第三方依赖**——这是核心约束。
   - 唯一例外是 `internal/tray`（Windows 系统托盘），使用 `fyne.io/systray` 并以 `//go:build windows` 隔离，不影响其它平台编译。新增类似平台 GUI 能力时请遵循同样的 build-tag 隔离方式。
   - 注释用中文或英文均可，与所在文件已有风格保持一致。
   - 新增公开类型/函数需有注释。
4. 如果改动影响 Token 估算或颜色阈值等行为，请在 PR 说明中写清楚动机。
5. 提交信息简洁清晰，建议符合 [Conventional Commits](https://www.conventionalcommits.org/) 风格。

## 项目结构

参见 README 的"项目结构"一节。核心分层：

- `internal/model` — 数据结构
- `internal/tokenizer` — Token 估算（接口 + Estimate 实现）
- `internal/claude` — 发现与流式解析
- `internal/scanner` — 并发扫描
- `internal/report` — 排名 / 聚合 / 同名消歧 / 全局 Top / ETA
- `internal/ui` — 终端颜色 / 格式化 / 表格
- `internal/web` — Web 仪表盘与 REST API
- `internal/tray` — 系统托盘（Windows，build-tag 隔离）
- `cmd` — CLI 命令实现
- `main.go` — 入口与命令分发

## 行为准则

请保持友善、尊重。对所有贡献者一视同仁。

## 许可证

提交的贡献将按 [MIT License](./LICENSE) 授权。
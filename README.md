<div align="center">

# ClaudeContextMonitor

监控 Claude Code 各 Session 的 **Context 占用** 与 **Token 统计**

![Go](https://img.shields.io/badge/Go-1.20+-00ADD8?logo=go&logoColor=white)
![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)
![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-lightgrey)
![No Dependencies](https://img.shields.io/badge/dependencies-zero-blue)

一个**单 EXE、零依赖**的命令行 + Web 工具，帮你随时掌握：当前 Session 用了多少 Context、还剩多少、多久会填满、哪些 Session / Project 最大、是否逼近 1M Token、是哪条消息导致暴涨。

</div>

<div align="center">

![Web 仪表盘](docs/Snipaste_2026-07-14_17-22-58.png)

*Web 仪表盘：当前会话卡 · 汇总 · Sessions 表（开始时间 / 真实 Token / ETA / 活跃标识）*

</div>

---

## 为什么需要

Claude Code 2.x 使用 1M Context Window，单个 Session 常持续数天甚至数周。原生客户端没有完整的 Context 管理视图——你可能直到"上下文已满"才发现。本工具扫描本地 `~/.claude/projects/**/*.jsonl`，**实时**统计每个 Session 的 Context 占用、剩余空间并按阈值报警。优先读取 Anthropic 返回的**真实 token 计数**（assistant 消息自带的 `usage`），无则回退智能估算。

## 特性

- 🎯 **真实 Token 计数** —— 优先用 assistant 消息自带的 Anthropic `usage`（`input + cache_read + cache_creation`）作为当前上下文，准确而非估算；无 usage 才回退估算（标注 `~`）
- 🚀 **单 EXE、零依赖** —— 纯 Go 标准库，`go build` 出一个可执行文件，拷到任何机器直接跑
- 🧮 **智能估算回退** —— 区分 CJK（≈1 token/字）与拉丁字符（≈4 字符/token），解析 JSON 只统计实际进入上下文的内容（text / thinking / tool_use / tool_result / attachment）
- ⏱️ **预计填满时间** —— 用近期真实 usage 采样算增速，外推"约 X 小时填满"，直接回答"会不会快满"
- 🔴 **活跃会话识别** —— 自动锁定最近活跃的会话并置顶高亮，回答"我这个会话还剩多少"
- 🏷️ **项目名同名消歧** —— 同名但不同目录的项目自动追加父级（如 `LWJK_V8/LWJK_APP` 与 `PHP/LWJK_APP`），不再混淆或错误合并
- 🌊 **流式 + 并发** —— 逐行解析（不限行长度，超大 tool_result 不崩溃），多 Session 并发扫描；只累积统计量，内存恒定
- 🎨 **颜色报警** —— `<80%` 绿 / `80%` 黄 / `90%` 橙 / `95%` 红 / `≥100%` 爆红 ERROR；Windows 自动启用虚拟终端
- 🖥️ **Web 仪表盘** —— 当前会话卡 / 进度条 / 开始时间 / ETA / 搜索 / 过滤 / 排序 / 标签页状态图标 / 全局 Top 消息 / 自动刷新 + 浏览器通知 + 端口自动避让
- 📦 **多命令 CLI** —— list / top / detail / project / export csv / watch / serve / messages

## 目录

- [安装](#安装)
- [快速开始](#快速开始)
- [命令详解](#命令详解)
- [Web 仪表盘](#web-仪表盘serve)
- [全局选项](#全局选项)
- [Token 计数说明](#token-计数说明)
- [性能](#性能)
- [项目结构](#项目结构)
- [路线图](#路线图)
- [贡献](#贡献)
- [许可证](#许可证)

## 安装

### 方式一：从源码构建（推荐）

```bash
git clone https://github.com/USER/claude-context-monitor.git
cd claude-context-monitor
go build -o ClaudeContext.exe .
```

> 📌 本仓库的 Go module 路径使用了占位符 `github.com/USER/claude-context-monitor`。
> Fork 后请把 `USER` 替换为你的 GitHub 用户名（一行命令）：
>
> ```bash
> # Windows (Git Bash)
> find . -name '*.go' -print0 | xargs -0 sed -i 's|github.com/USER/|github.com/<你的用户名>/|g'
> sed -i 's|github.com/USER/|github.com/<你的用户名>/|' go.mod
> ```
>
> 这样别人才能 `go install` 直装。不替换也不影响 `go build`。

### 方式二：go install

> 前提：module 路径已指向真实仓库。

```bash
go install github.com/USER/claude-context-monitor@latest
```

### 环境要求

- Go 1.20+（仅用标准库，无任何第三方依赖）
- 运行平台：Windows 10/11（主目标），Linux / macOS 同样可用

## 快速开始

```bash
# 列出所有 Session，按 Token 降序
ClaudeContext.exe

# 实时刷新（默认 3 秒）
ClaudeContext.exe watch

# 启动 Web 仪表盘，浏览器自动打开
ClaudeContext.exe serve

# 全局最大的 20 条消息（定位 Context 暴涨）
ClaudeContext.exe messages
```

默认扫描 `~/.claude`（Windows 下为 `%USERPROFILE%\.claude`）。

## 命令详解

| 命令 | 说明 |
|------|------|
| `(无)` / `list` | 列出所有 Session，按 Token 降序（默认） |
| `top [N]` | 显示 Token 最大的前 N 个 Session（默认 10） |
| `detail <sessionid>` | 单个 Session 详细统计 + Top 最大消息（支持 id 前缀匹配） |
| `project` | Project 排名（会话数 / 总 Token / 最大 Session） |
| `export csv [file]` | 导出 CSV（默认 `claude-context.csv`，带 UTF-8 BOM 兼容 Excel） |
| `watch [interval]` | 实时刷新（默认 `3s`，例：`watch 2.5s`） |
| `serve [addr]` | 启动 **Web 仪表盘**（默认 `127.0.0.1:8765`，端口被占自动避让，浏览器自动打开） |
| `messages [N]` | 全局最大的前 N 条消息（默认 20），定位 Context 暴涨来源 |
| `help` | 帮助 |

### 示例：默认列表

```
Claude Context Monitor
------------------------------------------------------------
Project      Session     Size   Tokens   Used  Remaining  Status
MYAPP        a1b2c3d4  21.6MB  663,975  66.4%    336,025  ●
DEMO         5e6f7a8b   3.8MB  180,582  18.1%    819,418  ●
WEBSITE      9f8e7d6c   2.6MB  235,945  23.6%    764,055  ●
...
------------------------------------------------------------
Total Sessions: 48   Total Tokens: 3.9M   (Max Context: 1,000,000)
```

> Token 列默认显示**真实**值（来自 `usage`）；纯估算的会话前缀 `~`。

### 示例：单 Session 详情

```
Claude Context Monitor — Session Detail
------------------------------------------------------------
  Project           MYAPP
  Session           a1b2c3d4-1111-2222-3333-444455556666
  Path              /home/user/projects/myapp
  File Size         21.6MB
  Messages          1755 (User 593 / Assistant 1162)
  Attachments       191   Tool Use 549 / Tool Result 547
  Estimated Tokens  877,000
  Context           663,975   real: in 743 + cache 663,232 + create 0
  Used              66.4% [●]
  Remaining         336,025
  ETA to Full       约 28.4 小时  (按近期增速)
  Started           07-02 01:08
  Last Active       07-13 18:11
------------------------------------------------------------
  ███████████████████████████████████░░░░░ 663,975 / 1,000,000  (66.4%)
------------------------------------------------------------
Top Largest Messages in this session
 #  Kind       Tokens  Preview
 1  assistant  12,751  Let me analyze this carefully. The user's YAML file...
 2  assistant  12,667  Write {"file_path":"/home/user/projects/myapp/...
 ...
```

> **Context** 行展示真实 token（来自 Anthropic `usage`：本轮新增 `in` + 命中缓存 `cache` + 写入缓存 `create`）。无 usage 的会话此处显示 `~估算值 (estimated)`。Top 消息列表帮你**定位是哪次聊天导致 Context 暴涨**。

### 示例：全局 Top 消息

```bash
ClaudeContext.exe messages 10
```

```
Claude Context Monitor — Top 10 Largest Messages (global)
------------------------------------------------------------
#  Project           Session   Kind       Tokens  Preview
1  MYAPP             a1b2c3d4  user       25,465  <system-reminder>[Truncated: PARTIAL view...
2  DEMO/SUB          5e6f7a8b  user       20,477  1 <?php 2  3 namespace app\services; 4  5 use...
...
```

## Web 仪表盘（`serve`）

纯标准库 `net/http` 实现，HTML/JS 通过 `go:embed` 打包进 exe，不新增任何依赖。

```bash
ClaudeContext.exe serve            # 启动并自动打开浏览器 → http://127.0.0.1:8765/
ClaudeContext.exe serve 9999       # 指定端口（被占自动 +1 避让）
ClaudeContext.exe --max-context 500000 serve   # 全局默认 max-context
```

**功能：**
- **当前会话卡**：自动锁定最近活跃会话，大进度条 + 剩余 + ETA + 活跃标识
- 汇总卡：Total Sessions / Total Tokens / 活跃会话 / ≥90% / ≥95%
- Sessions 表：项目（短名 + 灰色完整路径）、Session、**开始时间**、大小、Token、**带颜色进度条的 Used%**、剩余、**ETA**、状态；**点击行弹出详情 + Top 消息**
- **搜索**（项目/会话/路径）+ **过滤**（全部 / 仅活跃 / ≥90% / ≥95%）+ **表头点击排序**（Token / 最近活跃 / 开始时间 / 大小…）
- **全局 Top 消息**表：跨所有会话的最大消息，点击跳转详情
- Project 排名表（同名项目自动消歧、按目录拆分）
- **标签页标题 + favicon**：动态显示最高告警等级（🔴 66% · 1 活跃），不切回标签也能瞄一眼
- 自动刷新（1 / 3 / 5 / 10s 可选）、Max Context 控件实时覆盖
- 超 95% / 达 100% 时**浏览器原生通知 + 顶部告警条 + 可选提示音**（仅活跃会话弹通知，避免陈旧会话刷屏）
- **端口自动避让**：默认端口被占时自动 +1 重试（最多 20 个）

**REST API**（均支持 `?max_context=` / `?include_sidechain=` 覆盖；服务端预算 used%/level/status/eta）：

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/overview` | 汇总（含活跃会话数） |
| GET | `/api/sessions?limit=` | Session 列表（降序，含 start_time / eta / active / tokens_source） |
| GET | `/api/session/{id}` | 单 Session 详情（前缀匹配）+ Top 消息 + 真实 usage 明细 |
| GET | `/api/projects` | Project 排名（同名消歧） |
| GET | `/api/messages?limit=` | 全局 Top 消息 |

## 全局选项

| 选项 | 默认 | 说明 |
|------|------|------|
| `--max-context N` | `1000000` | Context 上限 |
| `--dir PATH` | `~/.claude` | .claude 目录 |
| `--include-sidechain` | 关 | 纳入子 agent（sidechain）上下文（默认排除，因其在独立上下文） |
| `--no-color` | 关 | 关闭颜色 |
| `--limit N` | 不限 | 列表/表格最多显示 N 行 |

## Token 计数说明

工具扫描 `%USERPROFILE%\.claude\projects\**\*.jsonl`（真实聊天内容所在；`history.jsonl`、`sessions/*.json` 仅索引，不参与统计）。

**真实模式（优先）**：每条 assistant 消息自带 Anthropic `usage`，取末条的

```
当前上下文 = input_tokens + cache_creation_input_tokens + cache_read_input_tokens
```

作为该会话的真实 Context 大小（已含系统提示与缓存命中，远比估算准确）。`Used%` / `Remaining` / 排名 / 报警均优先用此值；同时保留近期 (时间, 真实 token) 采样用于**增速与预计填满时间**。

**估算模式（回退）**：无 `usage` 的会话（极少数老会话）回退为解析 JSON 统计实际进入上下文的内容（`message.content`、`assistant.content` 的 text / thinking / tool_use 输入、`tool_result`、`thinking`、`attachment`），按字符类型估算：

```
tokens ≈ CJK字符数 × 1.0 + 拉丁字符数 × 0.25 + 其它多字节 × 0.5
```

估算值在表格中前缀 `~`、CSV 中 `TokensSource=estimate` 标注。系数为包级变量（`internal/tokenizer/estimate.go` 的 `CoefCJK` / `CoefLatin` / `CoefOther`），可调。`Tokenizer` 接口已预留 Exact 模式接入位。

## 性能

| 指标 | 结果 |
|------|------|
| 真实数据（48 Session / ~42MB） | ~0.5s |
| 压测（40 Session / 865MB） | ~2.4s |
| 峰值内存（865MB 数据） | ~32MB |

- 设计支持 **1000+ Session / 20GB JSONL 不崩溃**；普通规模（~1GB 内）扫描 <3s。
- 吞吐受标准库 `encoding/json` 限制（约 350–400 MB/s）。极端 20GB 规模约需 1 分钟，但内存恒定（仅随 Session 数线性增长，约 MB 级）。

## 项目结构

```
main.go                      入口：参数解析、命令分发、Windows VT 初始化
internal/
├── model/types.go           SessionStats / UsageInfo（真实 usage）/ SamplePoint（增速采样）等
├── claude/
│   ├── discovery.go         发现 projects/**/*.jsonl、解析项目名
│   └── parser.go            流式逐行解析 → SessionStats（真实 usage + Top-N 消息 + 采样）
├── tokenizer/
│   ├── tokenizer.go         Tokenizer 接口
│   └── estimate.go          字节级 CJK/拉丁智能估算（回退用）
├── scanner/scanner.go       并发扫描编排（worker pool）
├── report/report.go         排名 / 聚合 / 同名消歧 / 全局 Top 消息 / ETA 估算
├── web/
│   ├── server.go            Web 服务 + REST API + 1s 缓存 + 端口自动避让
│   └── dashboard.html       内嵌单页面（go:embed）
└── ui/
    ├── color.go             颜色规则 + 等级
    ├── color_windows.go     Windows VT 启用（syscall）
    ├── format.go            数字 / 进度条 / 百分比 / 时间 / 时长
    └── table.go             对齐表格
cmd/                         list/top/detail/project/export/watch/serve/messages 命令实现
docs/                        设计文档（需求稿）
```

## 路线图

**已完成**：真实 Token 计数 · 智能估算回退 · 项目名同名消歧 · 会话开始时间 · 活跃会话识别 · 预计填满时间 · 全局 Top 消息 · Web 仪表盘 + REST API · 搜索/过滤/排序 · 标签页状态图标 · 端口自动避让 · CSV 导出 · 颜色报警 + 浏览器通知

**待办**：GUI / Windows 托盘常驻 · 历史趋势图 / Session 增长曲线 · 每日 Token 增长统计 · 官方 Anthropic tokenizer 接入（Exact 模式）

## 贡献

欢迎贡献！详见 [CONTRIBUTING.md](./CONTRIBUTING.md)。核心约束：**仅用 Go 标准库，不引入第三方依赖**。

## 许可证

[MIT License](./LICENSE) © ClaudeContextMonitor Contributors
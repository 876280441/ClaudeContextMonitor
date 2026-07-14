// ClaudeContextMonitor：监控 Claude Code 各 Session 的 Context 占用与 Token 预估。
//
// 用法：
//
//	ClaudeContext.exe [全局选项] [命令] [命令参数]
//
// 命令：
//
//	(无)/list   列出所有 Session，按 Token 降序（默认）
//	top [N]     显示 Token 最大的前 N 个 Session（默认 10）
//	detail <id> 单个 Session 详细统计（支持 id 前缀）
//	project     Project 排名
//	export csv [file]  导出 CSV（默认 claude-context.csv）
//	watch [interval]   实时刷新（默认 3s）
//	serve [addr]       启动 Web 仪表盘（默认 127.0.0.1:8765）
//	messages [N]       全局最大的前 N 条消息（默认 20）
//
// 全局选项：
//
//	--max-context N     Context 上限，默认 1000000
//	--dir PATH          .claude 目录，默认 %USERPROFILE%\.claude
//	--include-sidechain 纳入子 agent（sidechain）上下文
//	--no-color          关闭颜色
//	--limit N           列表/表格最多显示 N 行
package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/USER/claude-context-monitor/cmd"
	"github.com/USER/claude-context-monitor/internal/ui"
)

func main() {
	cfg := cmd.NewConfig()

	positionals, err := parseFlags(cfg, os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		fmt.Fprintln(os.Stderr)
		printUsage(os.Stderr)
		os.Exit(2)
	}

	cfg.ApplyColor()

	command := ""
	if len(positionals) > 0 {
		command = positionals[0]
	}
	cmdArgs := positionals
	if len(cmdArgs) > 0 {
		cmdArgs = cmdArgs[1:]
	}

	run, ok := commands()[strings.ToLower(command)]
	if !ok {
		if command == "" {
			run = cmd.RunList
		} else {
			fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", command)
			printUsage(os.Stderr)
			os.Exit(2)
		}
	}

	if err := run(cfg, cmdArgs); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func commands() map[string]func(*cmd.Config, []string) error {
	return map[string]func(*cmd.Config, []string) error{
		"":         cmd.RunList,
		"list":     cmd.RunList,
		"top":      cmd.RunTop,
		"detail":   cmd.RunDetail,
		"project":  cmd.RunProject,
		"export":   cmd.RunExport,
		"watch":    cmd.RunWatch,
		"serve":    cmd.RunServe,
		"messages": cmd.RunMessages,
		"help": func(cfg *cmd.Config, _ []string) error {
			printUsage(cfg.Out)
			return nil
		},
	}
}

// boolFlags 返回不需要取值的全局开关。
func boolFlags() map[string]bool {
	return map[string]bool{
		"--include-sidechain": true,
		"--no-color":          true,
	}
}

// valueFlags 返回需要取值的全局选项及其解析回调。返回 false 表示该 flag 不被识别（按位置参数处理）。
func parseFlags(cfg *cmd.Config, args []string) ([]string, error) {
	bf := boolFlags()
	var positionals []string
	i := 0
	for i < len(args) {
		a := args[i]
		if a == "-h" || a == "--help" {
			positionals = append(positionals, "help")
			i++
			continue
		}
		if strings.HasPrefix(a, "--") {
			key := a
			val := ""
			inline := false
			if idx := strings.Index(a, "="); idx >= 0 {
				key, val, inline = a[:idx], a[idx+1:], true
			}
			if bf[key] {
				applyBool(cfg, key, true)
				i++
				continue
			}
			// 需要取值的 flag
			if !inline {
				if i+1 >= len(args) {
					return nil, fmt.Errorf("flag %s requires a value", key)
				}
				val = args[i+1]
				i += 2
			} else {
				i++
			}
			if err := applyValue(cfg, key, val); err != nil {
				return nil, err
			}
			continue
		}
		// 非 flag：位置参数（可能是命令或命令参数）
		positionals = append(positionals, a)
		i++
	}
	return positionals, nil
}

func applyBool(cfg *cmd.Config, key string, v bool) {
	switch key {
	case "--include-sidechain":
		cfg.IncludeSidechain = v
	case "--no-color":
		cfg.NoColor = v
	}
}

func applyValue(cfg *cmd.Config, key, val string) error {
	switch key {
	case "--max-context":
		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil || n <= 0 {
			return fmt.Errorf("invalid --max-context %q", val)
		}
		cfg.MaxContext = n
	case "--dir", "--claude-dir":
		cfg.ClaudeDir = val
	case "--limit":
		n, err := strconv.Atoi(val)
		if err != nil || n < 0 {
			return fmt.Errorf("invalid --limit %q", val)
		}
		cfg.Limit = n
	case "--watch-limit":
		n, err := strconv.Atoi(val)
		if err != nil || n < 0 {
			return fmt.Errorf("invalid --watch-limit %q", val)
		}
		cfg.WatchLimit = n
	default:
		// 未识别的带值 flag：忽略（兼容未来扩展），不影响命令解析。
	}
	return nil
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, ui.Cyan("ClaudeContextMonitor"))
	fmt.Fprintln(w, "Monitor Claude Code session context usage and token estimates.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  ClaudeContext.exe [options] <command> [args]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  (default)/list        List all sessions sorted by tokens")
	fmt.Fprintln(w, "  top [N]               Show top N sessions by tokens (default 10)")
	fmt.Fprintln(w, "  detail <sessionid>    Detailed stats for a session (prefix match)")
	fmt.Fprintln(w, "  project               Project ranking")
	fmt.Fprintln(w, "  export csv [file]     Export sessions to CSV")
	fmt.Fprintln(w, "  watch [interval]      Live refresh (e.g. 3s)")
	fmt.Fprintln(w, "  serve [addr]          Start web dashboard (default 127.0.0.1:8765)")
	fmt.Fprintln(w, "  messages [N]          Top N largest messages across all sessions")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  --max-context N       Context limit (default 1000000)")
	fmt.Fprintln(w, "  --dir PATH            .claude directory (default <USERPROFILE>\\.claude)")
	fmt.Fprintln(w, "  --include-sidechain   Include sub-agent (sidechain) context")
	fmt.Fprintln(w, "  --no-color            Disable colored output")
	fmt.Fprintln(w, "  --limit N             Max rows in list/table output")
}

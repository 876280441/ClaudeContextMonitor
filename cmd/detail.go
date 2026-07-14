package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/USER/claude-context-monitor/internal/report"
	"github.com/USER/claude-context-monitor/internal/ui"
)

// RunDetail 显示单个 Session 的详细统计与 Top 最大消息。
// 用法：detail <sessionid>（支持前缀匹配）
func RunDetail(cfg *Config, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ClaudeContext.exe detail <sessionid>")
	}
	res, err := Load(cfg)
	if err != nil {
		return err
	}
	s, exact := report.FindSession(res.Sessions, args[0])
	if s == nil {
		return fmt.Errorf("no session matches id/prefix %q", args[0])
	}

	w := cfg.out()
	lvl := ui.LevelFor(s.Used())
	status, _ := ui.StatusLabel(s.Used())
	names := report.ComputeProjectDisplayNames(res.Sessions)

	fmt.Fprintln(w, ui.Cyan("Claude Context Monitor — Session Detail"))
	fmt.Fprintln(w, ui.Divider(0))

	kv := [][2]string{
		{"Project", report.DisplayName(names, s)},
		{"Session", s.SessionID},
		{"Path", s.Cwd},
		{"File Size", ui.FormatSize(s.FileSize)},
		{"Messages", fmt.Sprintf("%d", s.TotalMessages())},
		{"  User", fmt.Sprintf("%d", s.UserMsgCount)},
		{"  Assistant", fmt.Sprintf("%d", s.AssistantMsgCount)},
		{"Attachments", fmt.Sprintf("%d", s.AttachmentCount)},
		{"Tool Use", fmt.Sprintf("%d", s.ToolUseCount)},
		{"Tool Result", fmt.Sprintf("%d", s.ToolResultCount)},
		{"Estimated Tokens", ui.FormatTokensFull(s.Tokens)},
		{"Used", ui.Color(ui.FormatPercent(s.Used())+" ["+status+"]", lvl)},
		{"Remaining", ui.FormatTokensFull(s.Remaining())},
		{"Started", ui.FormatTime(s.StartTime)},
		{"Last Active", ui.FormatTime(s.ModTime)},
	}
	if !exact {
		fmt.Fprintln(w, ui.Dim(fmt.Sprintf("(matched by prefix: %s)", s.SessionID)))
	}
	printKV(w, kv)
	if s.SidechainTokens > 0 {
		fmt.Fprintf(w, "  %s %s\n", ui.Dim("Sidechain Tokens (excluded):"), ui.FormatTokensFull(s.SidechainTokens))
	}
	fmt.Fprintln(w, ui.Divider(0))

	// 进度条
	barW := 40
	fmt.Fprintf(w, "  %s %s\n", ui.Bar(s.Used()/100, barW, lvl),
		ui.Color(fmt.Sprintf("%s / %s  (%s)", ui.FormatTokensFull(s.Tokens), ui.FormatTokensFull(cfg.MaxContext), ui.FormatPercent(s.Used())), lvl))
	fmt.Fprintln(w, ui.Divider(0))

	// Top 最大消息
	if len(s.TopMessages) > 0 {
		fmt.Fprintln(w, ui.Magenta("Top Largest Messages in this session"))
		tbl := &ui.Table{
			Headers: []string{"#", "Kind", "Tokens", "Preview"},
			Aligns:  []ui.Align{ui.AlignRight, ui.AlignLeft, ui.AlignRight, ui.AlignLeft},
		}
		for i, m := range s.TopMessages {
			tbl.Rows = append(tbl.Rows, []string{
				fmt.Sprintf("%d", i+1),
				m.Kind,
				ui.FormatTokensFull(m.Tokens),
				m.Preview,
			})
		}
		tbl.Print(w)
	} else {
		fmt.Fprintln(w, ui.Dim("(no messages)"))
	}
	return nil
}

// printKV 以 "键: 值" 形式左对齐打印键值对。
func printKV(w io.Writer, kv [][2]string) {
	maxKey := 0
	for _, p := range kv {
		if len(p[0]) > maxKey {
			maxKey = len(p[0])
		}
	}
	for _, p := range kv {
		key := p[0] + strings.Repeat(" ", maxKey-len(p[0]))
		fmt.Fprintf(w, "  %s  %s\n", ui.Dim(key), p[1])
	}
}

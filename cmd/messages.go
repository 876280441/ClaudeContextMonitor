package cmd

import (
	"fmt"
	"strconv"

	"github.com/USER/claude-context-monitor/internal/report"
	"github.com/USER/claude-context-monitor/internal/ui"
)

// RunMessages 显示全局最大的前 N 条消息（跨所有会话聚合）。
// 用法：messages [N]（默认 20）
// 用于定位"哪次聊天导致 Context 暴涨"。
func RunMessages(cfg *Config, args []string) error {
	n := 20
	if len(args) > 0 {
		if v, err := strconv.Atoi(args[0]); err == nil && v > 0 {
			n = v
		}
	}
	res, err := Load(cfg)
	if err != nil {
		return err
	}
	msgs := report.GlobalTopMessages(res.Sessions, n)

	w := cfg.out()
	fmt.Fprintln(w, ui.Cyan(fmt.Sprintf("Claude Context Monitor — Top %d Largest Messages (global)", n)))
	fmt.Fprintln(w, ui.Divider(0))
	tbl := &ui.Table{
		Headers: []string{"#", "Project", "Session", "Kind", "Tokens", "Preview"},
		Aligns:  []ui.Align{ui.AlignRight, ui.AlignLeft, ui.AlignLeft, ui.AlignLeft, ui.AlignRight, ui.AlignLeft},
	}
	for i, m := range msgs {
		tbl.Rows = append(tbl.Rows, []string{
			fmt.Sprintf("%d", i+1),
			m.Project,
			ui.ShortID(m.SessionID, 8),
			m.Kind,
			ui.FormatTokensFull(m.Tokens),
			m.Preview,
		})
	}
	tbl.Print(w)
	return nil
}

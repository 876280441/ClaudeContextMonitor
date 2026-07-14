package cmd

import (
	"fmt"

	"github.com/USER/claude-context-monitor/internal/model"
	"github.com/USER/claude-context-monitor/internal/report"
	"github.com/USER/claude-context-monitor/internal/ui"
)

// renderSessionTable 渲染 Session 列表表格（root 与 top 共用）。
// sessions 为要显示的（可能已截断的）会话；names 需基于【全部】会话计算，
// 以保证同名消歧在不同窗口下一致。
func renderSessionTable(cfg *Config, sessions []*model.SessionStats, names map[string]string, title string) {
	w := cfg.out()
	max := cfg.MaxContext

	fmt.Fprintln(w, ui.Cyan("Claude Context Monitor"))
	if title != "" {
		fmt.Fprintln(w, ui.Dim(title))
	}
	fmt.Fprintln(w, ui.Divider(0))

	tbl := &ui.Table{
		Headers: []string{"Project", "Session", "Size", "Tokens", "Used", "Remaining", "Status"},
		Aligns:  []ui.Align{ui.AlignLeft, ui.AlignLeft, ui.AlignRight, ui.AlignRight, ui.AlignRight, ui.AlignRight, ui.AlignLeft},
		RowLevel: func(i int) ui.Level {
			return ui.LevelFor(sessions[i].Used())
		},
	}
	rows := sessions
	if cfg.Limit > 0 && len(rows) > cfg.Limit {
		rows = rows[:cfg.Limit]
	}
	for _, s := range rows {
		status, _ := ui.StatusLabel(s.Used())
		tok := ui.FormatTokensFull(s.ContextTokens())
		if !s.HasRealTokens() {
			tok = "~" + tok // 纯估算，无真实 usage
		}
		tbl.Rows = append(tbl.Rows, []string{
			report.DisplayName(names, s),
			ui.ShortID(s.SessionID, 8),
			ui.FormatSize(s.FileSize),
			tok,
			ui.FormatPercent(s.Used()),
			ui.FormatTokensFull(s.Remaining()),
			status,
		})
	}
	tbl.Print(w)

	fmt.Fprintln(w, ui.Divider(0))
	total := report.TotalTokens(sessions)
	fmt.Fprintf(w, "Total Sessions: %s   Total Tokens: %s   (Max Context: %s)\n",
		ui.Bold(fmt.Sprintf("%d", len(sessions))),
		ui.Bold(ui.FormatTokens(total)),
		ui.FormatTokensFull(max))
}

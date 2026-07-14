package cmd

import (
	"fmt"

	"github.com/USER/claude-context-monitor/internal/model"
	"github.com/USER/claude-context-monitor/internal/report"
	"github.com/USER/claude-context-monitor/internal/ui"
)

// renderSessionTable 渲染 Session 列表表格（root 与 top 共用）。
// sessions 应已按需排序。
func renderSessionTable(cfg *Config, sessions []*model.SessionStats, title string) {
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
	names := report.ComputeProjectDisplayNames(sessions)
	for _, s := range rows {
		status, _ := ui.StatusLabel(s.Used())
		tbl.Rows = append(tbl.Rows, []string{
			report.DisplayName(names, s),
			ui.ShortID(s.SessionID, 8),
			ui.FormatSize(s.FileSize),
			ui.FormatTokensFull(s.Tokens),
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

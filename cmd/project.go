package cmd

import (
	"fmt"

	"github.com/USER/claude-context-monitor/internal/report"
	"github.com/USER/claude-context-monitor/internal/ui"
)

// RunProject 显示 Project 排名：会话数、总 Token、最大 Session。
func RunProject(cfg *Config, _ []string) error {
	res, err := Load(cfg)
	if err != nil {
		return err
	}
	projects := report.SortProjectsByTokens(report.AggregateProjects(res.Sessions))

	w := cfg.out()
	fmt.Fprintln(w, ui.Cyan("Claude Context Monitor — Project Ranking"))
	fmt.Fprintln(w, ui.Divider(0))

	tbl := &ui.Table{
		Headers: []string{"Project", "Sessions", "Total Tokens", "Largest Session", "Largest Tokens"},
		Aligns:  []ui.Align{ui.AlignLeft, ui.AlignRight, ui.AlignRight, ui.AlignLeft, ui.AlignRight},
		RowLevel: func(i int) ui.Level {
			return ui.LevelFor(float64(projects[i].LargestTokens) / float64(cfg.MaxContext) * 100)
		},
	}
	for _, p := range projects {
		tbl.Rows = append(tbl.Rows, []string{
			p.Name,
			fmt.Sprintf("%d", p.SessionCount),
			ui.FormatTokens(p.TotalTokens),
			ui.ShortID(p.LargestSession, 8),
			ui.FormatTokens(p.LargestTokens),
		})
	}
	tbl.Print(w)

	fmt.Fprintln(w, ui.Divider(0))
	total := report.TotalTokens(res.Sessions)
	fmt.Fprintf(w, "Projects: %s   Total Sessions: %s   Total Tokens: %s\n",
		ui.Bold(fmt.Sprintf("%d", len(projects))),
		ui.Bold(fmt.Sprintf("%d", len(res.Sessions))),
		ui.Bold(ui.FormatTokens(total)))
	return nil
}

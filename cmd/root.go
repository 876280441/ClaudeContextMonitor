package cmd

import (
	"fmt"

	"github.com/USER/claude-context-monitor/internal/report"
	"github.com/USER/claude-context-monitor/internal/ui"
)

// RunList 是默认命令：列出所有 Session，按 Token 降序。
func RunList(cfg *Config, _ []string) error {
	res, err := Load(cfg)
	if err != nil {
		return err
	}
	sorted := report.SortSessionsByTokens(res.Sessions)
	names := report.ComputeProjectDisplayNames(sorted)
	renderSessionTable(cfg, sorted, names, fmt.Sprintf("All sessions (sorted by tokens) — %d found", len(sorted)))
	if res.ParseErrors > 0 {
		fmt.Fprintln(cfg.out(), ui.Dim(fmt.Sprintf("Note: %d line(s) failed to parse and were skipped.", res.ParseErrors)))
	}
	return nil
}

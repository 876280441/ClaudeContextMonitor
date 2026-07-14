package cmd

import (
	"fmt"
	"strconv"

	"github.com/USER/claude-context-monitor/internal/report"
)

// RunTop 显示 Token 最大的前 N 个 Session（默认 10）。
// 用法：top [N]
func RunTop(cfg *Config, args []string) error {
	n := 10
	if len(args) > 0 {
		if v, err := strconv.Atoi(args[0]); err == nil && v > 0 {
			n = v
		}
	}
	res, err := Load(cfg)
	if err != nil {
		return err
	}
	top := report.Top(res.Sessions, n)
	names := report.ComputeProjectDisplayNames(res.Sessions)
	renderSessionTable(cfg, top, names, fmt.Sprintf("Top %d sessions by tokens", n))
	return nil
}

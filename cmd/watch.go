package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/USER/claude-context-monitor/internal/model"
	"github.com/USER/claude-context-monitor/internal/report"
	"github.com/USER/claude-context-monitor/internal/ui"
)

// RunWatch 实时刷新显示，每隔几秒扫描一次。
// 用法：watch [interval]（interval 如 3s / 2.5s，默认 3s）
func RunWatch(cfg *Config, args []string) error {
	interval := 3 * time.Second
	if len(args) > 0 {
		if d, err := time.ParseDuration(args[0]); err == nil && d >= time.Second {
			interval = d
		}
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	tick := time.NewTicker(interval)
	defer tick.Stop()

	render := func() {
		res, err := Load(cfg)
		if err != nil {
			fmt.Fprintf(cfg.out(), "\x1b[2J\x1b[Herror: %v\n", err)
			return
		}
		sorted := report.SortSessionsByTokens(res.Sessions)
		renderWatchFrame(cfg, sorted, interval)
	}

	render() // 首帧立即渲染
	for {
		select {
		case <-stop:
			fmt.Fprintln(cfg.out(), "\nStopped.")
			return nil
		case <-tick.C:
			render()
		}
	}
}

// renderWatchFrame 渲染 watch 单帧：清屏 + 标题 + 每会话进度条 + 汇总 + 报警。
func renderWatchFrame(cfg *Config, sessions []*model.SessionStats, interval time.Duration) {
	w := cfg.out()

	// 清屏并归位光标
	fmt.Fprint(w, "\x1b[2J\x1b[H")

	fmt.Fprintln(w, ui.Cyan("Claude Context Monitor — Watch")+
		ui.Dim(fmt.Sprintf("  (refresh every %s, max context %s)", interval, ui.FormatTokensFull(cfg.MaxContext))))
	fmt.Fprintln(w, ui.Divider(0))

	limit := cfg.WatchLimit
	rows := sessions
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}

	for _, s := range rows {
		lvl := ui.LevelFor(s.Used())
		status, _ := ui.StatusLabel(s.Used())
		bar := ui.Bar(s.Used()/100, 24, lvl)
		line := fmt.Sprintf("%-16s %s  %s  %s/%s  %s %s",
			truncate(s.Project, 16),
			ui.ShortID(s.SessionID, 8),
			bar,
			ui.FormatTokens(s.Tokens),
			ui.FormatTokens(cfg.MaxContext),
			ui.FormatPercent(s.Used()),
			status,
		)
		fmt.Fprintln(w, ui.Color(line, lvl))
	}
	if len(sessions) > len(rows) {
		fmt.Fprintln(w, ui.Dim(fmt.Sprintf("... and %d more sessions (use 'top' or default list to see all)", len(sessions)-len(rows))))
	}

	fmt.Fprintln(w, ui.Divider(0))
	total := report.TotalTokens(sessions)
	over := 0
	for _, s := range sessions {
		if ui.LevelFor(s.Used()) >= ui.LevelRed {
			over++
		}
	}
	summary := fmt.Sprintf("Total Sessions: %s   Total Tokens: %s",
		ui.Bold(fmt.Sprintf("%d", len(sessions))),
		ui.Bold(ui.FormatTokens(total)))
	fmt.Fprintln(w, summary)
	if over > 0 {
		fmt.Fprintln(w, ui.Color(fmt.Sprintf("⚠  %d session(s) above 95%% context!", over), ui.LevelRed))
	}

	// 底部提示
	fmt.Fprintln(w, ui.Dim(strings.TrimSpace("Press Ctrl+C to exit.")))
}

// truncate 按显示宽度截断字符串，超长加省略号。
func truncate(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-1] + "…"
}

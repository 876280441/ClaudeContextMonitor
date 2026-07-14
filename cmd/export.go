package cmd

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/USER/claude-context-monitor/internal/model"
	"github.com/USER/claude-context-monitor/internal/report"
)

// RunExport 导出所有 Session 为 CSV。
// 用法：export csv [file]（默认 claude-context.csv）
func RunExport(cfg *Config, args []string) error {
	filename := "claude-context.csv"
	// 支持 "export csv path" 和 "export path" 两种写法
	for _, a := range args {
		if a == "csv" {
			continue
		}
		if filename == "claude-context.csv" {
			filename = a
		}
	}

	res, err := Load(cfg)
	if err != nil {
		return err
	}
	sorted := report.SortSessionsByTokens(res.Sessions)

	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create %s: %w", filename, err)
	}
	defer f.Close()

	// 写入 UTF-8 BOM，便于 Excel 正确识别中文。
	if _, err := f.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return err
	}
	w := csv.NewWriter(f)
	defer w.Flush()

	header := []string{
		"Project", "SessionID", "Path", "FileSize", "Messages",
		"UserMsg", "AssistantMsg", "Attachments", "ToolUse", "ToolResult",
		"Tokens", "TokensSource", "UsedPct", "Remaining", "MaxContext",
		"StartTime", "ModTime", "ParseErrors",
	}
	if err := w.Write(header); err != nil {
		return err
	}
	names := report.ComputeProjectDisplayNames(sorted)
	for _, s := range sorted {
		st := ""
		if !s.StartTime.IsZero() {
			st = s.StartTime.Format(time.RFC3339)
		}
		mt := ""
		if !s.ModTime.IsZero() {
			mt = s.ModTime.Format(time.RFC3339)
		}
		row := []string{
			report.DisplayName(names, s),
			s.SessionID,
			s.Cwd,
			strconv.FormatInt(s.FileSize, 10),
			strconv.Itoa(s.TotalMessages()),
			strconv.Itoa(s.UserMsgCount),
			strconv.Itoa(s.AssistantMsgCount),
			strconv.Itoa(s.AttachmentCount),
			strconv.Itoa(s.ToolUseCount),
			strconv.Itoa(s.ToolResultCount),
			strconv.FormatInt(s.ContextTokens(), 10),
			tokensSource(s),
			fmt.Sprintf("%.2f", s.Used()),
			strconv.FormatInt(s.Remaining(), 10),
			strconv.FormatInt(cfg.MaxContext, 10),
			st,
			mt,
			strconv.Itoa(s.ParseErrors),
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}
	fmt.Fprintf(cfg.out(), "Exported %d sessions to %s\n", len(sorted), filename)
	return nil
}

func tokensSource(s *model.SessionStats) string {
	if s.HasRealTokens() {
		return "real"
	}
	return "estimate"
}

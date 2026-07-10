// Package cmd 实现 CLI 各子命令。
package cmd

import (
	"io"
	"os"

	"github.com/USER/claude-context-monitor/internal/claude"
	"github.com/USER/claude-context-monitor/internal/scanner"
	"github.com/USER/claude-context-monitor/internal/tokenizer"
	"github.com/USER/claude-context-monitor/internal/ui"
)

// Config 是全局配置，由 main 解析后传入各命令。
type Config struct {
	ClaudeDir        string
	MaxContext       int64
	IncludeSidechain bool
	NoColor          bool
	Limit            int // 列表/表格行数上限；<=0 表示不限
	WatchLimit       int // watch 模式显示的 Session 行数
	Out              io.Writer
}

// NewConfig 返回带默认值的配置。
func NewConfig() *Config {
	return &Config{
		ClaudeDir:  claude.DefaultClaudeDir(),
		MaxContext: 1_000_000,
		Limit:      0,
		WatchLimit: 15,
		Out:        os.Stdout,
	}
}

// ApplyColor 根据 NoColor 配置同步到 ui 包。
func (c *Config) ApplyColor() {
	ui.SetColorEnabled(!c.NoColor)
}

// Load 发现并扫描所有 Session，返回结果。
func Load(cfg *Config) (*scanner.Result, error) {
	files, err := claude.Discover(cfg.ClaudeDir)
	if err != nil {
		return nil, err
	}
	tok := tokenizer.NewEstimate()
	return scanner.Scan(files, tok, scanner.Options{
		MaxContext:       cfg.MaxContext,
		IncludeSidechain: cfg.IncludeSidechain,
	}), nil
}

// out 返回输出目标（便于测试时替换）。
func (c *Config) out() io.Writer {
	if c.Out == nil {
		return os.Stdout
	}
	return c.Out
}

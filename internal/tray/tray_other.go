//go:build !windows

package tray

import (
	"fmt"

	"github.com/USER/claude-context-monitor/internal/web"
)

// Run 在非 Windows 平台返回错误（托盘仅支持 Windows）。
func Run(srv *web.Server, url string, maxContext int64, includeSidechain bool) error {
	return fmt.Errorf("tray command is only supported on Windows")
}

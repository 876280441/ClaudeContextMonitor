package cmd

import (
	"fmt"
	"net/http"

	"github.com/USER/claude-context-monitor/internal/tray"
	"github.com/USER/claude-context-monitor/internal/web"
)

// RunTray 启动系统托盘常驻：后台运行 Web 仪表盘 + 托盘图标监控与报警。
// 用法：tray [addr]（addr 为端口或 host:port，默认 127.0.0.1:8765）
func RunTray(cfg *Config, args []string) error {
	addr := ""
	if len(args) > 0 {
		addr = args[0]
	}
	listen := web.ListenAddr(addr)
	srv := web.NewServer(cfg.ClaudeDir, cfg.MaxContext, cfg.IncludeSidechain)

	ln, used, err := web.AcquireListener(listen, 20)
	if err != nil {
		return err
	}
	url := "http://" + used + "/"

	// 后台运行 Web 仪表盘
	go func() {
		_ = http.Serve(ln, srv.Handler())
	}()

	fmt.Fprintf(cfg.out(), "Claude Context Monitor — Tray\n")
	if used != listen {
		fmt.Fprintf(cfg.out(), "  %s 被占用，自动改用 %s\n", listen, used)
	}
	fmt.Fprintf(cfg.out(), "  dashboard: %s\n", url)
	fmt.Fprintf(cfg.out(), "  data dir : %s   max ctx: %s\n", cfg.ClaudeDir, fmtInt(cfg.MaxContext))
	fmt.Fprintf(cfg.out(), "  托盘已驻留：右键菜单可打开仪表盘/退出，Ctrl+C 也可退出。\n")

	// 阻塞：托盘消息循环（退出菜单返回后结束）
	return tray.Run(srv, url, cfg.MaxContext, cfg.IncludeSidechain)
}

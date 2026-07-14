package cmd

import (
	"fmt"
	"net/http"

	"github.com/USER/claude-context-monitor/internal/web"
)

// RunServe 启动 Web 仪表盘服务。
// 用法：serve [addr]（addr 为端口号或 host:port，默认 127.0.0.1:8765）
// 若端口被占用，自动 +1 避让，最多尝试 20 个端口。
func RunServe(cfg *Config, args []string) error {
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

	fmt.Fprintf(cfg.out(), "Claude Context Monitor — Web Dashboard\n")
	if used != listen {
		fmt.Fprintf(cfg.out(), "  %s 被占用，自动改用 %s\n", listen, used)
	}
	fmt.Fprintf(cfg.out(), "  listening: %s\n", url)
	fmt.Fprintf(cfg.out(), "  data dir : %s\n", cfg.ClaudeDir)
	fmt.Fprintf(cfg.out(), "  max ctx  : %s   (include-sidechain: %v)\n",
		fmtInt(cfg.MaxContext), cfg.IncludeSidechain)
	fmt.Fprintf(cfg.out(), "  Ctrl+C to stop.\n")

	// 尝试自动打开浏览器（失败不影响服务）。
	if err := web.OpenBrowser(url); err != nil {
		fmt.Fprintf(cfg.out(), "  (could not auto-open browser, open the URL manually)\n")
	}

	return http.Serve(ln, srv.Handler())
}

func fmtInt(n int64) string {
	in := fmt.Sprintf("%d", n)
	// 简单千位分隔
	out := ""
	for i, c := range in {
		if i > 0 && (len(in)-i)%3 == 0 {
			out += ","
		}
		out += string(c)
	}
	return out
}

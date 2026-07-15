//go:build windows

package tray

import (
	"encoding/binary"
	"fmt"
	"os/exec"
	"time"

	"fyne.io/systray"

	"github.com/USER/claude-context-monitor/internal/ui"
	"github.com/USER/claude-context-monitor/internal/web"
)

// levelBGRA 返回某告警等级对应的图标颜色（B,G,R,A）。
func levelBGRA(l ui.Level) [4]byte {
	switch l {
	case ui.LevelGreen:
		return [4]byte{78, 197, 34, 255}
	case ui.LevelYellow:
		return [4]byte{8, 179, 234, 255}
	case ui.LevelOrange:
		return [4]byte{22, 115, 249, 255}
	case ui.LevelRed:
		return [4]byte{68, 68, 239, 255}
	case ui.LevelBurst:
		return [4]byte{45, 45, 255, 255}
	}
	return [4]byte{255, 168, 74, 255} // idle 蓝
}

// coloredICO 生成一个 size×size、32bpp 的纯色实心方块 ICO（BMP 条目），
// 兼容 systray 在 Windows 下使用的 LoadImage(IMAGE_ICON, LR_LOADFROMFILE)。
func coloredICO(c [4]byte, size int) []byte {
	xor := make([]byte, size*size*4)
	for i := 0; i < size*size; i++ {
		xor[i*4+0] = c[0] // B
		xor[i*4+1] = c[1] // G
		xor[i*4+2] = c[2] // R
		xor[i*4+3] = c[3] // A
	}
	andRowBytes := ((size + 31) / 32) * 4
	andMask := make([]byte, andRowBytes*size) // 全 0 = 不透明
	bmpSize := 40 + len(xor) + len(andMask)

	bih := make([]byte, 40)
	binary.LittleEndian.PutUint32(bih[0:4], 40)
	binary.LittleEndian.PutUint32(bih[4:8], uint32(size))
	binary.LittleEndian.PutUint32(bih[8:12], uint32(size*2)) // ICO BMP 高度=2×（XOR+AND）
	binary.LittleEndian.PutUint16(bih[12:14], 1)             // planes
	binary.LittleEndian.PutUint16(bih[14:16], 32)            // bpp

	hdr := make([]byte, 6)
	binary.LittleEndian.PutUint16(hdr[2:4], 1) // type=ICO
	binary.LittleEndian.PutUint16(hdr[4:6], 1) // count=1
	entry := make([]byte, 16)
	if size >= 256 {
		entry[0], entry[1] = 0, 0 // 0 表示 256
	} else {
		entry[0], entry[1] = byte(size), byte(size)
	}
	binary.LittleEndian.PutUint16(entry[4:6], 1)  // planes
	binary.LittleEndian.PutUint16(entry[6:8], 32) // bpp
	binary.LittleEndian.PutUint32(entry[8:12], uint32(bmpSize))
	binary.LittleEndian.PutUint32(entry[12:16], 22) // offset = 6+16

	out := make([]byte, 0, 6+16+bmpSize)
	out = append(out, hdr...)
	out = append(out, entry...)
	out = append(out, bih...)
	out = append(out, xor...)
	out = append(out, andMask...)
	return out
}

// Run 启动系统托盘（阻塞，直到用户选择"退出"）。srv 用于复用扫描缓存。
func Run(srv *web.Server, url string, maxContext int64, includeSidechain bool) error {
	t := &trayApp{srv: srv, url: url, maxContext: maxContext, isc: includeSidechain}
	systray.Run(t.onReady, t.onExit)
	return nil
}

type trayApp struct {
	srv        *web.Server
	url        string
	maxContext int64
	isc        bool
	prevLevel  ui.Level
	mOpen      *systray.MenuItem
	mRefresh   *systray.MenuItem
	mQuit      *systray.MenuItem
}

func (t *trayApp) onReady() {
	systray.SetIcon(coloredICO(levelBGRA(ui.LevelGreen), 32))
	systray.SetTooltip("Claude Context Monitor — scanning…")
	t.mOpen = systray.AddMenuItem("打开仪表盘", "在浏览器打开 Web 仪表盘")
	systray.AddSeparator()
	t.mRefresh = systray.AddMenuItem("立即刷新", "立即重新扫描")
	t.mQuit = systray.AddMenuItem("退出", "退出程序")

	// 单击/双击图标打开仪表盘
	systray.SetOnTapped(func() { _ = web.OpenBrowser(t.url) })

	// 菜单事件循环
	go func() {
		for {
			select {
			case <-t.mOpen.ClickedCh:
				_ = web.OpenBrowser(t.url)
			case <-t.mRefresh.ClickedCh:
				t.refresh()
			case <-t.mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()

	t.refresh() // 首次立即
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			t.refresh()
		}
	}()
}

func (t *trayApp) onExit() {}

func (t *trayApp) refresh() {
	res := t.srv.Scan(t.maxContext, t.isc)
	st := ComputeStatus(res, t.maxContext)

	systray.SetIcon(coloredICO(levelBGRA(st.Level), 32))
	tip := fmt.Sprintf("Claude Context · %s · %d 活跃", ui.FormatPercent(st.MaxUsed), st.ActiveCount)
	if st.Over95 > 0 {
		tip = fmt.Sprintf("⚠ %d 会话 ≥95%% · 最高 %s · %d 活跃", st.Over95, ui.FormatPercent(st.MaxUsed), st.ActiveCount)
	}
	systray.SetTooltip(tip)

	// 阈值跨越（首次进入 ≥95%）弹 Windows toast
	if st.Level >= ui.LevelRed && t.prevLevel < ui.LevelRed {
		go toast("Claude Context 告警", fmt.Sprintf("%d 个会话超过 95%%，最高 %s", st.Over95, ui.FormatPercent(st.MaxUsed)))
	}
	t.prevLevel = st.Level
}

// toast 通过 PowerShell 调用 Windows 原生 toast 通知（best-effort，失败静默）。
func toast(title, body string) {
	script := fmt.Sprintf(`[Windows.UI.Notifications.ToastNotificationManager,Windows.UI.Notifications,ContentType=WindowsRuntime] | Out-Null
$ErrorActionPreference='SilentlyContinue'
$t=[Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02)
$x=$t.GetXml()
$n=$x.GetElementsByTagName('text')
[void]$n.Item(0).AppendChild($x.CreateTextNode('%s'))
[void]$n.Item(1).AppendChild($x.CreateTextNode('%s'))
$to=[Windows.UI.Notifications.ToastNotification]::new($x)
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('ClaudeContextMonitor').Show($to)`,
		escapePS(title), escapePS(body))
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	if err := cmd.Start(); err != nil {
		return
	}
	go func() { _ = cmd.Wait() }()
}

// escapePS 转义 PowerShell 单引号字符串中的单引号。
func escapePS(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\'' {
			out = append(out, '\'', '\'')
		} else {
			out = append(out, s[i])
		}
	}
	return string(out)
}

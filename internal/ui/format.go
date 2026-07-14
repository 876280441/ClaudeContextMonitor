package ui

import (
	"fmt"
	"strings"
	"time"
)

// FormatTokens 将 Token 数格式化为紧凑形式：>=1e6 用 M、>=1e3 用 K。
// 例：8200000 → "8.2M"；954321 → "954.3K"；52111 → "52111"。
func FormatTokens(n int64) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

// FormatTokensFull 返回带千位分隔的完整数字。
func FormatTokensFull(n int64) string {
	s := fmt.Sprintf("%d", n)
	// 简单的千位分隔
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	}
	if len(s) <= 3 {
		if neg {
			return "-" + s
		}
		return s
	}
	var b strings.Builder
	pre := len(s) % 3
	if pre > 0 {
		b.WriteString(s[:pre])
		if len(s) > pre {
			b.WriteByte(',')
		}
	}
	for i := pre; i < len(s); i += 3 {
		b.WriteString(s[i : i+3])
		if i+3 < len(s) {
			b.WriteByte(',')
		}
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

// FormatSize 将字节数格式化为人类可读（B/KB/MB/GB）。
func FormatSize(bytes int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.1fGB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.1fMB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.0fKB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

// FormatPercent 格式化百分比，保留 1 位小数。
func FormatPercent(pct float64) string {
	return fmt.Sprintf("%.1f%%", pct)
}

// ShortID 返回 ID 的前 n 个字符（默认 8）。
func ShortID(id string, n int) string {
	if n <= 0 {
		n = 8
	}
	if len(id) <= n {
		return id
	}
	return id[:n]
}

// Bar 渲染进度条（width 个单元），frac 被裁剪到 [0,1]。
// 返回带颜色的字符串（颜色由调用方按等级传入）。
func Bar(frac float64, width int, l Level) string {
	if width < 1 {
		width = 1
	}
	if frac < 0 {
		frac = 0
	} else if frac > 1 {
		frac = 1
	}
	filled := int(frac*float64(width) + 0.5)
	if filled > width {
		filled = width
	}
	if !colorEnabled {
		return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	}
	return levelColor(l) + strings.Repeat("█", filled) + ansiGray + strings.Repeat("░", width-filled) + ansiReset
}

// Divider 返回指定宽度的分隔线。
func Divider(width int) string {
	if width < 1 {
		width = 60
	}
	return strings.Repeat("-", width)
}

// FormatTime 将时间格式化为紧凑显示：零值返回 "-"；当年显示 "MM-DD HH:MM"，
// 跨年显示 "YYYY-MM-DD"。用于会话开始/更新时间列。
func FormatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	now := time.Now()
	if t.Year() == now.Year() {
		return t.Format("01-02 15:04")
	}
	return t.Format("2006-01-02")
}

// Package ui 负责终端输出：颜色规则、数字与进度条格式化、对齐表格。
package ui

import "fmt"

// 颜色开关，默认开启；Windows 实现会在检测到非控制台（输出被重定向）时关闭。
var colorEnabled = true

// SetColorEnabled 全局开关颜色输出。
func SetColorEnabled(v bool) { colorEnabled = v }

// ColorEnabled 返回当前是否启用颜色。
func ColorEnabled() bool { return colorEnabled }

// Level 表示上下文使用率的告警等级。
type Level int

const (
	LevelGreen  Level = iota // <80%
	LevelYellow              // 80%–<90%
	LevelOrange              // 90%–<95%
	LevelRed                 // 95%–<100%
	LevelBurst               // >=100% 爆红
)

// LevelFor 根据使用百分比返回告警等级。
func LevelFor(usedPct float64) Level {
	switch {
	case usedPct >= 100:
		return LevelBurst
	case usedPct >= 95:
		return LevelRed
	case usedPct >= 90:
		return LevelOrange
	case usedPct >= 80:
		return LevelYellow
	default:
		return LevelGreen
	}
}

// LevelName 返回等级中文名。
func LevelName(l Level) string {
	switch l {
	case LevelGreen:
		return "OK"
	case LevelYellow:
		return "WARN"
	case LevelOrange:
		return "HIGH"
	case LevelRed:
		return "CRIT"
	case LevelBurst:
		return "ERROR"
	}
	return ""
}

// ANSI 转义码。
const (
	ansiReset   = "\x1b[0m"
	ansiBold    = "\x1b[1m"
	ansiDim     = "\x1b[2m"
	ansiGreen   = "\x1b[32m"
	ansiYellow  = "\x1b[33m"
	ansiOrange  = "\x1b[38;5;208m" // 256 色：橙
	ansiRed     = "\x1b[31m"
	ansiBurst   = "\x1b[1;31m" // 加粗红
	ansiCyan    = "\x1b[36m"
	ansiMagenta = "\x1b[35m"
	ansiGray    = "\x1b[90m"
)

func levelColor(l Level) string {
	switch l {
	case LevelGreen:
		return ansiGreen
	case LevelYellow:
		return ansiYellow
	case LevelOrange:
		return ansiOrange
	case LevelRed:
		return ansiRed
	case LevelBurst:
		return ansiBurst
	}
	return ""
}

// Color 用指定等级包裹字符串（颜色关闭时原样返回）。
func Color(s string, l Level) string {
	if !colorEnabled {
		return s
	}
	return levelColor(l) + s + ansiReset
}

// ColorStr 用任意 ANSI 颜色包裹字符串。
func ColorStr(s, code string) string {
	if !colorEnabled {
		return s
	}
	return code + s + ansiReset
}

// Bold 加粗。
func Bold(s string) string { return ColorStr(s, ansiBold) }

// Dim 灰暗。
func Dim(s string) string { return ColorStr(s, ansiDim) }

// Cyan 着青色。
func Cyan(s string) string { return ColorStr(s, ansiCyan) }

// Gray 着灰色。
func Gray(s string) string { return ColorStr(s, ansiGray) }

// Magenta 着品红。
func Magenta(s string) string { return ColorStr(s, ansiMagenta) }

// StatusLabel 返回状态标签文本（如 "●"/"ERROR"）及其等级。
func StatusLabel(usedPct float64) (string, Level) {
	l := LevelFor(usedPct)
	if l == LevelBurst {
		return "ERROR", l
	}
	return "●", l
}

// sprintf 版本的颜色辅助。
func Colorf(l Level, format string, a ...any) string {
	return Color(fmt.Sprintf(format, a...), l)
}

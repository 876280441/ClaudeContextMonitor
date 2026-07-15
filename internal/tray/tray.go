// Package tray 提供系统托盘常驻能力（Windows）。
//
// 通过 build tag 隔离平台实现：Windows 用 fyne.io/systray，其它平台为 stub。
// 本文件存放跨平台共享的状态计算逻辑。
package tray

import (
	"github.com/USER/claude-context-monitor/internal/scanner"
	"github.com/USER/claude-context-monitor/internal/ui"
)

// Status 是托盘展示所需的聚合状态。
type Status struct {
	MaxUsed       float64 // 全部会话中最高的使用率
	Level         ui.Level
	ActiveCount   int // 近期活跃会话数
	Over90        int
	Over95        int
	TotalSessions int
}

// ComputeStatus 由扫描结果计算托盘展示状态。
func ComputeStatus(res *scanner.Result, maxContext int64) Status {
	var st Status
	st.TotalSessions = len(res.Sessions)
	for _, s := range res.Sessions {
		u := s.Used()
		lvl := ui.LevelFor(u)
		if u > st.MaxUsed {
			st.MaxUsed = u
			st.Level = lvl
		}
		if s.IsActive() {
			st.ActiveCount++
		}
		if lvl >= ui.LevelOrange {
			st.Over90++
		}
		if lvl >= ui.LevelRed {
			st.Over95++
		}
	}
	if st.TotalSessions == 0 {
		st.Level = ui.LevelGreen
	}
	return st
}

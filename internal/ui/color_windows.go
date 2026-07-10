//go:build windows

package ui

import (
	"syscall"
	"unsafe"
)

// 在 Windows 上启用虚拟终端处理（VT100/ANSI 转义序列），并检测输出是否为控制台。
// 使用 syscall 直接调用 kernel32，避免引入外部依赖。
func init() {
	enableWindowsVT()
}

func enableWindowsVT() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getMode := kernel32.NewProc("GetConsoleMode")
	setMode := kernel32.NewProc("SetConsoleMode")

	const enableVTProcessing = 0x0004
	handle := syscall.Handle(syscall.Stdout)

	var mode uint32
	r, _, _ := getMode.Call(uintptr(handle), uintptr(unsafe.Pointer(&mode)))
	if r == 0 {
		// GetConsoleMode 失败：标准输出被重定向到文件/管道，关闭颜色。
		colorEnabled = false
		return
	}
	mode |= enableVTProcessing
	setMode.Call(uintptr(handle), uintptr(mode))
}

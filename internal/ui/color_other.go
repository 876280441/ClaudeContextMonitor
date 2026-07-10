//go:build !windows

package ui

// 非 Windows 平台：现代终端原生支持 ANSI 转义序列，无需特殊处理。
func init() {}

package ui

import (
	"fmt"
	"io"
	"strings"
)

// Table 是一个简单的对齐表格打印器。支持表头、按列对齐、可选的行级着色回调。
type Table struct {
	Headers []string
	Aligns  []Align // 每列对齐方式；长度可小于列数，缺省为 AlignLeft
	Rows    [][]string
	// RowLevel 返回某一行应使用的颜色等级（nil 表示不着色）。
	RowLevel func(rowIdx int) Level
}

// Align 列对齐方式。
type Align int

const (
	AlignLeft Align = iota
	AlignRight
)

// Print 将表格写到 w（不使用颜色时输出纯文本）。
func (t *Table) Print(w io.Writer) {
	if len(t.Headers) == 0 && len(t.Rows) == 0 {
		return
	}
	nCols := len(t.Headers)
	if len(t.Rows) > 0 && len(t.Rows[0]) > nCols {
		nCols = len(t.Rows[0])
	}
	widths := make([]int, nCols)
	for c := 0; c < nCols; c++ {
		if c < len(t.Headers) {
			widths[c] = displayWidth(t.Headers[c])
		}
	}
	for _, row := range t.Rows {
		for c := 0; c < nCols && c < len(row); c++ {
			if dw := displayWidth(row[c]); dw > widths[c] {
				widths[c] = dw
			}
		}
	}

	align := func(c int) Align {
		if c < len(t.Aligns) {
			return t.Aligns[c]
		}
		return AlignLeft
	}

	// 表头
	if len(t.Headers) > 0 {
		var sb strings.Builder
		for c := 0; c < nCols; c++ {
			if c > 0 {
				sb.WriteString("  ")
			}
			h := ""
			if c < len(t.Headers) {
				h = t.Headers[c]
			}
			sb.WriteString(padCell(h, widths[c], align(c)))
		}
		fmt.Fprintln(w, Dim(sb.String()))
	}

	for r, row := range t.Rows {
		var line strings.Builder
		for c := 0; c < nCols; c++ {
			if c > 0 {
				line.WriteString("  ")
			}
			cell := ""
			if c < len(row) {
				cell = row[c]
			}
			line.WriteString(padCell(cell, widths[c], align(c)))
		}
		if t.RowLevel != nil {
			fmt.Fprintln(w, Color(line.String(), t.RowLevel(r)))
		} else {
			fmt.Fprintln(w, line.String())
		}
	}
}

// displayWidth 计算字符串显示宽度（去除 ANSI 转义；CJK 宽字符按 2 计）。
func displayWidth(s string) int {
	w := 0
	inEsc := false
	for _, r := range s {
		if r == 0x1b {
			inEsc = true
			continue
		}
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		if isWide(r) {
			w += 2
		} else {
			w++
		}
	}
	return w
}

// isWide 判断 rune 是否为终端中的宽字符（CJK 等）。
func isWide(r rune) bool {
	return (r >= 0x1100 && r <= 0x115F) ||
		(r >= 0x2E80 && r <= 0xA4CF) ||
		(r >= 0xAC00 && r <= 0xD7A3) ||
		(r >= 0xF900 && r <= 0xFAFF) ||
		(r >= 0xFE30 && r <= 0xFE4F) ||
		(r >= 0xFF00 && r <= 0xFF60) ||
		(r >= 0xFFE0 && r <= 0xFFE6) ||
		(r >= 0x1F300 && r <= 0x1FAFF)
}

// padCell 按对齐方式填充到指定宽度。注意：当 cell 含 ANSI 转义时，
// 填充按显示宽度计算，补足的空格加在 ANSI 序列之外。
func padCell(cell string, width int, a Align) string {
	dw := displayWidth(cell)
	if dw >= width {
		return cell
	}
	pad := strings.Repeat(" ", width-dw)
	if a == AlignRight {
		return pad + cell
	}
	return cell + pad
}

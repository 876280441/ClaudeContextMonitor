package tokenizer

import "math"

// 估算系数（经验值，可调）。设计目标：中文≈1 token/字，英文≈4 字符/token。
var (
	CoefCJK            = 1.0  // 每个 CJK 字符约 1 token
	CoefLatin          = 0.25 // 约 4 字符/token（英文/ASCII）
	CoefOther          = 0.5  // 其他多字节字符（西里尔/阿拉伯/emoji 等）
	PerMessageOverhead = 4    // 每条消息的结构/角色开销（token）
)

// Estimate 是默认的估算模式实现。
type Estimate struct{}

// NewEstimate 创建一个估算模式 tokenizer。
func NewEstimate() *Estimate { return &Estimate{} }

// Mode 返回 "estimate"。
func (e *Estimate) Mode() string { return "estimate" }

// Estimate 返回给定文本的 Token 估算值。
// 采用字节级遍历：ASCII 走快速分支，仅多字节字符解码 rune 判定 CJK，
// 保证大文件（数十 MB）下的吞吐。
func (e *Estimate) Estimate(s string) int {
	cjk, latin, other := classify(s)
	if cjk == 0 && latin == 0 && other == 0 {
		return 0
	}
	t := float64(cjk)*CoefCJK + float64(latin)*CoefLatin + float64(other)*CoefOther
	if t < 1 {
		t = 1
	}
	return int(math.Round(t))
}

// EstimateBytes 直接估算字节切片（避免 string 拷贝）。语义与 Estimate 相同。
func (e *Estimate) EstimateBytes(p []byte) int {
	cjk, latin, other := classifyBytes(p)
	if cjk == 0 && latin == 0 && other == 0 {
		return 0
	}
	t := float64(cjk)*CoefCJK + float64(latin)*CoefLatin + float64(other)*CoefOther
	if t < 1 {
		t = 1
	}
	return int(math.Round(t))
}

// classify 按 CJK / 拉丁 / 其他 三类统计字符数（基于 string）。
func classify(s string) (cjk, latin, other int) {
	n := len(s)
	i := 0
	for i < n {
		b := s[i]
		if b < 0x80 {
			latin++
			i++
			continue
		}
		r, size := decodeRune(s, i)
		if isCJK(r) {
			cjk++
		} else {
			other++
		}
		i += size
	}
	return
}

// classifyBytes 与 classify 相同，但作用于 []byte。
func classifyBytes(p []byte) (cjk, latin, other int) {
	n := len(p)
	i := 0
	for i < n {
		b := p[i]
		if b < 0x80 {
			latin++
			i++
			continue
		}
		r, size := decodeRuneBytes(p, i)
		if isCJK(r) {
			cjk++
		} else {
			other++
		}
		i += size
	}
	return
}

// decodeRune 从 string 的位置 i 解码一个 rune（已确保 s[i]>=0x80）。
// 返回 rune 与该字符的字节长度。非法字节按 1 字节处理并返回 RuneError。
func decodeRune(s string, i int) (rune, int) {
	b := s[i]
	var r rune
	var size int
	switch {
	case b&0xE0 == 0xC0:
		r, size = rune(b&0x1F), 2
	case b&0xF0 == 0xE0:
		r, size = rune(b&0x0F), 3
	case b&0xF8 == 0xF0:
		r, size = rune(b&0x07), 4
	default:
		// 非法前导字节，按单字节处理。
		return 0xFFFD, 1
	}
	if i+size > len(s) {
		return 0xFFFD, 1
	}
	for j := 1; j < size; j++ {
		c := s[i+j]
		if c&0xC0 != 0x80 {
			return 0xFFFD, 1 // 续字节非法
		}
		r = r<<6 | rune(c&0x3F)
	}
	return r, size
}

func decodeRuneBytes(p []byte, i int) (rune, int) {
	b := p[i]
	var r rune
	var size int
	switch {
	case b&0xE0 == 0xC0:
		r, size = rune(b&0x1F), 2
	case b&0xF0 == 0xE0:
		r, size = rune(b&0x0F), 3
	case b&0xF8 == 0xF0:
		r, size = rune(b&0x07), 4
	default:
		return 0xFFFD, 1
	}
	if i+size > len(p) {
		return 0xFFFD, 1
	}
	for j := 1; j < size; j++ {
		c := p[i+j]
		if c&0xC0 != 0x80 {
			return 0xFFFD, 1
		}
		r = r<<6 | rune(c&0x3F)
	}
	return r, size
}

// isCJK 判断 rune 是否属于中日韩相关区间。
func isCJK(r rune) bool {
	return (r >= 0x3000 && r <= 0x4DBF) || // CJK 符号/标点、平假名、片假名、CJK 扩展A
		(r >= 0x4E00 && r <= 0x9FFF) || // CJK 统一表意文字
		(r >= 0xAC00 && r <= 0xD7AF) || // 谚文音节
		(r >= 0xF900 && r <= 0xFAFF) || // CJK 兼容表意
		(r >= 0xFF00 && r <= 0xFFEF) // 全角形式
}

// Package tokenizer 提供 Token 估算能力。
//
// 由于目前没有 Anthropic 官方的 Go 语言 tokenizer，默认使用 Estimate 模式
// （区分 CJK 与拉丁字符的字节级启发式估算）。接口预留 Exact 模式，未来官方
// tokenizer 发布后可实现该接口平滑接入。
package tokenizer

// Tokenizer 是 Token 计算的抽象接口。
type Tokenizer interface {
	// Estimate 返回给定文本的 Token 估算值。
	Estimate(text string) int
	// Mode 返回当前模式名："estimate" / "exact"。
	Mode() string
}

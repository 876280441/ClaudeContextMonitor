// Package model 定义 ClaudeContextMonitor 的核心数据结构。
package model

import "time"

// EntryType 是 jsonl 单行的 type 字段。仅 user/assistant/attachment 计入内容统计。
type EntryType string

const (
	TypeUser       EntryType = "user"
	TypeAssistant  EntryType = "assistant"
	TypeAttachment EntryType = "attachment"
	TypeToolResult EntryType = "tool_result"
	TypeThinking   EntryType = "thinking"
	TypeText       EntryType = "text"
	TypeToolUse    EntryType = "tool_use"
)

// MessageStat 记录单条消息（user 或 assistant）的统计信息，用于 Top-N 最大消息。
type MessageStat struct {
	Kind    string // "user" / "assistant"
	Tokens  int64  // 该条消息估算 Token
	Preview string // 内容预览（前若干字符，用于定位）
}

// SessionStats 是单个 Session（一个 jsonl 文件）解析后的统计结果。
type SessionStats struct {
	// 标识
	SessionID string // 文件名（去 .jsonl），规范化来源
	Project   string // 项目显示名（cwd 末段大写，回退目录名）
	Cwd       string // 记录中读取到的 cwd（取最后一条非空）

	// 文件信息
	FilePath  string
	FileSize  int64
	ModTime   time.Time // 文件最后修改时间
	StartTime time.Time // 会话开始时间（首条记录的 timestamp）

	// 消息计数
	UserMsgCount      int
	AssistantMsgCount int
	AttachmentCount   int
	ToolUseCount      int // tool_use block 数量
	ToolResultCount   int // tool_result block 数量
	ParseErrors       int // 解析失败行数（容错统计）

	// Token 与 Context
	Tokens          int64 // 估算总 Token（主上下文）
	MaxContext      int64 // Context 上限，来自 --max-context
	SidechainTokens int64 // 被排除的 sidechain Token（参考）

	// Top-N 最大消息（按 Token 降序，最多 TopN 条）
	TopMessages []MessageStat
}

// Used 返回上下文使用百分比（0-100+）。
func (s *SessionStats) Used() float64 {
	if s.MaxContext <= 0 {
		return 0
	}
	return float64(s.Tokens) / float64(s.MaxContext) * 100
}

// Remaining 返回剩余 Token。
func (s *SessionStats) Remaining() int64 {
	r := s.MaxContext - s.Tokens
	if r < 0 {
		return 0
	}
	return r
}

// TotalMessages 返回 user + assistant 消息总数。
func (s *SessionStats) TotalMessages() int {
	return s.UserMsgCount + s.AssistantMsgCount
}

// LargestMessage 返回该 Session 中最大的单条消息（无则零值）。
func (s *SessionStats) LargestMessage() MessageStat {
	if len(s.TopMessages) == 0 {
		return MessageStat{}
	}
	return s.TopMessages[0]
}

// ProjectStats 是单个 Project 的聚合统计。
type ProjectStats struct {
	Name           string
	SessionCount   int
	TotalTokens    int64
	LargestSession string // 最大 Session 的 SessionID
	LargestTokens  int64  // 最大 Session 的 Token
}

// TopN 是默认维护的最大消息数量上限。
const TopN = 10

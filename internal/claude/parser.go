package claude

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/USER/claude-context-monitor/internal/model"
	"github.com/USER/claude-context-monitor/internal/tokenizer"
)

// ParseOptions 控制解析行为。
type ParseOptions struct {
	MaxContext       int64
	IncludeSidechain bool
}

// rawEntry 对应 jsonl 单行的结构（仅保留需要的字段，content 用 RawMessage 延迟解析）。
type rawEntry struct {
	Type        string          `json:"type"`
	Message     *rawMessage     `json:"message"`
	Attachment  json.RawMessage `json:"attachment"`
	Cwd         string          `json:"cwd"`
	SessionID   string          `json:"sessionId"`
	Timestamp   string          `json:"timestamp"`
	IsSidechain bool            `json:"isSidechain"`
	IsMeta      bool            `json:"isMeta"`
}

type rawMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
	Usage   *rawUsage       `json:"usage"`
}

// rawUsage 对应 assistant 消息的 usage 字段（Anthropic 真实 token 计数）。
type rawUsage struct {
	InputTokens              int64 `json:"input_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
}

// contentBlock 描述 message.content 数组中的一个块（text/thinking/tool_use/tool_result）。
type contentBlock struct {
	Type     string          `json:"type"`
	Text     string          `json:"text"`
	Thinking string          `json:"thinking"`
	Name     string          `json:"name"` // tool_use 的工具名
	Input    json.RawMessage `json:"input"`
	Content  json.RawMessage `json:"content"` // tool_result 内容：字符串或数组
}

// ParseFile 流式解析单个 Session jsonl 文件，返回统计结果。
// 使用 bufio.Reader.ReadBytes 逐行读取（直接得 []byte，避免 string→[]byte 拷贝），
// 不限制行长度，避免超大 tool_result 触发缓冲上限。
func ParseFile(sf SessionFile, tok tokenizer.Tokenizer, opts ParseOptions) (*model.SessionStats, error) {
	f, err := os.Open(sf.Path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fi, err := f.Stat()
	size := int64(0)
	if err == nil {
		size = fi.Size()
	}

	stats := &model.SessionStats{
		SessionID:  sf.SessionID,
		Project:    ProjectNameFromDir(sf.DirName),
		FilePath:   sf.Path,
		FileSize:   size,
		MaxContext: opts.MaxContext,
	}
	if fi != nil {
		stats.ModTime = fi.ModTime()
	}

	reader := bufio.NewReaderSize(f, 128*1024)
	for {
		line, rerr := reader.ReadBytes('\n')
		if len(line) > 0 {
			processLine(stats, line, tok, opts.IncludeSidechain)
		}
		if rerr != nil {
			break // io.EOF 或读错误，统一结束
		}
	}
	return stats, nil
}

func processLine(stats *model.SessionStats, line []byte, tok tokenizer.Tokenizer, includeSidechain bool) {
	// 去除行尾换行/空白
	for len(line) > 0 {
		c := line[len(line)-1]
		if c == '\n' || c == '\r' || c == ' ' || c == '\t' {
			line = line[:len(line)-1]
		} else {
			break
		}
	}
	if len(line) == 0 {
		return
	}

	var e rawEntry
	if err := json.Unmarshal(line, &e); err != nil {
		stats.ParseErrors++
		return
	}

	// cwd 采集：取最后一条非空，覆盖项目名回退值。
	if e.Cwd != "" {
		stats.Cwd = e.Cwd
		if name := ProjectNameFromCwd(e.Cwd); name != "" {
			stats.Project = name
		}
	}

	// 开始时间：取首条非空 timestamp（文件按时间顺序追加，首条即会话开始）。
	if stats.StartTime.IsZero() && e.Timestamp != "" {
		if t, err := time.Parse(time.RFC3339, e.Timestamp); err == nil {
			stats.StartTime = t
		}
	}

	// sidechain（子 agent）默认排除出主上下文，仅累计参考 Token。
	if e.IsSidechain && !includeSidechain {
		stats.SidechainTokens += extractTokens(&e, tok, stats, false)
		return
	}

	switch e.Type {
	case "user":
		stats.UserMsgCount++
		t := extractTokens(&e, tok, stats, true) + int64(tokenizer.PerMessageOverhead)
		stats.Tokens += t
		addTopMessage(stats, "user", t, previewOf(&e))
	case "assistant":
		stats.AssistantMsgCount++
		t := extractTokens(&e, tok, stats, true) + int64(tokenizer.PerMessageOverhead)
		stats.Tokens += t
		addTopMessage(stats, "assistant", t, previewOf(&e))
		// 记录末条 assistant 的真实 usage（覆盖式，最终即最新一轮的上下文）。
		if e.Message != nil && e.Message.Usage != nil {
			u := e.Message.Usage
			stats.LastUsage = model.UsageInfo{
				HasReal:       true,
				InputTokens:   u.InputTokens,
				CacheRead:     u.CacheReadInputTokens,
				CacheCreation: u.CacheCreationInputTokens,
				OutputTokens:  u.OutputTokens,
				ContextTokens: u.InputTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens,
			}
			// 采样（时间点, 真实上下文），用于增速与预计填满；保留最近 16 个。
			if t, err := time.Parse(time.RFC3339, e.Timestamp); err == nil {
				stats.Samples = append(stats.Samples, model.SamplePoint{At: t, Tokens: stats.LastUsage.ContextTokens})
				if len(stats.Samples) > 16 {
					stats.Samples = stats.Samples[len(stats.Samples)-16:]
				}
			}
		}
	case "attachment":
		stats.AttachmentCount++
		stats.Tokens += extractTokens(&e, tok, stats, true)
	}
}

// extractTokens 计算一个 entry 的内容 Token。
// 当 updCounts 为 true 时，同步累加 tool_use / tool_result 计数。
func extractTokens(e *rawEntry, tok tokenizer.Tokenizer, stats *model.SessionStats, updCounts bool) int64 {
	var total int64
	if e.Message != nil {
		total += rawContentTokens(e.Message.Content, tok, stats, updCounts)
	}
	trimmed := bytes.TrimSpace(e.Attachment)
	if len(trimmed) > 0 && !bytes.Equal(trimmed, []byte("null")) {
		total += int64(tok.Estimate(string(trimmed)))
	}
	return total
}

// rawContentTokens 处理 message.content（可能是 JSON 字符串或数组）。
func rawContentTokens(raw json.RawMessage, tok tokenizer.Tokenizer, stats *model.SessionStats, updCounts bool) int64 {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return 0
	}
	switch raw[0] {
	case '"':
		var s string
		if json.Unmarshal(raw, &s) == nil {
			return int64(tok.Estimate(s))
		}
		return 0
	case '[':
		var blocks []json.RawMessage
		if json.Unmarshal(raw, &blocks) != nil {
			return 0
		}
		var total int64
		for _, b := range blocks {
			total += blockTokens(b, tok, stats, updCounts)
		}
		return total
	}
	return 0
}

func blockTokens(b json.RawMessage, tok tokenizer.Tokenizer, stats *model.SessionStats, updCounts bool) int64 {
	var blk contentBlock
	if json.Unmarshal(b, &blk) != nil {
		return 0
	}
	var total int64
	switch blk.Type {
	case "text":
		total += int64(tok.Estimate(blk.Text))
	case "thinking":
		total += int64(tok.Estimate(blk.Thinking))
	case "tool_use":
		if updCounts {
			stats.ToolUseCount++
		}
		if len(bytes.TrimSpace(blk.Input)) > 0 {
			total += int64(tok.Estimate(string(blk.Input)))
		}
	case "tool_result":
		if updCounts {
			stats.ToolResultCount++
		}
		// tool_result 的 content 可能是字符串，也可能是内容块数组。
		total += rawContentTokens(blk.Content, tok, stats, updCounts)
	}
	return total
}

// addTopMessage 维护按 Token 降序、上限 model.TopN 的最大消息列表（恒定内存）。
func addTopMessage(stats *model.SessionStats, kind string, tokens int64, preview string) {
	if tokens <= 0 {
		return
	}
	ms := model.MessageStat{Kind: kind, Tokens: tokens, Preview: preview}
	idx := sort.Search(len(stats.TopMessages), func(i int) bool {
		return stats.TopMessages[i].Tokens < tokens
	})
	stats.TopMessages = append(stats.TopMessages, model.MessageStat{})
	copy(stats.TopMessages[idx+1:], stats.TopMessages[idx:])
	stats.TopMessages[idx] = ms
	if len(stats.TopMessages) > model.TopN {
		stats.TopMessages = stats.TopMessages[:model.TopN]
	}
}

// previewOf 抽取一条消息的可读预览（首个文本片段），用于定位 Context 暴涨来源。
func previewOf(e *rawEntry) string {
	var s string
	if e.Message != nil {
		s = firstText(e.Message.Content)
	}
	if s == "" {
		trimmed := bytes.TrimSpace(e.Attachment)
		if len(trimmed) > 0 && !bytes.Equal(trimmed, []byte("null")) {
			s = string(trimmed)
		}
	}
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\t", " ")
	if r := []rune(s); len(r) > 60 {
		s = string(r[:60]) + "..."
	}
	return s
}

func firstText(raw json.RawMessage) string {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return ""
	}
	switch raw[0] {
	case '"':
		var s string
		_ = json.Unmarshal(raw, &s)
		return s
	case '[':
		var blocks []json.RawMessage
		if json.Unmarshal(raw, &blocks) != nil {
			return ""
		}
		for _, b := range blocks {
			var blk contentBlock
			if json.Unmarshal(b, &blk) != nil {
				continue
			}
			switch blk.Type {
			case "text":
				if blk.Text != "" {
					return blk.Text
				}
			case "thinking":
				if blk.Thinking != "" {
					return blk.Thinking
				}
			case "tool_use":
				if blk.Name != "" {
					// 工具调用：返回 "工具名 + 输入摘要"，便于定位暴涨来源
					snippet := strings.TrimSpace(string(blk.Input))
					return blk.Name + " " + snippet
				}
			case "tool_result":
				if t := firstText(blk.Content); t != "" {
					return t
				}
			}
		}
	}
	return ""
}

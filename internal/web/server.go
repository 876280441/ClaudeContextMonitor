// Package web 提供 ClaudeContextMonitor 的 Web 仪表盘与 REST API。
//
// 仅使用标准库：net/http 起本地服务，dashboard.html 通过 go:embed 打包进 exe。
// server 端预算 used%/level/status，前端只负责配色与展示。
package web

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/USER/claude-context-monitor/internal/claude"
	"github.com/USER/claude-context-monitor/internal/model"
	"github.com/USER/claude-context-monitor/internal/report"
	"github.com/USER/claude-context-monitor/internal/scanner"
	"github.com/USER/claude-context-monitor/internal/tokenizer"
	"github.com/USER/claude-context-monitor/internal/ui"
)

//go:embed dashboard.html
var dashboardHTML []byte

// Server 是 Web 仪表盘服务。
type Server struct {
	claudeDir        string
	maxContext       int64
	includeSidechain bool

	mu    sync.Mutex
	cache *cacheEntry
}

type cacheKey struct {
	max int64
	isc bool
}

type cacheEntry struct {
	key    cacheKey
	at     time.Time
	result *scanner.Result
}

// NewServer 创建一个 Web 服务。claudeDir/maxContext/includeSidechain 为默认值，
// 可被请求的 query 参数临时覆盖。
func NewServer(claudeDir string, maxContext int64, includeSidechain bool) *Server {
	return &Server{
		claudeDir:        claudeDir,
		maxContext:       maxContext,
		includeSidechain: includeSidechain,
	}
}

// cacheTTL 控制扫描结果缓存有效期，避免一次页面刷新内多请求重复全量扫描。
const cacheTTL = time.Second

// scan 返回（带 1s 缓存的）扫描结果。
func (s *Server) scan(maxContext int64, includeSidechain bool) *scanner.Result {
	key := cacheKey{max: maxContext, isc: includeSidechain}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if s.cache != nil && s.cache.key == key && now.Sub(s.cache.at) < cacheTTL {
		return s.cache.result
	}
	files, err := claude.Discover(s.claudeDir)
	if err != nil {
		return &scanner.Result{}
	}
	tok := tokenizer.NewEstimate()
	res := scanner.Scan(files, tok, scanner.Options{
		MaxContext:       maxContext,
		IncludeSidechain: includeSidechain,
	})
	s.cache = &cacheEntry{key: key, at: now, result: res}
	return res
}

// params 解析请求中的 max_context / include_sidechain 覆盖值，缺省用 server 默认。
func (s *Server) params(r *http.Request) (int64, bool) {
	q := r.URL.Query()
	mc := s.maxContext
	if v := q.Get("max_context"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			mc = n
		}
	}
	isc := s.includeSidechain
	if v := q.Get("include_sidechain"); v != "" {
		isc = v == "1" || v == "true"
	}
	return mc, isc
}

// Handler 返回路由后的 http.Handler。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleDashboard)
	mux.HandleFunc("/api/overview", s.handleOverview)
	mux.HandleFunc("/api/sessions", s.handleSessions)
	mux.HandleFunc("/api/projects", s.handleProjects)
	mux.HandleFunc("/api/session/", s.handleSessionDetail) // 注意末尾斜杠：前缀匹配
	return mux
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// 将服务端默认 max-context 注入页面输入框，避免与 --max-context flag 不一致。
	html := bytes.Replace(dashboardHTML, []byte("__MAX_CTX__"),
		[]byte(strconv.FormatInt(s.maxContext, 10)), 1)
	w.Write(html)
}

// ---- DTO ----

type overviewDTO struct {
	TotalSessions int    `json:"total_sessions"`
	TotalTokens   int64  `json:"total_tokens"`
	TotalTokensH  string `json:"total_tokens_h"`
	Over90        int    `json:"over_90"`
	Over95        int    `json:"over_95"`
	MaxContext    int64  `json:"max_context"`
	ParseErrors   int    `json:"parse_errors"`
	GeneratedAt   string `json:"generated_at"`
}

type messageDTO struct {
	Kind    string `json:"kind"`
	Tokens  int64  `json:"tokens"`
	Preview string `json:"preview"`
}

type sessionDTO struct {
	Project         string       `json:"project"`
	SessionID       string       `json:"session_id"`
	ShortID         string       `json:"short_id"`
	Cwd             string       `json:"cwd"`
	FileSize        int64        `json:"file_size"`
	FileSizeH       string       `json:"file_size_h"`
	Tokens          int64        `json:"tokens"`
	UsedPct         float64      `json:"used_pct"`
	Remaining       int64        `json:"remaining"`
	Status          string       `json:"status"`
	Level           string       `json:"level"`
	Messages        int          `json:"messages"`
	UserMsg         int          `json:"user"`
	AssistantMsg    int          `json:"assistant"`
	Attachments     int          `json:"attachments"`
	ToolUse         int          `json:"tool_use"`
	ToolResult      int          `json:"tool_result"`
	ModTime         string       `json:"mod_time"`
	TopMessages     []messageDTO `json:"top_messages,omitempty"`
	SidechainTokens int64        `json:"sidechain_tokens,omitempty"`
}

type projectDTO struct {
	Name           string `json:"name"`
	SessionCount   int    `json:"session_count"`
	TotalTokens    int64  `json:"total_tokens"`
	TotalTokensH   string `json:"total_tokens_h"`
	LargestSession string `json:"largest_session"`
	LargestTokens  int64  `json:"largest_tokens"`
	LargestTokensH string `json:"largest_tokens_h"`
}

// ---- handlers ----

func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	mc, isc := s.params(r)
	res := s.scan(mc, isc)
	var over90, over95 int
	for _, st := range res.Sessions {
		u := usedPct(st.Tokens, mc)
		lvl := ui.LevelFor(u)
		if lvl >= ui.LevelOrange {
			over90++
		}
		if lvl >= ui.LevelRed {
			over95++
		}
	}
	writeJSON(w, overviewDTO{
		TotalSessions: len(res.Sessions),
		TotalTokens:   report.TotalTokens(res.Sessions),
		TotalTokensH:  ui.FormatTokens(report.TotalTokens(res.Sessions)),
		Over90:        over90,
		Over95:        over95,
		MaxContext:    mc,
		ParseErrors:   res.ParseErrors,
		GeneratedAt:   time.Now().Format(time.RFC3339),
	})
}

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	mc, isc := s.params(r)
	res := s.scan(mc, isc)
	sorted := report.SortSessionsByTokens(res.Sessions)
	limit := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	out := sorted
	if limit > 0 && limit < len(sorted) {
		out = sorted[:limit]
	}
	dtos := make([]sessionDTO, 0, len(out))
	for _, st := range out {
		dtos = append(dtos, toSessionDTO(st, mc))
	}
	writeJSON(w, dtos)
}

func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	mc, isc := s.params(r)
	res := s.scan(mc, isc)
	projects := report.SortProjectsByTokens(report.AggregateProjects(res.Sessions))
	dtos := make([]projectDTO, 0, len(projects))
	for _, p := range projects {
		dtos = append(dtos, projectDTO{
			Name:           p.Name,
			SessionCount:   p.SessionCount,
			TotalTokens:    p.TotalTokens,
			TotalTokensH:   ui.FormatTokens(p.TotalTokens),
			LargestSession: p.LargestSession,
			LargestTokens:  p.LargestTokens,
			LargestTokensH: ui.FormatTokens(p.LargestTokens),
		})
	}
	writeJSON(w, dtos)
}

func (s *Server) handleSessionDetail(w http.ResponseWriter, r *http.Request) {
	mc, isc := s.params(r)
	id := strings.TrimPrefix(r.URL.Path, "/api/session/")
	if id == "" {
		http.Error(w, "missing session id", http.StatusBadRequest)
		return
	}
	res := s.scan(mc, isc)
	st, _ := report.FindSession(res.Sessions, id)
	if st == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	dto := toSessionDTO(st, mc)
	dto.TopMessages = make([]messageDTO, 0, len(st.TopMessages))
	for _, m := range st.TopMessages {
		dto.TopMessages = append(dto.TopMessages, messageDTO{Kind: m.Kind, Tokens: m.Tokens, Preview: m.Preview})
	}
	dto.SidechainTokens = st.SidechainTokens
	writeJSON(w, dto)
}

// ---- helpers ----

func usedPct(tokens, maxContext int64) float64 {
	if maxContext <= 0 {
		return 0
	}
	return float64(tokens) / float64(maxContext) * 100
}

func toSessionDTO(st *model.SessionStats, maxContext int64) sessionDTO {
	u := usedPct(st.Tokens, maxContext)
	lvl := ui.LevelFor(u)
	status, _ := ui.StatusLabel(u)
	mt := ""
	if !st.ModTime.IsZero() {
		mt = st.ModTime.Format(time.RFC3339)
	}
	remaining := maxContext - st.Tokens
	if remaining < 0 {
		remaining = 0
	}
	return sessionDTO{
		Project:      st.Project,
		SessionID:    st.SessionID,
		ShortID:      ui.ShortID(st.SessionID, 8),
		Cwd:          st.Cwd,
		FileSize:     st.FileSize,
		FileSizeH:    ui.FormatSize(st.FileSize),
		Tokens:       st.Tokens,
		UsedPct:      u,
		Remaining:    remaining,
		Status:       status,
		Level:        levelStr(lvl),
		Messages:     st.TotalMessages(),
		UserMsg:      st.UserMsgCount,
		AssistantMsg: st.AssistantMsgCount,
		Attachments:  st.AttachmentCount,
		ToolUse:      st.ToolUseCount,
		ToolResult:   st.ToolResultCount,
		ModTime:      mt,
	}
}

func levelStr(l ui.Level) string {
	switch l {
	case ui.LevelGreen:
		return "green"
	case ui.LevelYellow:
		return "yellow"
	case ui.LevelOrange:
		return "orange"
	case ui.LevelRed:
		return "red"
	case ui.LevelBurst:
		return "burst"
	}
	return "green"
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "encode error", http.StatusInternalServerError)
	}
}

// OpenBrowser 尝试用系统默认浏览器打开 URL（失败仅返回错误，不中断）。
func OpenBrowser(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("cmd", "/c", "start", "", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}

// ListenAddr 规范化监听地址：纯端口 "8080" → "127.0.0.1:8080"；空 → 默认 8765。
func ListenAddr(arg string) string {
	a := strings.TrimSpace(arg)
	if a == "" {
		return "127.0.0.1:8765"
	}
	if _, err := strconv.Atoi(a); err == nil {
		return "127.0.0.1:" + a
	}
	if !strings.Contains(a, ":") {
		a = "127.0.0.1:" + a
	}
	return a
}

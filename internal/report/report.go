// Package report 负责 Session 排名与 Project 聚合。
package report

import (
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/USER/claude-context-monitor/internal/model"
)

// SortSessionsByTokens 按 Token 降序排序（返回新切片，不改原切片）。
// Token 取真实 usage（无则估算）。
func SortSessionsByTokens(sessions []*model.SessionStats) []*model.SessionStats {
	out := make([]*model.SessionStats, len(sessions))
	copy(out, sessions)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].ContextTokens() > out[j].ContextTokens()
	})
	return out
}

// Top 返回 Token 最大的前 n 个 Session（降序）。n<=0 时返回全部。
func Top(sessions []*model.SessionStats, n int) []*model.SessionStats {
	sorted := SortSessionsByTokens(sessions)
	if n <= 0 || n >= len(sorted) {
		return sorted
	}
	return sorted[:n]
}

// ProjectKeyOf 返回 Session 所属项目的稳定分组键（目录级，不区分大小写）。
// 同一目录下的多个 Session 返回相同键；不同目录即使末段同名也返回不同键。
func ProjectKeyOf(s *model.SessionStats) string {
	if s.Cwd != "" {
		return normKey(s.Cwd)
	}
	if s.FilePath != "" {
		return normKey(filepath.Dir(s.FilePath))
	}
	return s.Project // 最终回退（极少触发）
}

func normKey(p string) string {
	return strings.ToLower(filepath.ToSlash(p))
}

// displayPathOf 返回用于生成显示名的可读路径（优先 cwd，回退到文件所在目录）。
func displayPathOf(s *model.SessionStats) string {
	if s.Cwd != "" {
		return s.Cwd
	}
	if s.FilePath != "" {
		return filepath.Dir(s.FilePath)
	}
	return s.Project
}

func splitSegments(p string) []string {
	p = strings.ReplaceAll(p, "\\", "/")
	raw := strings.Split(p, "/")
	segs := make([]string, 0, len(raw))
	for _, r := range raw {
		r = strings.TrimSpace(r)
		if r != "" && r != "." {
			segs = append(segs, r)
		}
	}
	return segs
}

// ComputeProjectDisplayNames 返回 "项目键 -> 简短显示名" 的映射。
// 末段（叶子）唯一时直接用末段；发生同名碰撞时自动追加父级目录，
// 直到能区分为止（如 php/lwjk_app 与 lwjk_v8/lwjk_app）。结果大写。
func ComputeProjectDisplayNames(sessions []*model.SessionStats) map[string]string {
	repPath := map[string]string{}
	keys := []string{}
	for _, s := range sessions {
		k := ProjectKeyOf(s)
		if _, ok := repPath[k]; !ok {
			repPath[k] = displayPathOf(s)
			keys = append(keys, k)
		}
	}
	segs := map[string][]string{}
	maxLen := 1
	for _, k := range keys {
		s := splitSegments(repPath[k])
		segs[k] = s
		if len(s) > maxLen {
			maxLen = len(s)
		}
	}
	suffix := func(k string, n int) string {
		s := segs[k]
		if n > len(s) {
			n = len(s)
		}
		return strings.Join(s[len(s)-n:], "/")
	}
	out := make(map[string]string, len(keys))
	for _, k := range keys {
		n := 1
		for n < maxLen {
			suf := suffix(k, n)
			collide := false
			for _, k2 := range keys {
				if k2 != k && suffix(k2, n) == suf {
					collide = true
					break
				}
			}
			if !collide {
				break
			}
			n++
		}
		out[k] = strings.ToUpper(suffix(k, n))
	}
	return out
}

// DisplayName 返回某 Session 的项目显示名（需先由 ComputeProjectDisplayNames 计算映射）。
func DisplayName(names map[string]string, s *model.SessionStats) string {
	if d, ok := names[ProjectKeyOf(s)]; ok && d != "" {
		return d
	}
	return s.Project
}

// AggregateProjects 按"项目目录"聚合统计：会话数、总 Token、最大 Session。
// 同名但不同目录的项目会被正确拆分为独立项目，显示名经同名消歧处理。
func AggregateProjects(sessions []*model.SessionStats) []*model.ProjectStats {
	type agg struct {
		p   *model.ProjectStats
		rep *model.SessionStats
	}
	groups := map[string]*agg{}
	order := []string{}
	for _, s := range sessions {
		k := ProjectKeyOf(s)
		a, ok := groups[k]
		if !ok {
			a = &agg{p: &model.ProjectStats{}, rep: s}
			groups[k] = a
			order = append(order, k)
		}
		a.p.SessionCount++
		a.p.TotalTokens += s.ContextTokens()
		if s.ContextTokens() > a.p.LargestTokens {
			a.p.LargestTokens = s.ContextTokens()
			a.p.LargestSession = s.SessionID
		}
	}
	names := ComputeProjectDisplayNames(sessions)
	out := make([]*model.ProjectStats, 0, len(order))
	for _, k := range order {
		p := groups[k].p
		p.Name = names[k]
		if p.Name == "" {
			p.Name = groups[k].rep.Project
			if p.Name == "" {
				p.Name = "(unknown)"
			}
		}
		out = append(out, p)
	}
	return out
}

// SortProjectsByTokens 按 TotalTokens 降序排序项目。
func SortProjectsByTokens(projects []*model.ProjectStats) []*model.ProjectStats {
	out := make([]*model.ProjectStats, len(projects))
	copy(out, projects)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].TotalTokens > out[j].TotalTokens
	})
	return out
}

// FindSession 按完整 ID 或前缀模糊匹配 Session。返回匹配结果与是否精确匹配。
// 若有多个前缀匹配，返回 Token 最大的那个。
func FindSession(sessions []*model.SessionStats, query string) (*model.SessionStats, bool) {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil, false
	}
	var prefixMatch *model.SessionStats
	for _, s := range sessions {
		id := strings.ToLower(s.SessionID)
		if id == q {
			return s, true // 精确匹配优先
		}
		if strings.HasPrefix(id, q) {
			if prefixMatch == nil || s.ContextTokens() > prefixMatch.ContextTokens() {
				prefixMatch = s
			}
		}
	}
	return prefixMatch, false
}

// TotalTokens 汇总所有 Session 的 Token（真实优先）。
func TotalTokens(sessions []*model.SessionStats) int64 {
	var sum int64
	for _, s := range sessions {
		sum += s.ContextTokens()
	}
	return sum
}

// GlobalTopMessages 聚合所有会话的 Top 消息，返回全局最大的前 n 条（带来源）。
// 依赖每个会话已捕获的 TopMessages；n<=0 时默认 20。
func GlobalTopMessages(sessions []*model.SessionStats, n int) []*model.GlobalMessageStat {
	if n <= 0 {
		n = 20
	}
	names := ComputeProjectDisplayNames(sessions)
	var all []*model.GlobalMessageStat
	for _, s := range sessions {
		name := DisplayName(names, s)
		for _, m := range s.TopMessages {
			all = append(all, &model.GlobalMessageStat{
				Project:   name,
				SessionID: s.SessionID,
				Kind:      m.Kind,
				Tokens:    m.Tokens,
				Preview:   m.Preview,
			})
		}
	}
	sort.SliceStable(all, func(i, j int) bool {
		return all[i].Tokens > all[j].Tokens
	})
	if n < len(all) {
		all = all[:n]
	}
	return all
}

// EstimateFill 依据近期真实 usage 采样估算上下文增速与预计填满时间。
// 返回 (增速 token/秒, 距填满时长, 是否可估算)。ok=false 表示样本不足或无增长。
func EstimateFill(s *model.SessionStats, maxContext int64) (ratePerSec float64, eta time.Duration, ok bool) {
	n := len(s.Samples)
	if n < 2 || maxContext <= 0 {
		return 0, 0, false
	}
	start := 0
	if n > 8 { // 取最近 8 个采样反映"当前"增速
		start = n - 8
	}
	a := s.Samples[start]
	b := s.Samples[n-1]
	dt := b.At.Sub(a.At).Seconds()
	if dt <= 0 {
		return 0, 0, false
	}
	rate := float64(b.Tokens-a.Tokens) / dt
	if rate <= 0 { // 无增长或正在缩减（如压缩）
		return rate, 0, false
	}
	remaining := float64(maxContext - b.Tokens)
	if remaining <= 0 {
		return rate, 0, false // 已满
	}
	return rate, time.Duration(remaining / rate * float64(time.Second)), true
}

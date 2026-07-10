// Package report 负责 Session 排名与 Project 聚合。
package report

import (
	"sort"
	"strings"

	"github.com/USER/claude-context-monitor/internal/model"
)

// SortSessionsByTokens 按 Token 降序排序（返回新切片，不改原切片）。
func SortSessionsByTokens(sessions []*model.SessionStats) []*model.SessionStats {
	out := make([]*model.SessionStats, len(sessions))
	copy(out, sessions)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Tokens > out[j].Tokens
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

// AggregateProjects 按项目聚合统计：会话数、总 Token、最大 Session。
func AggregateProjects(sessions []*model.SessionStats) []*model.ProjectStats {
	byName := map[string]*model.ProjectStats{}
	var order []string
	for _, s := range sessions {
		name := s.Project
		if name == "" {
			name = "(unknown)"
		}
		p, ok := byName[name]
		if !ok {
			p = &model.ProjectStats{Name: name}
			byName[name] = p
			order = append(order, name)
		}
		p.SessionCount++
		p.TotalTokens += s.Tokens
		if s.Tokens > p.LargestTokens {
			p.LargestTokens = s.Tokens
			p.LargestSession = s.SessionID
		}
	}
	out := make([]*model.ProjectStats, 0, len(order))
	for _, name := range order {
		out = append(out, byName[name])
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
			if prefixMatch == nil || s.Tokens > prefixMatch.Tokens {
				prefixMatch = s
			}
		}
	}
	return prefixMatch, false
}

// TotalTokens 汇总所有 Session 的 Token。
func TotalTokens(sessions []*model.SessionStats) int64 {
	var sum int64
	for _, s := range sessions {
		sum += s.Tokens
	}
	return sum
}

// Package scanner 编排对所有 Session 的并发扫描。
package scanner

import (
	"runtime"
	"sync"

	"github.com/USER/claude-context-monitor/internal/claude"
	"github.com/USER/claude-context-monitor/internal/model"
	"github.com/USER/claude-context-monitor/internal/tokenizer"
)

// Options 是扫描配置。
type Options struct {
	MaxContext       int64
	IncludeSidechain bool
	Workers          int // 并发度；<=0 时自动取 min(16, NumCPU)
}

// Result 是一次扫描的结果汇总。
type Result struct {
	Sessions    []*model.SessionStats
	ParseErrors int // 全部文件的解析失败行总数
}

// Scan 并发解析所有 SessionFile，返回按原（无序）顺序的结果。
// 每个 Session 独立处理，单 Session 超大不会影响其他。
func Scan(files []claude.SessionFile, tok tokenizer.Tokenizer, opts Options) *Result {
	workers := opts.Workers
	if workers <= 0 {
		workers = runtime.NumCPU()
		if workers > 16 {
			workers = 16
		}
		if workers < 1 {
			workers = 1
		}
	}
	if workers > len(files) {
		workers = len(files)
	}
	if workers < 1 {
		workers = 1
	}

	results := make([]*model.SessionStats, len(files))
	parseErrs := make([]int, len(files))

	// 分块调度：每个 worker 处理一段索引区间，比通道派发更省调度开销。
	var wg sync.WaitGroup
	chunk := (len(files) + workers - 1) / workers
	for w := 0; w < workers; w++ {
		start := w * chunk
		end := start + chunk
		if end > len(files) {
			end = len(files)
		}
		if start >= end {
			continue
		}
		wg.Add(1)
		go func(s, e int) {
			defer wg.Done()
			po := claude.ParseOptions{
				MaxContext:       opts.MaxContext,
				IncludeSidechain: opts.IncludeSidechain,
			}
			for i := s; i < e; i++ {
				st, err := claude.ParseFile(files[i], tok, po)
				if err != nil {
					continue // 容错：单文件失败不中断
				}
				results[i] = st
				parseErrs[i] = st.ParseErrors
			}
		}(start, end)
	}
	wg.Wait()

	// 过滤掉 nil（打开失败的文件），累加解析错误。
	out := &Result{Sessions: make([]*model.SessionStats, 0, len(results))}
	for i, st := range results {
		if st == nil {
			continue
		}
		out.Sessions = append(out.Sessions, st)
		out.ParseErrors += parseErrs[i]
	}
	return out
}

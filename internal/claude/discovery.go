// Package claude 负责发现与解析本地 Claude Code 的 Session 数据。
package claude

import (
	"os"
	"path/filepath"
	"strings"
)

// SessionFile 描述一个待解析的 Session 文件。
type SessionFile struct {
	Path      string // 完整路径
	SessionID string // 文件名（不含扩展名）
	DirName   string // 所属项目目录名（项目名回退用）
}

// ProjectsDir 返回 .claude\projects 目录路径。
func ProjectsDir(claudeDir string) string {
	return filepath.Join(claudeDir, "projects")
}

// DefaultClaudeDir 返回当前用户的默认 .claude 目录（%USERPROFILE%\.claude）。
func DefaultClaudeDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		// Windows 回退：%USERPROFILE%
		if h := os.Getenv("USERPROFILE"); h != "" {
			return filepath.Join(h, ".claude")
		}
		return filepath.Join(os.Getenv("HOMEDRIVE")+os.Getenv("HOMEPATH"), ".claude")
	}
	return filepath.Join(home, ".claude")
}

// Discover 扫描 projects 目录，返回所有 *.jsonl 文件清单。
// 容错：无法访问的子目录会被跳过，不中断整体扫描。
func Discover(claudeDir string) ([]SessionFile, error) {
	root := ProjectsDir(claudeDir)
	var files []SessionFile
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, err
	}

	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 容错跳过
		}
		if info.IsDir() {
			return nil
		}
		name := info.Name()
		if !strings.EqualFold(filepath.Ext(name), ".jsonl") {
			return nil
		}
		dir := filepath.Base(filepath.Dir(path))
		sid := strings.TrimSuffix(name, filepath.Ext(name))
		files = append(files, SessionFile{
			Path:      path,
			SessionID: sid,
			DirName:   dir,
		})
		return nil
	})
	if err != nil {
		return files, err
	}
	return files, nil
}

// ProjectNameFromCwd 从 cwd 路径提取项目显示名（末段，大写）。
// 例如 "/home/user/projects/myapp" → "MYAPP"。
func ProjectNameFromCwd(cwd string) string {
	cwd = strings.TrimRight(cwd, `/\`)
	if cwd == "" {
		return ""
	}
	seg := cwd
	if idx := strings.LastIndexAny(cwd, `\/`); idx >= 0 {
		seg = cwd[idx+1:]
	}
	seg = strings.TrimSpace(seg)
	if seg == "" {
		return ""
	}
	return strings.ToUpper(seg)
}

// ProjectNameFromDir 从项目目录名回退提取项目显示名（最后一个非空段，大写）。
// 例如 "X--home-user-projects-myapp" → "MYAPP"。
func ProjectNameFromDir(dir string) string {
	parts := strings.Split(dir, "-")
	for i := len(parts) - 1; i >= 0; i-- {
		if p := strings.TrimSpace(parts[i]); p != "" {
			return strings.ToUpper(p)
		}
	}
	return strings.ToUpper(dir)
}

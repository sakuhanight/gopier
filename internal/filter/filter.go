package filter

import (
	"path/filepath"
	"strings"
)

// Filter はファイルフィルタリングを行う構造体
type Filter struct {
	includePatterns []string
	excludePatterns []string
}

// NewFilter は新しいフィルタを作成する
func NewFilter(includePattern, excludePattern string) *Filter {
	var includePatterns []string
	var excludePatterns []string

	// 含めるパターンの解析
	if includePattern != "" {
		includePatterns = strings.Split(includePattern, ",")
		for i, p := range includePatterns {
			includePatterns[i] = strings.TrimSpace(p)
		}
	}

	// 除外パターンの解析
	if excludePattern != "" {
		excludePatterns = strings.Split(excludePattern, ",")
		for i, p := range excludePatterns {
			excludePatterns[i] = strings.TrimSpace(p)
		}
	}

	return &Filter{
		includePatterns: includePatterns,
		excludePatterns: excludePatterns,
	}
}

// ShouldInclude はファイルを含めるべきかどうかを判断する
func (f *Filter) ShouldInclude(path string) bool {
	// 除外パターンのチェック
	for _, pattern := range f.excludePatterns {
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err == nil && matched {
			return false
		}
	}

	// 含めるパターンが指定されていない場合は全て含める
	if len(f.includePatterns) == 0 {
		return true
	}

	// 含めるパターンのチェック
	for _, pattern := range f.includePatterns {
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err == nil && matched {
			return true
		}
	}

	return false
}

// IsExcluded はファイルが除外パターンに一致するかどうかを判断する
func (f *Filter) IsExcluded(path string) bool {
	// 除外パターンのチェック
	for _, pattern := range f.excludePatterns {
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err == nil && matched {
			return true
		}
	}
	return false
}

// IsIncluded はファイルが含めるパターンに一致するかどうかを判断する
func (f *Filter) IsIncluded(path string) bool {
	// 含めるパターンが指定されていない場合は全て含める
	if len(f.includePatterns) == 0 {
		return true
	}

	// 含めるパターンのチェック
	for _, pattern := range f.includePatterns {
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err == nil && matched {
			return true
		}
	}
	return false
}

// GetIncludePatterns は含めるパターンのリストを取得する
func (f *Filter) GetIncludePatterns() []string {
	return f.includePatterns
}

// GetExcludePatterns は除外パターンのリストを取得する
func (f *Filter) GetExcludePatterns() []string {
	return f.excludePatterns
}

// HasPatterns はフィルタにパターンが設定されているかどうかを判断する
func (f *Filter) HasPatterns() bool {
	return len(f.includePatterns) > 0 || len(f.excludePatterns) > 0
}

// MatchesPath はパスがパターンに一致するかどうかを判断する
// パターンはカンマ区切りの複数のパターンを含む文字列
func MatchesPath(path, patterns string) bool {
	if patterns == "" {
		return false
	}

	patternList := strings.Split(patterns, ",")
	for _, pattern := range patternList {
		pattern = strings.TrimSpace(pattern)
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err == nil && matched {
			return true
		}
	}

	return false
}

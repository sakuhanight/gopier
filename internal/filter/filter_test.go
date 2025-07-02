package filter

import (
	"testing"
)

func TestNewFilter(t *testing.T) {
	tests := []struct {
		name            string
		includePattern  string
		excludePattern  string
		expectedInclude []string
		expectedExclude []string
	}{
		{
			name:            "空のパターン",
			includePattern:  "",
			excludePattern:  "",
			expectedInclude: []string{},
			expectedExclude: []string{},
		},
		{
			name:            "単一の含めるパターン",
			includePattern:  "*.txt",
			excludePattern:  "",
			expectedInclude: []string{"*.txt"},
			expectedExclude: []string{},
		},
		{
			name:            "複数の含めるパターン",
			includePattern:  "*.txt,*.docx",
			excludePattern:  "",
			expectedInclude: []string{"*.txt", "*.docx"},
			expectedExclude: []string{},
		},
		{
			name:            "単一の除外パターン",
			includePattern:  "",
			excludePattern:  "*.tmp",
			expectedInclude: []string{},
			expectedExclude: []string{"*.tmp"},
		},
		{
			name:            "複数の除外パターン",
			includePattern:  "",
			excludePattern:  "*.tmp,*.bak",
			expectedInclude: []string{},
			expectedExclude: []string{"*.tmp", "*.bak"},
		},
		{
			name:            "含めるパターンと除外パターンの両方",
			includePattern:  "*.txt,*.docx",
			excludePattern:  "*.tmp,*.bak",
			expectedInclude: []string{"*.txt", "*.docx"},
			expectedExclude: []string{"*.tmp", "*.bak"},
		},
		{
			name:            "空白を含むパターン",
			includePattern:  " *.txt , *.docx ",
			excludePattern:  " *.tmp , *.bak ",
			expectedInclude: []string{"*.txt", "*.docx"},
			expectedExclude: []string{"*.tmp", "*.bak"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewFilter(tt.includePattern, tt.excludePattern)

			if len(filter.includePatterns) != len(tt.expectedInclude) {
				t.Errorf("含めるパターンの数が一致しません: 期待値=%d, 実際=%d", len(tt.expectedInclude), len(filter.includePatterns))
			}

			if len(filter.excludePatterns) != len(tt.expectedExclude) {
				t.Errorf("除外パターンの数が一致しません: 期待値=%d, 実際=%d", len(tt.expectedExclude), len(filter.excludePatterns))
			}

			for i, pattern := range tt.expectedInclude {
				if filter.includePatterns[i] != pattern {
					t.Errorf("含めるパターン[%d]が一致しません: 期待値=%s, 実際=%s", i, pattern, filter.includePatterns[i])
				}
			}

			for i, pattern := range tt.expectedExclude {
				if filter.excludePatterns[i] != pattern {
					t.Errorf("除外パターン[%d]が一致しません: 期待値=%s, 実際=%s", i, pattern, filter.excludePatterns[i])
				}
			}
		})
	}
}

func TestShouldInclude(t *testing.T) {
	tests := []struct {
		name            string
		includePattern  string
		excludePattern  string
		filePath        string
		expectedInclude bool
	}{
		{
			name:            "パターンなし - 含める",
			includePattern:  "",
			excludePattern:  "",
			filePath:        "test.txt",
			expectedInclude: true,
		},
		{
			name:            "含めるパターンに一致",
			includePattern:  "*.txt",
			excludePattern:  "",
			filePath:        "test.txt",
			expectedInclude: true,
		},
		{
			name:            "含めるパターンに一致しない",
			includePattern:  "*.txt",
			excludePattern:  "",
			filePath:        "test.doc",
			expectedInclude: false,
		},
		{
			name:            "除外パターンに一致",
			includePattern:  "",
			excludePattern:  "*.tmp",
			filePath:        "test.tmp",
			expectedInclude: false,
		},
		{
			name:            "除外パターンに一致しない",
			includePattern:  "",
			excludePattern:  "*.tmp",
			filePath:        "test.txt",
			expectedInclude: true,
		},
		{
			name:            "含めるパターンと除外パターンの両方に一致",
			includePattern:  "*.txt",
			excludePattern:  "*.tmp",
			filePath:        "test.txt",
			expectedInclude: true,
		},
		{
			name:            "除外パターンが優先される",
			includePattern:  "*.txt",
			excludePattern:  "test.txt",
			filePath:        "test.txt",
			expectedInclude: false,
		},
		{
			name:            "複数の含めるパターン",
			includePattern:  "*.txt,*.doc",
			excludePattern:  "",
			filePath:        "test.doc",
			expectedInclude: true,
		},
		{
			name:            "複数の除外パターン",
			includePattern:  "",
			excludePattern:  "*.tmp,*.bak",
			filePath:        "test.bak",
			expectedInclude: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewFilter(tt.includePattern, tt.excludePattern)
			result := filter.ShouldInclude(tt.filePath)

			if result != tt.expectedInclude {
				t.Errorf("ShouldInclude() = %v, 期待値 %v", result, tt.expectedInclude)
			}
		})
	}
}

func TestIsExcluded(t *testing.T) {
	tests := []struct {
		name           string
		excludePattern string
		filePath       string
		expectedResult bool
	}{
		{
			name:           "除外パターンなし",
			excludePattern: "",
			filePath:       "test.txt",
			expectedResult: false,
		},
		{
			name:           "除外パターンに一致",
			excludePattern: "*.tmp",
			filePath:       "test.tmp",
			expectedResult: true,
		},
		{
			name:           "除外パターンに一致しない",
			excludePattern: "*.tmp",
			filePath:       "test.txt",
			expectedResult: false,
		},
		{
			name:           "複数の除外パターン",
			excludePattern: "*.tmp,*.bak",
			filePath:       "test.bak",
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewFilter("", tt.excludePattern)
			result := filter.IsExcluded(tt.filePath)

			if result != tt.expectedResult {
				t.Errorf("IsExcluded() = %v, 期待値 %v", result, tt.expectedResult)
			}
		})
	}
}

func TestIsIncluded(t *testing.T) {
	tests := []struct {
		name           string
		includePattern string
		filePath       string
		expectedResult bool
	}{
		{
			name:           "含めるパターンなし",
			includePattern: "",
			filePath:       "test.txt",
			expectedResult: true,
		},
		{
			name:           "含めるパターンに一致",
			includePattern: "*.txt",
			filePath:       "test.txt",
			expectedResult: true,
		},
		{
			name:           "含めるパターンに一致しない",
			includePattern: "*.txt",
			filePath:       "test.doc",
			expectedResult: false,
		},
		{
			name:           "複数の含めるパターン",
			includePattern: "*.txt,*.doc",
			filePath:       "test.doc",
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewFilter(tt.includePattern, "")
			result := filter.IsIncluded(tt.filePath)

			if result != tt.expectedResult {
				t.Errorf("IsIncluded() = %v, 期待値 %v", result, tt.expectedResult)
			}
		})
	}
}

func TestGetIncludePatterns(t *testing.T) {
	filter := NewFilter("*.txt,*.doc", "*.tmp")
	patterns := filter.GetIncludePatterns()

	expected := []string{"*.txt", "*.doc"}
	if len(patterns) != len(expected) {
		t.Errorf("含めるパターンの数が一致しません: 期待値=%d, 実際=%d", len(expected), len(patterns))
	}

	for i, pattern := range expected {
		if patterns[i] != pattern {
			t.Errorf("含めるパターン[%d]が一致しません: 期待値=%s, 実際=%s", i, pattern, patterns[i])
		}
	}
}

func TestGetExcludePatterns(t *testing.T) {
	filter := NewFilter("*.txt", "*.tmp,*.bak")
	patterns := filter.GetExcludePatterns()

	expected := []string{"*.tmp", "*.bak"}
	if len(patterns) != len(expected) {
		t.Errorf("除外パターンの数が一致しません: 期待値=%d, 実際=%d", len(expected), len(patterns))
	}

	for i, pattern := range expected {
		if patterns[i] != pattern {
			t.Errorf("除外パターン[%d]が一致しません: 期待値=%s, 実際=%s", i, pattern, patterns[i])
		}
	}
}

func TestHasPatterns(t *testing.T) {
	tests := []struct {
		name           string
		includePattern string
		excludePattern string
		expectedResult bool
	}{
		{
			name:           "パターンなし",
			includePattern: "",
			excludePattern: "",
			expectedResult: false,
		},
		{
			name:           "含めるパターンのみ",
			includePattern: "*.txt",
			excludePattern: "",
			expectedResult: true,
		},
		{
			name:           "除外パターンのみ",
			includePattern: "",
			excludePattern: "*.tmp",
			expectedResult: true,
		},
		{
			name:           "両方のパターン",
			includePattern: "*.txt",
			excludePattern: "*.tmp",
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewFilter(tt.includePattern, tt.excludePattern)
			result := filter.HasPatterns()

			if result != tt.expectedResult {
				t.Errorf("HasPatterns() = %v, 期待値 %v", result, tt.expectedResult)
			}
		})
	}
}

func TestMatchesPath(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		patterns       string
		expectedResult bool
	}{
		{
			name:           "空のパターン",
			path:           "test.txt",
			patterns:       "",
			expectedResult: false,
		},
		{
			name:           "単一パターンに一致",
			path:           "test.txt",
			patterns:       "*.txt",
			expectedResult: true,
		},
		{
			name:           "単一パターンに一致しない",
			path:           "test.doc",
			patterns:       "*.txt",
			expectedResult: false,
		},
		{
			name:           "複数パターンに一致",
			path:           "test.doc",
			patterns:       "*.txt,*.doc",
			expectedResult: true,
		},
		{
			name:           "複数パターンに一致しない",
			path:           "test.pdf",
			patterns:       "*.txt,*.doc",
			expectedResult: false,
		},
		{
			name:           "空白を含むパターン",
			path:           "test.txt",
			patterns:       " *.txt , *.doc ",
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MatchesPath(tt.path, tt.patterns)

			if result != tt.expectedResult {
				t.Errorf("MatchesPath() = %v, 期待値 %v", result, tt.expectedResult)
			}
		})
	}
}

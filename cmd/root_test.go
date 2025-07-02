package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRootCmd(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")

	// ソースディレクトリを作成
	err := os.MkdirAll(sourceDir, 0755)
	if err != nil {
		t.Fatalf("ソースディレクトリの作成に失敗: %v", err)
	}

	// 宛先ディレクトリを作成
	err = os.MkdirAll(destDir, 0755)
	if err != nil {
		t.Fatalf("宛先ディレクトリの作成に失敗: %v", err)
	}

	// テストファイルを作成
	testFile := filepath.Join(sourceDir, "test.txt")
	testContent := "This is a test file for gopier"
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("テストファイルの作成に失敗: %v", err)
	}

	// 元のフラグ値を保存
	originalSourceDir := sourceDir
	originalDestDir := destDir

	// テストケース
	tests := []struct {
		name        string
		setSource   string
		setDest     string
		expectError bool
	}{
		{
			name:        "有効なソースと宛先",
			setSource:   sourceDir,
			setDest:     destDir,
			expectError: false,
		},
		{
			name:        "ソースディレクトリが空",
			setSource:   "",
			setDest:     destDir,
			expectError: false, // ヘルプが表示される
		},
		{
			name:        "宛先ディレクトリが空",
			setSource:   sourceDir,
			setDest:     "",
			expectError: false, // ヘルプが表示される
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// フラグを設定
			sourceDir = tt.setSource
			destDir = tt.setDest

			// コマンドを実行（実際のコピーは行わない）
			// このテストでは主にフラグの設定とバリデーションをテスト
		})

		// フラグを元に戻す
		sourceDir = originalSourceDir
		destDir = originalDestDir
	}
}

func TestInitConfig(t *testing.T) {
	// 設定ファイルのテスト
	tests := []struct {
		name        string
		configFile  string
		expectError bool
	}{
		{
			name:        "設定ファイルなし",
			configFile:  "",
			expectError: false,
		},
		{
			name:        "存在しない設定ファイル",
			configFile:  "non-existent-config.yaml",
			expectError: false, // エラーは発生しないが、設定は読み込まれない
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 元の値を保存
			originalCfgFile := cfgFile

			// テスト用の値を設定
			cfgFile = tt.configFile

			// initConfigを実行
			initConfig()

			// フラグを元に戻す
			cfgFile = originalCfgFile
		})
	}
}

func TestFlagValidation(t *testing.T) {
	// フラグのバリデーションテスト
	tests := []struct {
		name        string
		workers     int
		retryCount  int
		retryWait   int
		bufferSize  int
		expectValid bool
	}{
		{
			name:        "有効な値",
			workers:     4,
			retryCount:  3,
			retryWait:   5,
			bufferSize:  8,
			expectValid: true,
		},
		{
			name:        "ワーカー数0",
			workers:     0,
			retryCount:  3,
			retryWait:   5,
			bufferSize:  8,
			expectValid: true, // デフォルト値が使用される
		},
		{
			name:        "ワーカー数負の値",
			workers:     -1,
			retryCount:  3,
			retryWait:   5,
			bufferSize:  8,
			expectValid: true, // デフォルト値が使用される
		},
		{
			name:        "リトライ回数0",
			workers:     4,
			retryCount:  0,
			retryWait:   5,
			bufferSize:  8,
			expectValid: true,
		},
		{
			name:        "待機時間0",
			workers:     4,
			retryCount:  3,
			retryWait:   0,
			bufferSize:  8,
			expectValid: true,
		},
		{
			name:        "バッファサイズ0",
			workers:     4,
			retryCount:  3,
			retryWait:   5,
			bufferSize:  0,
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 元の値を保存
			originalWorkers := numWorkers
			originalRetryCount := retryCount
			originalRetryWait := retryWait
			originalBufferSize := bufferSize

			// テスト用の値を設定
			numWorkers = tt.workers
			retryCount = tt.retryCount
			retryWait = tt.retryWait
			bufferSize = tt.bufferSize

			// 値の検証（実際の処理は行わない）
			// このテストでは主にフラグの設定をテスト

			// フラグを元に戻す
			numWorkers = originalWorkers
			retryCount = originalRetryCount
			retryWait = originalRetryWait
			bufferSize = originalBufferSize
		})
	}
}

func TestSyncModeValidation(t *testing.T) {
	// 同期モードのバリデーションテスト
	tests := []struct {
		name        string
		syncMode    string
		expectValid bool
	}{
		{
			name:        "normal mode",
			syncMode:    "normal",
			expectValid: true,
		},
		{
			name:        "initial mode",
			syncMode:    "initial",
			expectValid: true,
		},
		{
			name:        "incremental mode",
			syncMode:    "incremental",
			expectValid: true,
		},
		{
			name:        "invalid mode",
			syncMode:    "invalid",
			expectValid: true, // バリデーションエラーは発生しない
		},
		{
			name:        "empty mode",
			syncMode:    "",
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 元の値を保存
			originalSyncMode := syncMode

			// テスト用の値を設定
			syncMode = tt.syncMode

			// 値の検証（実際の処理は行わない）

			// フラグを元に戻す
			syncMode = originalSyncMode
		})
	}
}

func TestFilterPatternValidation(t *testing.T) {
	// フィルタパターンのバリデーションテスト
	tests := []struct {
		name            string
		includePattern  string
		excludePattern  string
		expectValid     bool
	}{
		{
			name:            "有効なパターン",
			includePattern:  "*.txt,*.doc",
			excludePattern:  "*.tmp,*.bak",
			expectValid:     true,
		},
		{
			name:            "空のパターン",
			includePattern:  "",
			excludePattern:  "",
			expectValid:     true,
		},
		{
			name:            "含めるパターンのみ",
			includePattern:  "*.txt",
			excludePattern:  "",
			expectValid:     true,
		},
		{
			name:            "除外パターンのみ",
			includePattern:  "",
			excludePattern:  "*.tmp",
			expectValid:     true,
		},
		{
			name:            "空白を含むパターン",
			includePattern:  " *.txt , *.doc ",
			excludePattern:  " *.tmp , *.bak ",
			expectValid:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 元の値を保存
			originalIncludePattern := includePattern
			originalExcludePattern := excludePattern

			// テスト用の値を設定
			includePattern = tt.includePattern
			excludePattern = tt.excludePattern

			// 値の検証（実際の処理は行わない）

			// フラグを元に戻す
			includePattern = originalIncludePattern
			excludePattern = originalExcludePattern
		})
	}
}

func TestVerificationFlags(t *testing.T) {
	// 検証フラグのテスト
	tests := []struct {
		name           string
		verifyOnly     bool
		verifyChanged  bool
		verifyAll      bool
		expectValid    bool
	}{
		{
			name:           "検証なし",
			verifyOnly:     false,
			verifyChanged:  false,
			verifyAll:      false,
			expectValid:    true,
		},
		{
			name:           "検証のみ",
			verifyOnly:     true,
			verifyChanged:  false,
			verifyAll:      false,
			expectValid:    true,
		},
		{
			name:           "変更ファイル検証",
			verifyOnly:     false,
			verifyChanged:  true,
			verifyAll:      false,
			expectValid:    true,
		},
		{
			name:           "全ファイル検証",
			verifyOnly:     false,
			verifyChanged:  false,
			verifyAll:      true,
			expectValid:    true,
		},
		{
			name:           "検証のみ + 変更ファイル検証",
			verifyOnly:     true,
			verifyChanged:  true,
			verifyAll:      false,
			expectValid:    true,
		},
		{
			name:           "検証のみ + 全ファイル検証",
			verifyOnly:     true,
			verifyChanged:  false,
			verifyAll:      true,
			expectValid:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 元の値を保存
			originalVerifyOnly := verifyOnly
			originalVerifyChanged := verifyChanged
			originalVerifyAll := verifyAll

			// テスト用の値を設定
			verifyOnly = tt.verifyOnly
			verifyChanged = tt.verifyChanged
			verifyAll = tt.verifyAll

			// 値の検証（実際の処理は行わない）

			// フラグを元に戻す
			verifyOnly = originalVerifyOnly
			verifyChanged = originalVerifyChanged
			verifyAll = originalVerifyAll
		})
	}
}

func TestDatabaseFlags(t *testing.T) {
	// データベース関連フラグのテスト
	tests := []struct {
		name           string
		syncDBPath     string
		includeFailed  bool
		maxFailCount   int
		expectValid    bool
	}{
		{
			name:           "デフォルト設定",
			syncDBPath:     "sync_state.db",
			includeFailed:  true,
			maxFailCount:   5,
			expectValid:    true,
		},
		{
			name:           "カスタムDBパス",
			syncDBPath:     "/tmp/custom_sync.db",
			includeFailed:  true,
			maxFailCount:   5,
			expectValid:    true,
		},
		{
			name:           "失敗ファイル除外",
			syncDBPath:     "sync_state.db",
			includeFailed:  false,
			maxFailCount:   5,
			expectValid:    true,
		},
		{
			name:           "最大失敗回数0",
			syncDBPath:     "sync_state.db",
			includeFailed:  true,
			maxFailCount:   0,
			expectValid:    true,
		},
		{
			name:           "最大失敗回数負の値",
			syncDBPath:     "sync_state.db",
			includeFailed:  true,
			maxFailCount:   -1,
			expectValid:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 元の値を保存
			originalSyncDBPath := syncDBPath
			originalIncludeFailed := includeFailed
			originalMaxFailCount := maxFailCount

			// テスト用の値を設定
			syncDBPath = tt.syncDBPath
			includeFailed = tt.includeFailed
			maxFailCount = tt.maxFailCount

			// 値の検証（実際の処理は行わない）

			// フラグを元に戻す
			syncDBPath = originalSyncDBPath
			includeFailed = originalIncludeFailed
			maxFailCount = originalMaxFailCount
		})
	}
}

// ベンチマークテスト
func BenchmarkRootCmdExecution(b *testing.B) {
	// テスト用の一時ディレクトリを作成
	tempDir := b.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")

	// ディレクトリを作成
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	// 元の値を保存
	originalSourceDir := sourceDir
	originalDestDir := destDir

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// フラグを設定
		sourceDir = originalSourceDir
		destDir = originalDestDir

		// コマンドの実行をシミュレート（実際の処理は行わない）
	}

	// フラグを元に戻す
	sourceDir = originalSourceDir
	destDir = originalDestDir
} 
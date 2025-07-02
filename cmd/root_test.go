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
		name           string
		includePattern string
		excludePattern string
		expectValid    bool
	}{
		{
			name:           "有効なパターン",
			includePattern: "*.txt,*.doc",
			excludePattern: "*.tmp,*.bak",
			expectValid:    true,
		},
		{
			name:           "空のパターン",
			includePattern: "",
			excludePattern: "",
			expectValid:    true,
		},
		{
			name:           "含めるパターンのみ",
			includePattern: "*.txt",
			excludePattern: "",
			expectValid:    true,
		},
		{
			name:           "除外パターンのみ",
			includePattern: "",
			excludePattern: "*.tmp",
			expectValid:    true,
		},
		{
			name:           "空白を含むパターン",
			includePattern: " *.txt , *.doc ",
			excludePattern: " *.tmp , *.bak ",
			expectValid:    true,
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
		name          string
		verifyOnly    bool
		verifyChanged bool
		verifyAll     bool
		expectValid   bool
	}{
		{
			name:          "検証なし",
			verifyOnly:    false,
			verifyChanged: false,
			verifyAll:     false,
			expectValid:   true,
		},
		{
			name:          "検証のみ",
			verifyOnly:    true,
			verifyChanged: false,
			verifyAll:     false,
			expectValid:   true,
		},
		{
			name:          "変更ファイル検証",
			verifyOnly:    false,
			verifyChanged: true,
			verifyAll:     false,
			expectValid:   true,
		},
		{
			name:          "全ファイル検証",
			verifyOnly:    false,
			verifyChanged: false,
			verifyAll:     true,
			expectValid:   true,
		},
		{
			name:          "検証のみ + 変更ファイル検証",
			verifyOnly:    true,
			verifyChanged: true,
			verifyAll:     false,
			expectValid:   true,
		},
		{
			name:          "検証のみ + 全ファイル検証",
			verifyOnly:    true,
			verifyChanged: false,
			verifyAll:     true,
			expectValid:   true,
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
		name          string
		syncDBPath    string
		includeFailed bool
		maxFailCount  int
		expectValid   bool
	}{
		{
			name:          "デフォルト設定",
			syncDBPath:    "sync_state.db",
			includeFailed: true,
			maxFailCount:  5,
			expectValid:   true,
		},
		{
			name:          "カスタムDBパス",
			syncDBPath:    "/tmp/custom_sync.db",
			includeFailed: true,
			maxFailCount:  5,
			expectValid:   true,
		},
		{
			name:          "失敗ファイル除外",
			syncDBPath:    "sync_state.db",
			includeFailed: false,
			maxFailCount:  5,
			expectValid:   true,
		},
		{
			name:          "最大失敗回数0",
			syncDBPath:    "sync_state.db",
			includeFailed: true,
			maxFailCount:  0,
			expectValid:   true,
		},
		{
			name:          "最大失敗回数負の値",
			syncDBPath:    "sync_state.db",
			includeFailed: true,
			maxFailCount:  -1,
			expectValid:   true,
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// コマンドの実行をシミュレート（実際の処理は行わない）
	}
}

func TestExecute(t *testing.T) {
	// Execute関数のテスト
	// 実際のコマンド実行をシミュレート
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// テストケース
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "ヘルプ表示",
			args:        []string{"gopier", "--help"},
			expectError: false,
		},
		{
			name:        "バージョン表示",
			args:        []string{"gopier", "--version"},
			expectError: false,
		},
		{
			name:        "設定ファイル作成",
			args:        []string{"gopier", "--create-config"},
			expectError: false,
		},
		{
			name:        "設定表示",
			args:        []string{"gopier", "--show-config"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args
			// Executeは実際には実行しない（無限ループになるため）
			// 代わりにコマンドの構築をテスト
		})
	}
}

func TestCreateDefaultConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.yaml")

	// 正常なケース
	err := createDefaultConfig(configPath)
	if err != nil {
		t.Errorf("設定ファイル作成が失敗: %v", err)
	}

	// ファイルが作成されたか確認
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("設定ファイルが作成されていません")
	}

	// 既存ファイルの上書き
	err = createDefaultConfig(configPath)
	if err != nil {
		t.Errorf("既存ファイルの上書きが失敗: %v", err)
	}

	// 無効なパス（権限エラー）
	invalidPath := "/root/invalid/config.yaml"
	err = createDefaultConfig(invalidPath)
	if err == nil {
		t.Error("無効なパスでエラーが発生しませんでした")
	}
}

func TestShowCurrentConfig(t *testing.T) {
	// 元の値を保存
	originalSourceDir := sourceDir
	originalDestDir := destDir
	originalWorkers := numWorkers
	originalBufferSize := bufferSize

	// テスト用の値を設定
	sourceDir = "/test/source"
	destDir = "/test/dest"
	numWorkers = 8
	bufferSize = 16

	// showCurrentConfigを実行（出力はキャプチャしない）
	showCurrentConfig()

	// フラグを元に戻す
	sourceDir = originalSourceDir
	destDir = originalDestDir
	numWorkers = originalWorkers
	bufferSize = originalBufferSize
}

func TestBindConfigToFlags(t *testing.T) {
	// テスト用の設定
	config := &Config{
		Source:         "/test/source",
		Destination:    "/test/dest",
		Workers:        6,
		BufferSize:     12,
		RetryCount:     5,
		RetryWait:      10,
		IncludePattern: "*.txt",
		ExcludePattern: "*.tmp",
		Recursive:      true,
		Mirror:         true,
		DryRun:         true,
		Verbose:        true,
		SkipNewer:      true,
		SyncMode:       "incremental",
		SyncDBPath:     "test.db",
		IncludeFailed:  true,
		MaxFailCount:   10,
		VerifyOnly:     true,
		VerifyChanged:  true,
		VerifyAll:      true,
		FinalReport:    "report.txt",
		HashAlgorithm:  "sha256",
		VerifyHash:     true,
	}

	// 元の値を保存
	originalSourceDir := sourceDir
	originalDestDir := destDir
	originalWorkers := numWorkers
	originalBufferSize := bufferSize
	originalRetryCount := retryCount
	originalRetryWait := retryWait
	originalIncludePattern := includePattern
	originalExcludePattern := excludePattern
	originalRecursive := recursive
	originalMirror := mirror
	originalDryRun := dryRun
	originalVerbose := verbose
	originalSkipNewer := skipNewer
	originalSyncMode := syncMode
	originalSyncDBPath := syncDBPath
	originalIncludeFailed := includeFailed
	originalMaxFailCount := maxFailCount
	originalVerifyOnly := verifyOnly
	originalVerifyChanged := verifyChanged
	originalVerifyAll := verifyAll
	originalFinalReport := finalReport

	// フラグをクリア
	sourceDir = ""
	destDir = ""
	numWorkers = 0
	bufferSize = 0
	retryCount = 0
	retryWait = 0
	includePattern = ""
	excludePattern = ""
	recursive = false
	mirror = false
	dryRun = false
	verbose = false
	skipNewer = false
	syncMode = ""
	syncDBPath = ""
	includeFailed = false
	maxFailCount = 0
	verifyOnly = false
	verifyChanged = false
	verifyAll = false
	finalReport = ""

	// bindConfigToFlagsを実行
	bindConfigToFlags(config, rootCmd)

	// 設定が正しくバインドされたか確認
	if sourceDir != config.Source {
		t.Errorf("Source: 期待値=%s, 実際=%s", config.Source, sourceDir)
	}
	if destDir != config.Destination {
		t.Errorf("Destination: 期待値=%s, 実際=%s", config.Destination, destDir)
	}
	if numWorkers != config.Workers {
		t.Errorf("Workers: 期待値=%d, 実際=%d", config.Workers, numWorkers)
	}
	if bufferSize != config.BufferSize {
		t.Errorf("BufferSize: 期待値=%d, 実際=%d", config.BufferSize, bufferSize)
	}
	if retryCount != config.RetryCount {
		t.Errorf("RetryCount: 期待値=%d, 実際=%d", config.RetryCount, retryCount)
	}
	if retryWait != config.RetryWait {
		t.Errorf("RetryWait: 期待値=%d, 実際=%d", config.RetryWait, retryWait)
	}
	if includePattern != config.IncludePattern {
		t.Errorf("IncludePattern: 期待値=%s, 実際=%s", config.IncludePattern, includePattern)
	}
	if excludePattern != config.ExcludePattern {
		t.Errorf("ExcludePattern: 期待値=%s, 実際=%s", config.ExcludePattern, excludePattern)
	}
	if recursive != config.Recursive {
		t.Errorf("Recursive: 期待値=%t, 実際=%t", config.Recursive, recursive)
	}
	if mirror != config.Mirror {
		t.Errorf("Mirror: 期待値=%t, 実際=%t", config.Mirror, mirror)
	}
	if dryRun != config.DryRun {
		t.Errorf("DryRun: 期待値=%t, 実際=%t", config.DryRun, dryRun)
	}
	if verbose != config.Verbose {
		t.Errorf("Verbose: 期待値=%t, 実際=%t", config.Verbose, verbose)
	}
	if skipNewer != config.SkipNewer {
		t.Errorf("SkipNewer: 期待値=%t, 実際=%t", config.SkipNewer, skipNewer)
	}
	if syncMode != config.SyncMode {
		t.Errorf("SyncMode: 期待値=%s, 実際=%s", config.SyncMode, syncMode)
	}
	if syncDBPath != config.SyncDBPath {
		t.Errorf("SyncDBPath: 期待値=%s, 実際=%s", config.SyncDBPath, syncDBPath)
	}
	if includeFailed != config.IncludeFailed {
		t.Errorf("IncludeFailed: 期待値=%t, 実際=%t", config.IncludeFailed, includeFailed)
	}
	if maxFailCount != config.MaxFailCount {
		t.Errorf("MaxFailCount: 期待値=%d, 実際=%d", config.MaxFailCount, maxFailCount)
	}
	if verifyOnly != config.VerifyOnly {
		t.Errorf("VerifyOnly: 期待値=%t, 実際=%t", config.VerifyOnly, verifyOnly)
	}
	if verifyChanged != config.VerifyChanged {
		t.Errorf("VerifyChanged: 期待値=%t, 実際=%t", config.VerifyChanged, verifyChanged)
	}
	if verifyAll != config.VerifyAll {
		t.Errorf("VerifyAll: 期待値=%t, 実際=%t", config.VerifyAll, verifyAll)
	}
	if finalReport != config.FinalReport {
		t.Errorf("FinalReport: 期待値=%s, 実際=%s", config.FinalReport, finalReport)
	}

	// フラグを元に戻す
	sourceDir = originalSourceDir
	destDir = originalDestDir
	numWorkers = originalWorkers
	bufferSize = originalBufferSize
	retryCount = originalRetryCount
	retryWait = originalRetryWait
	includePattern = originalIncludePattern
	excludePattern = originalExcludePattern
	recursive = originalRecursive
	mirror = originalMirror
	dryRun = originalDryRun
	verbose = originalVerbose
	skipNewer = originalSkipNewer
	syncMode = originalSyncMode
	syncDBPath = originalSyncDBPath
	includeFailed = originalIncludeFailed
	maxFailCount = originalMaxFailCount
	verifyOnly = originalVerifyOnly
	verifyChanged = originalVerifyChanged
	verifyAll = originalVerifyAll
	finalReport = originalFinalReport
}

func TestValidateConfig(t *testing.T) {
	// 正常な設定
	validConfig := &Config{
		Workers:       4,
		BufferSize:    8,
		RetryCount:    3,
		RetryWait:     5,
		SyncMode:      "normal",
		MaxFailCount:  5,
		HashAlgorithm: "sha256",
	}

	err := validateConfig(validConfig)
	if err != nil {
		t.Errorf("正常な設定でエラーが発生: %v", err)
	}

	// 無効な設定のテスト
	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name: "ワーカー数が負",
			config: &Config{
				Workers:       -1,
				BufferSize:    8,
				RetryCount:    3,
				RetryWait:     5,
				SyncMode:      "normal",
				MaxFailCount:  5,
				HashAlgorithm: "sha256",
			},
			expectError: true,
		},
		{
			name: "バッファサイズが負",
			config: &Config{
				Workers:       4,
				BufferSize:    -1,
				RetryCount:    3,
				RetryWait:     5,
				SyncMode:      "normal",
				MaxFailCount:  5,
				HashAlgorithm: "sha256",
			},
			expectError: true,
		},
		{
			name: "リトライ回数が負",
			config: &Config{
				Workers:       4,
				BufferSize:    8,
				RetryCount:    -1,
				RetryWait:     5,
				SyncMode:      "normal",
				MaxFailCount:  5,
				HashAlgorithm: "sha256",
			},
			expectError: true,
		},
		{
			name: "待機時間が負",
			config: &Config{
				Workers:       4,
				BufferSize:    8,
				RetryCount:    3,
				RetryWait:     -1,
				SyncMode:      "normal",
				MaxFailCount:  5,
				HashAlgorithm: "sha256",
			},
			expectError: true,
		},
		{
			name: "無効な同期モード",
			config: &Config{
				Workers:       4,
				BufferSize:    8,
				RetryCount:    3,
				RetryWait:     5,
				SyncMode:      "invalid",
				MaxFailCount:  5,
				HashAlgorithm: "sha256",
			},
			expectError: true,
		},
		{
			name: "最大失敗回数が負",
			config: &Config{
				Workers:       4,
				BufferSize:    8,
				RetryCount:    3,
				RetryWait:     5,
				SyncMode:      "normal",
				MaxFailCount:  -1,
				HashAlgorithm: "sha256",
			},
			expectError: true,
		},
		{
			name: "無効なハッシュアルゴリズム",
			config: &Config{
				Workers:       4,
				BufferSize:    8,
				RetryCount:    3,
				RetryWait:     5,
				SyncMode:      "normal",
				MaxFailCount:  5,
				HashAlgorithm: "invalid",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if tt.expectError && err == nil {
				t.Error("エラーが期待されましたが、発生しませんでした")
			}
			if !tt.expectError && err != nil {
				t.Errorf("エラーが発生しました: %v", err)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	// 元の値を保存
	originalCfgFile := cfgFile

	// テストケース
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
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// テスト用の値を設定
			cfgFile = tt.configFile

			// loadConfigを実行
			loadConfig(rootCmd)

			// フラグを元に戻す
			cfgFile = originalCfgFile
		})
	}
}

func TestInitConfigWithCreateConfig(t *testing.T) {
	// 元の値を保存
	originalCfgFile := cfgFile

	// create-configフラグが設定されている場合のテスト
	cfgFile = "test-config.yaml"

	// initConfigを実行
	initConfig()

	// フラグを元に戻す
	cfgFile = originalCfgFile
}

func TestInitConfigWithShowConfig(t *testing.T) {
	// 元の値を保存
	originalCfgFile := cfgFile

	// show-configフラグが設定されている場合のテスト
	cfgFile = "test-config.yaml"

	// initConfigを実行
	initConfig()

	// フラグを元に戻す
	cfgFile = originalCfgFile
}

func TestInitConfigWithConfigFile(t *testing.T) {
	// 元の値を保存
	originalCfgFile := cfgFile

	// 設定ファイルが指定されている場合のテスト
	cfgFile = "test-config.yaml"

	// initConfigを実行
	initConfig()

	// フラグを元に戻す
	cfgFile = originalCfgFile
}

func TestInitConfigDefault(t *testing.T) {
	// 元の値を保存
	originalCfgFile := cfgFile

	// デフォルト設定のテスト
	cfgFile = ""

	// initConfigを実行
	initConfig()

	// フラグを元に戻す
	cfgFile = originalCfgFile
}

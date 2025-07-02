package verifier

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sakuhanight/gopier/internal/database"
	"github.com/sakuhanight/gopier/internal/filter"
	"github.com/sakuhanight/gopier/internal/hasher"
)

// TestDefaultOptions はDefaultOptions関数のテスト
func TestDefaultOptions(t *testing.T) {
	options := DefaultOptions()

	// デフォルト値の検証
	if options.BufferSize != 32*1024*1024 {
		t.Errorf("期待されるバッファサイズ: %d, 実際: %d", 32*1024*1024, options.BufferSize)
	}

	if !options.Recursive {
		t.Error("Recursiveはデフォルトでtrueであるべき")
	}

	if options.HashAlgorithm != string(hasher.SHA256) {
		t.Errorf("期待されるハッシュアルゴリズム: %s, 実際: %s", hasher.SHA256, options.HashAlgorithm)
	}

	if options.ProgressInterval != time.Second*1 {
		t.Errorf("期待される進捗間隔: %v, 実際: %v", time.Second*1, options.ProgressInterval)
	}

	if options.MaxConcurrent != 4 {
		t.Errorf("期待される最大並行数: %d, 実際: %d", 4, options.MaxConcurrent)
	}

	if options.FailFast {
		t.Error("FailFastはデフォルトでfalseであるべき")
	}

	if options.IgnoreMissing {
		t.Error("IgnoreMissingはデフォルトでfalseであるべき")
	}

	if options.IgnoreExtra {
		t.Error("IgnoreExtraはデフォルトでfalseであるべき")
	}
}

// TestNewVerifier はNewVerifier関数のテスト
func TestNewVerifier(t *testing.T) {
	sourceDir := "/source"
	destDir := "/dest"
	options := DefaultOptions()
	fileFilter := filter.NewFilter("*.txt", "*.tmp")
	syncDB := &database.SyncDB{}

	verifier := NewVerifier(sourceDir, destDir, options, fileFilter, syncDB)

	if verifier == nil {
		t.Fatal("NewVerifierはnilを返すべきではありません")
	}

	if verifier.sourceDir != sourceDir {
		t.Errorf("期待されるソースディレクトリ: %s, 実際: %s", sourceDir, verifier.sourceDir)
	}

	if verifier.destDir != destDir {
		t.Errorf("期待される宛先ディレクトリ: %s, 実際: %s", destDir, verifier.destDir)
	}

	if verifier.filter != fileFilter {
		t.Error("フィルターが正しく設定されていません")
	}

	if verifier.db != syncDB {
		t.Error("データベースが正しく設定されていません")
	}

	if verifier.hasher == nil {
		t.Error("ハッシャーが初期化されていません")
	}

	if verifier.stats == nil {
		t.Error("統計情報が初期化されていません")
	}

	if verifier.progressChan == nil {
		t.Error("進捗チャンネルが初期化されていません")
	}

	if verifier.semaphore == nil {
		t.Error("セマフォが初期化されていません")
	}

	if verifier.ctx == nil {
		t.Error("コンテキストが初期化されていません")
	}

	if verifier.cancel == nil {
		t.Error("キャンセル関数が初期化されていません")
	}
}

// TestVerifierMethods はVerifierの基本メソッドのテスト
func TestVerifierMethods(t *testing.T) {
	verifier := NewVerifier("/source", "/dest", DefaultOptions(), nil, nil)

	// SetProgressCallbackのテスト
	callback := func(current, total int64, currentFile string) {
		// コールバックが呼ばれたことを確認するための処理
	}
	verifier.SetProgressCallback(callback)

	if verifier.progressFunc == nil {
		t.Error("プログレスコールバックが設定されていません")
	}

	// GetStatsのテスト
	stats := verifier.GetStats()
	if stats == nil {
		t.Error("統計情報が返されていません")
	}

	// GetResultsのテスト
	results := verifier.GetResults()
	if results == nil {
		t.Error("結果が返されていません")
	}

	// GetErrorCountのテスト
	errorCount := verifier.GetErrorCount()
	if errorCount != 0 {
		t.Errorf("期待されるエラー数: 0, 実際: %d", errorCount)
	}

	// Cancelのテスト
	verifier.Cancel()
	select {
	case <-verifier.ctx.Done():
		// 期待される動作
	default:
		t.Error("コンテキストがキャンセルされていません")
	}
}

// TestAddResult はaddResultメソッドのテスト
func TestAddResult(t *testing.T) {
	verifier := NewVerifier("/source", "/dest", DefaultOptions(), nil, nil)

	// 正常な結果の追加
	result := VerificationResult{
		Path:         "test.txt",
		SourceExists: true,
		DestExists:   true,
		SizeMatch:    true,
		HashMatch:    true,
	}

	verifier.addResult(result)

	results := verifier.GetResults()
	if len(results) != 1 {
		t.Errorf("期待される結果数: 1, 実際: %d", len(results))
	}

	if results[0].Path != "test.txt" {
		t.Errorf("期待されるパス: test.txt, 実際: %s", results[0].Path)
	}

	// エラー結果の追加
	errorResult := VerificationResult{
		Path:  "error.txt",
		Error: fmt.Errorf("テストエラー"),
	}

	verifier.addResult(errorResult)

	errorCount := verifier.GetErrorCount()
	if errorCount != 1 {
		t.Errorf("期待されるエラー数: 1, 実際: %d", errorCount)
	}
}

// TestAddResultFailFast はFailFastオプションのテスト
func TestAddResultFailFast(t *testing.T) {
	options := DefaultOptions()
	options.FailFast = true
	verifier := NewVerifier("/source", "/dest", options, nil, nil)

	// エラー結果を追加
	errorResult := VerificationResult{
		Path:  "error.txt",
		Error: fmt.Errorf("テストエラー"),
	}

	verifier.addResult(errorResult)

	// コンテキストがキャンセルされているかチェック
	select {
	case <-verifier.ctx.Done():
		// 期待される動作
	default:
		t.Error("FailFastが有効な場合、コンテキストがキャンセルされるべきです")
	}
}

// TestVerifyFile はverifyFileメソッドのテスト
func TestVerifyFile(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "verifier_test")
	if err != nil {
		t.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("ソースディレクトリの作成に失敗: %v", err)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("宛先ディレクトリの作成に失敗: %v", err)
	}

	verifier := NewVerifier(sourceDir, destDir, DefaultOptions(), nil, nil)

	// テストケース1: 存在しないソースファイル
	result, err := verifier.verifyFile("/nonexistent/source.txt", filepath.Join(destDir, "dest.txt"))
	if err != nil {
		t.Errorf("エラーが発生すべきではありません: %v", err)
	}
	if result == nil {
		t.Fatal("結果が返されるべきです")
	}
	if result.SourceExists {
		t.Error("ソースファイルが存在しない場合、SourceExistsはfalseであるべきです")
	}

	// テストケース2: 存在しない宛先ファイル
	sourceFile := filepath.Join(sourceDir, "source.txt")
	if err := os.WriteFile(sourceFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("ソースファイルの作成に失敗: %v", err)
	}

	result, err = verifier.verifyFile(sourceFile, filepath.Join(destDir, "nonexistent.txt"))
	if err != nil {
		t.Errorf("エラーが発生すべきではありません: %v", err)
	}
	if result == nil {
		t.Fatal("結果が返されるべきです")
	}
	if result.DestExists {
		t.Error("宛先ファイルが存在しない場合、DestExistsはfalseであるべきです")
	}

	// テストケース3: 同じ内容のファイル
	destFile := filepath.Join(destDir, "dest.txt")
	if err := os.WriteFile(destFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("宛先ファイルの作成に失敗: %v", err)
	}

	result, err = verifier.verifyFile(sourceFile, destFile)
	if err != nil {
		t.Errorf("エラーが発生すべきではありません: %v", err)
	}
	if result == nil {
		t.Fatal("結果が返されるべきです")
	}
	if !result.SourceExists {
		t.Error("ソースファイルが存在する場合、SourceExistsはtrueであるべきです")
	}
	if !result.DestExists {
		t.Error("宛先ファイルが存在する場合、DestExistsはtrueであるべきです")
	}
	if !result.SizeMatch {
		t.Error("同じサイズのファイルの場合、SizeMatchはtrueであるべきです")
	}
	if !result.HashMatch {
		t.Error("同じ内容のファイルの場合、HashMatchはtrueであるべきです")
	}

	// テストケース4: 異なる内容のファイル
	destFile2 := filepath.Join(destDir, "dest2.txt")
	if err := os.WriteFile(destFile2, []byte("different content"), 0644); err != nil {
		t.Fatalf("宛先ファイルの作成に失敗: %v", err)
	}

	result, err = verifier.verifyFile(sourceFile, destFile2)
	if err != nil {
		t.Errorf("エラーが発生すべきではありません: %v", err)
	}
	if result == nil {
		t.Fatal("結果が返されるべきです")
	}
	if result.HashMatch {
		t.Error("異なる内容のファイルの場合、HashMatchはfalseであるべきです")
	}
}

// TestVerifyFileWithIgnoreMissing はIgnoreMissingオプションのテスト
func TestVerifyFileWithIgnoreMissing(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "verifier_test")
	if err != nil {
		t.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("ソースディレクトリの作成に失敗: %v", err)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("宛先ディレクトリの作成に失敗: %v", err)
	}

	options := DefaultOptions()
	options.IgnoreMissing = true
	verifier := NewVerifier(sourceDir, destDir, options, nil, nil)

	// ソースファイルを作成
	sourceFile := filepath.Join(sourceDir, "source.txt")
	if err := os.WriteFile(sourceFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("ソースファイルの作成に失敗: %v", err)
	}

	// 宛先ファイルが存在しない場合、nilが返されるべき
	result, err := verifier.verifyFile(sourceFile, filepath.Join(destDir, "nonexistent.txt"))
	if err != nil {
		t.Errorf("エラーが発生すべきではありません: %v", err)
	}
	if result != nil {
		t.Error("IgnoreMissingが有効な場合、結果はnilであるべきです")
	}
}

// TestVerifyDirectory はverifyDirectoryメソッドのテスト
func TestVerifyDirectory(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "verifier_test")
	if err != nil {
		t.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("ソースディレクトリの作成に失敗: %v", err)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("宛先ディレクトリの作成に失敗: %v", err)
	}

	// テストファイルを作成
	sourceFile := filepath.Join(sourceDir, "test.txt")
	destFile := filepath.Join(destDir, "test.txt")
	if err := os.WriteFile(sourceFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("ソースファイルの作成に失敗: %v", err)
	}
	if err := os.WriteFile(destFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("宛先ファイルの作成に失敗: %v", err)
	}

	verifier := NewVerifier(sourceDir, destDir, DefaultOptions(), nil, nil)

	// ディレクトリの検証
	err = verifier.verifyDirectory(sourceDir, destDir)
	if err != nil {
		t.Errorf("ディレクトリ検証でエラーが発生: %v", err)
	}

	// 非同期処理の完了を待つ
	verifier.wg.Wait()

	// 結果を確認
	results := verifier.GetResults()
	if len(results) == 0 {
		t.Error("検証結果が記録されていません")
	}

	// 成功した結果があるかチェック
	successFound := false
	for _, result := range results {
		if result.Path == "test.txt" && result.HashMatch {
			successFound = true
			break
		}
	}
	if !successFound {
		t.Error("成功した検証結果が見つかりません")
	}
}

// TestVerifyDirectoryWithFilter はフィルター付きのディレクトリ検証テスト
func TestVerifyDirectoryWithFilter(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "verifier_test")
	if err != nil {
		t.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("ソースディレクトリの作成に失敗: %v", err)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("宛先ディレクトリの作成に失敗: %v", err)
	}

	// テストファイルを作成
	sourceFile1 := filepath.Join(sourceDir, "test.txt")
	sourceFile2 := filepath.Join(sourceDir, "test.log")
	destFile1 := filepath.Join(destDir, "test.txt")
	destFile2 := filepath.Join(destDir, "test.log")

	if err := os.WriteFile(sourceFile1, []byte("test content"), 0644); err != nil {
		t.Fatalf("ソースファイルの作成に失敗: %v", err)
	}
	if err := os.WriteFile(sourceFile2, []byte("log content"), 0644); err != nil {
		t.Fatalf("ソースファイルの作成に失敗: %v", err)
	}
	if err := os.WriteFile(destFile1, []byte("test content"), 0644); err != nil {
		t.Fatalf("宛先ファイルの作成に失敗: %v", err)
	}
	if err := os.WriteFile(destFile2, []byte("log content"), 0644); err != nil {
		t.Fatalf("宛先ファイルの作成に失敗: %v", err)
	}

	// .txtファイルのみを含むフィルターを作成
	fileFilter := filter.NewFilter("*.txt", "")
	verifier := NewVerifier(sourceDir, destDir, DefaultOptions(), fileFilter, nil)

	// ディレクトリの検証
	err = verifier.verifyDirectory(sourceDir, destDir)
	if err != nil {
		t.Errorf("ディレクトリ検証でエラーが発生: %v", err)
	}

	// 非同期処理の完了を待つ
	verifier.wg.Wait()

	// 結果を確認
	results := verifier.GetResults()
	if len(results) != 1 {
		t.Errorf("期待される結果数: 1, 実際: %d", len(results))
	}

	if len(results) > 0 && results[0].Path != "test.txt" {
		t.Errorf("期待されるパス: test.txt, 実際: %s", results[0].Path)
	}
}

// TestCheckExtraFiles はcheckExtraFilesメソッドのテスト
func TestCheckExtraFiles(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "verifier_test")
	if err != nil {
		t.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("ソースディレクトリの作成に失敗: %v", err)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("宛先ディレクトリの作成に失敗: %v", err)
	}

	// ソースファイルを作成
	sourceFile := filepath.Join(sourceDir, "source.txt")
	if err := os.WriteFile(sourceFile, []byte("source content"), 0644); err != nil {
		t.Fatalf("ソースファイルの作成に失敗: %v", err)
	}

	// 宛先に余分なファイルを作成
	extraFile := filepath.Join(destDir, "extra.txt")
	if err := os.WriteFile(extraFile, []byte("extra content"), 0644); err != nil {
		t.Fatalf("余分なファイルの作成に失敗: %v", err)
	}

	verifier := NewVerifier(sourceDir, destDir, DefaultOptions(), nil, nil)

	// 余分なファイルのチェック
	err = verifier.checkExtraFiles(sourceDir, destDir)
	if err != nil {
		t.Errorf("余分なファイルチェックでエラーが発生: %v", err)
	}

	// 結果を確認
	results := verifier.GetResults()
	if len(results) != 1 {
		t.Errorf("期待される結果数: 1, 実際: %d", len(results))
	}

	if !strings.Contains(results[0].Error.Error(), "余分なファイルが存在します") {
		t.Errorf("期待されるエラーメッセージに含まれるべき文字列: 余分なファイルが存在します, 実際: %s", results[0].Error.Error())
	}
}

// TestGenerateReport はGenerateReportメソッドのテスト
func TestGenerateReport(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "verifier_test")
	if err != nil {
		t.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	verifier := NewVerifier("/source", "/dest", DefaultOptions(), nil, nil)

	// テスト結果を追加
	result := VerificationResult{
		Path:         "test.txt",
		SourceExists: true,
		DestExists:   true,
		SizeMatch:    true,
		HashMatch:    true,
		SourceHash:   "abc123",
		DestHash:     "abc123",
		SourceSize:   100,
		DestSize:     100,
		SourceTime:   time.Now(),
		DestTime:     time.Now(),
	}
	verifier.addResult(result)

	// レポートファイルのパス
	reportPath := filepath.Join(tempDir, "report.csv")

	// レポートの生成
	err = verifier.GenerateReport(reportPath)
	if err != nil {
		t.Errorf("レポート生成でエラーが発生: %v", err)
	}

	// レポートファイルの存在確認
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Error("レポートファイルが作成されていません")
	}

	// レポートファイルの内容確認
	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Errorf("レポートファイルの読み込みに失敗: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "test.txt") {
		t.Error("レポートにテストファイルの情報が含まれていません")
	}

	if !strings.Contains(contentStr, "abc123") {
		t.Error("レポートにハッシュ値が含まれていません")
	}
}

// TestVerifyWithContext はコンテキストキャンセルのテスト
func TestVerifyWithContext(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "verifier_test")
	if err != nil {
		t.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("ソースディレクトリの作成に失敗: %v", err)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("宛先ディレクトリの作成に失敗: %v", err)
	}

	verifier := NewVerifier(sourceDir, destDir, DefaultOptions(), nil, nil)
	verifier.Cancel()

	err = verifier.Verify()
	if err == nil {
		t.Error("キャンセルされたコンテキストで検証を実行した場合、エラーが返されるべきです")
	}
	if err != nil && !strings.Contains(err.Error(), "検証処理がキャンセルされました") {
		t.Errorf("期待されるエラーメッセージに含まれるべき文字列: 検証処理がキャンセルされました, 実際: %s", err.Error())
	}
}

// TestVerifyWithDatabase はデータベース付きの検証テスト
func TestVerifyWithDatabase(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "verifier_test")
	if err != nil {
		t.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	dbPath := filepath.Join(tempDir, "test.db")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("ソースディレクトリの作成に失敗: %v", err)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("宛先ディレクトリの作成に失敗: %v", err)
	}

	// データベースを作成
	syncDB, err := database.NewSyncDB(dbPath, database.NormalSync)
	if err != nil {
		t.Fatalf("データベースの作成に失敗: %v", err)
	}
	defer syncDB.Close()

	// テストファイルを作成
	sourceFile := filepath.Join(sourceDir, "test.txt")
	destFile := filepath.Join(destDir, "test.txt")
	if err := os.WriteFile(sourceFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("ソースファイルの作成に失敗: %v", err)
	}
	if err := os.WriteFile(destFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("宛先ファイルの作成に失敗: %v", err)
	}

	verifier := NewVerifier(sourceDir, destDir, DefaultOptions(), nil, syncDB)

	// 検証を実行
	err = verifier.Verify()
	if err != nil {
		t.Errorf("検証でエラーが発生: %v", err)
	}

	// 非同期処理の完了を待つ
	verifier.wg.Wait()

	// データベースに記録されたファイルを確認
	files, err := syncDB.GetAllFiles()
	if err != nil {
		t.Errorf("データベースからのファイル取得でエラーが発生: %v", err)
	}

	if len(files) == 0 {
		t.Error("データベースにファイルが記録されていません")
	}
}

// TestVerifyWithProgressCallback は進捗コールバック付きの検証テスト
func TestVerifyWithProgressCallback(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "verifier_test")
	if err != nil {
		t.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("ソースディレクトリの作成に失敗: %v", err)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("宛先ディレクトリの作成に失敗: %v", err)
	}

	// テストファイルを複数作成
	for i := 0; i < 10; i++ {
		sourceFile := filepath.Join(sourceDir, fmt.Sprintf("test%d.txt", i))
		destFile := filepath.Join(destDir, fmt.Sprintf("test%d.txt", i))
		if err := os.WriteFile(sourceFile, []byte("test content"), 0644); err != nil {
			t.Fatalf("ソースファイルの作成に失敗: %v", err)
		}
		if err := os.WriteFile(destFile, []byte("test content"), 0644); err != nil {
			t.Fatalf("宛先ファイルの作成に失敗: %v", err)
		}
	}

	options := DefaultOptions()
	options.ProgressInterval = 10 * time.Millisecond
	verifier := NewVerifier(sourceDir, destDir, options, nil, nil)

	var progressCount int32
	callback := func(current, total int64, currentFile string) {
		atomic.AddInt32(&progressCount, 1)
	}
	verifier.SetProgressCallback(callback)

	err = verifier.Verify()
	if err != nil {
		t.Errorf("検証でエラーが発生: %v", err)
	}

	verifier.wg.Wait()
	time.Sleep(100 * time.Millisecond)
	if atomic.LoadInt32(&progressCount) == 0 {
		t.Error("進捗コールバックが呼ばれていません")
	}
}

// ベンチマーク関数
func BenchmarkVerifyFile_Small(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "benchmark")
	if err != nil {
		b.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	// 小さなファイルを作成
	sourceFile := filepath.Join(sourceDir, "test.txt")
	destFile := filepath.Join(destDir, "test.txt")
	content := []byte("hello world")
	os.WriteFile(sourceFile, content, 0644)
	os.WriteFile(destFile, content, 0644)

	options := DefaultOptions()
	verifier := NewVerifier(sourceDir, destDir, options, nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := verifier.verifyFile(sourceFile, destFile)
		if err != nil {
			b.Fatalf("verifyFileが失敗: %v", err)
		}
		if !result.HashMatch {
			b.Fatalf("ファイルが一致しません: %v", result)
		}
	}
}

func BenchmarkVerifyFile_Large(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "benchmark")
	if err != nil {
		b.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	// 大きなファイルを作成（5MB）
	sourceFile := filepath.Join(sourceDir, "large.txt")
	destFile := filepath.Join(destDir, "large.txt")
	content := make([]byte, 5*1024*1024)
	for i := range content {
		content[i] = byte(i % 256)
	}
	os.WriteFile(sourceFile, content, 0644)
	os.WriteFile(destFile, content, 0644)

	options := DefaultOptions()
	verifier := NewVerifier(sourceDir, destDir, options, nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := verifier.verifyFile(sourceFile, destFile)
		if err != nil {
			b.Fatalf("verifyFileが失敗: %v", err)
		}
		if !result.HashMatch {
			b.Fatalf("ファイルが一致しません: %v", result)
		}
	}
}

func BenchmarkVerifyDirectory_Small(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "benchmark")
	if err != nil {
		b.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	// 小さなファイルを複数作成
	for i := 0; i < 50; i++ {
		fileName := fmt.Sprintf("file_%d.txt", i)
		content := []byte(fmt.Sprintf("content %d", i))
		os.WriteFile(filepath.Join(sourceDir, fileName), content, 0644)
		os.WriteFile(filepath.Join(destDir, fileName), content, 0644)
	}

	options := DefaultOptions()
	options.MaxConcurrent = 4
	verifier := NewVerifier(sourceDir, destDir, options, nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := verifier.verifyDirectory(sourceDir, destDir)
		if err != nil {
			b.Fatalf("verifyDirectoryが失敗: %v", err)
		}
	}
}

func BenchmarkVerifyDirectory_Large(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "benchmark")
	if err != nil {
		b.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	// 大きなファイルを複数作成
	for i := 0; i < 10; i++ {
		fileName := fmt.Sprintf("large_%d.txt", i)
		content := make([]byte, 1024*1024) // 1MB
		for j := range content {
			content[j] = byte((i + j) % 256)
		}
		os.WriteFile(filepath.Join(sourceDir, fileName), content, 0644)
		os.WriteFile(filepath.Join(destDir, fileName), content, 0644)
	}

	options := DefaultOptions()
	options.MaxConcurrent = 4
	verifier := NewVerifier(sourceDir, destDir, options, nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := verifier.verifyDirectory(sourceDir, destDir)
		if err != nil {
			b.Fatalf("verifyDirectoryが失敗: %v", err)
		}
	}
}

func BenchmarkVerifyAll_Parallel(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "benchmark")
	if err != nil {
		b.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	// 複数のファイルを作成
	for i := 0; i < 100; i++ {
		fileName := fmt.Sprintf("file_%d.txt", i)
		content := make([]byte, 1024) // 1KB
		for j := range content {
			content[j] = byte((i + j) % 256)
		}
		os.WriteFile(filepath.Join(sourceDir, fileName), content, 0644)
		os.WriteFile(filepath.Join(destDir, fileName), content, 0644)
	}

	options := DefaultOptions()
	options.MaxConcurrent = 8
	options.ProgressInterval = time.Hour // 進捗表示を無効化
	verifier := NewVerifier(sourceDir, destDir, options, nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := verifier.Verify()
		if err != nil {
			b.Fatalf("Verifyが失敗: %v", err)
		}
	}
}

func BenchmarkVerifyAll_WithFilter(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "benchmark")
	if err != nil {
		b.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	// 異なる拡張子のファイルを作成
	for i := 0; i < 200; i++ {
		extensions := []string{".txt", ".log", ".tmp", ".bak", ".dat"}
		ext := extensions[i%len(extensions)]
		fileName := fmt.Sprintf("file_%d%s", i, ext)
		content := []byte(fmt.Sprintf("content %d", i))

		sourceFile := filepath.Join(sourceDir, fileName)
		destFile := filepath.Join(destDir, fileName)

		os.WriteFile(sourceFile, content, 0644)
		os.WriteFile(destFile, content, 0644)
	}

	options := DefaultOptions()
	options.MaxConcurrent = 4
	options.ProgressInterval = time.Hour // 進捗表示を無効化
	fileFilter := filter.NewFilter("*.txt,*.log", "*.tmp,*.bak")
	verifier := NewVerifier(sourceDir, destDir, options, fileFilter, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := verifier.Verify()
		if err != nil {
			b.Fatalf("Verifyが失敗: %v", err)
		}
	}
}

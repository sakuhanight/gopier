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

// TestCheckExtraFiles_EdgeCases はcheckExtraFiles関数のエッジケースをテスト
func TestCheckExtraFiles_EdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	// 宛先にのみ存在するファイルを作成
	extraFile := filepath.Join(destDir, "extra.txt")
	os.WriteFile(extraFile, []byte("extra content"), 0644)

	// サブディレクトリにのみ存在するファイル
	extraSubDir := filepath.Join(destDir, "subdir")
	os.MkdirAll(extraSubDir, 0755)
	extraSubFile := filepath.Join(extraSubDir, "extra_sub.txt")
	os.WriteFile(extraSubFile, []byte("extra sub content"), 0644)

	options := DefaultOptions()
	verifier := NewVerifier(sourceDir, destDir, options, nil, nil)

	// 通常のチェック（IgnoreExtra=false）
	err := verifier.checkExtraFiles(sourceDir, destDir)
	if err != nil {
		t.Errorf("余分なファイルのチェックでエラーが発生しました: %v", err)
	}
	// 結果を確認
	results := verifier.GetResults()
	if len(results) == 0 {
		t.Error("余分なファイルが検出されませんでした")
	}

	// IgnoreExtra=trueの場合
	options.IgnoreExtra = true
	verifier2 := NewVerifier(sourceDir, destDir, options, nil, nil)
	err = verifier2.checkExtraFiles(sourceDir, destDir)
	if err != nil {
		t.Errorf("IgnoreExtra=trueの場合にエラーが発生しました: %v", err)
	}

	// 空のディレクトリ
	emptyDestDir := filepath.Join(tempDir, "empty_dest")
	os.MkdirAll(emptyDestDir, 0755)
	verifier3 := NewVerifier(sourceDir, emptyDestDir, options, nil, nil)
	err = verifier3.checkExtraFiles(sourceDir, emptyDestDir)
	if err != nil {
		t.Errorf("空のディレクトリでエラーが発生しました: %v", err)
	}
}

// TestVerifyFile_EdgeCases はverifyFile関数のエッジケースをテスト
func TestVerifyFile_EdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	// テストファイルを作成
	testFile := filepath.Join(sourceDir, "test.txt")
	testContent := "test content"
	os.WriteFile(testFile, []byte(testContent), 0644)

	// 宛先にも同じファイルを作成
	destFile := filepath.Join(destDir, "test.txt")
	os.WriteFile(destFile, []byte(testContent), 0644)

	options := DefaultOptions()
	verifier := NewVerifier(sourceDir, destDir, options, nil, nil)

	// 正常な検証
	result, err := verifier.verifyFile(testFile, destFile)
	if err != nil {
		t.Errorf("正常な検証が失敗: %v", err)
	}
	if result == nil {
		t.Error("結果がnilです")
	}

	// ソースファイルが存在しない場合
	result, err = verifier.verifyFile(filepath.Join(sourceDir, "nonexistent.txt"), filepath.Join(destDir, "nonexistent.txt"))
	if err != nil {
		t.Errorf("存在しないソースファイルでエラーが発生しました: %v", err)
	}
	if result == nil {
		t.Error("結果がnilです")
	}

	// 宛先ファイルが存在しない場合
	os.Remove(destFile)
	result, err = verifier.verifyFile(testFile, destFile)
	if err != nil {
		t.Errorf("存在しない宛先ファイルでエラーが発生しました: %v", err)
	}
	if result == nil {
		t.Error("結果がnilです")
	}

	// IgnoreMissing=trueの場合
	options.IgnoreMissing = true
	verifier2 := NewVerifier(sourceDir, destDir, options, nil, nil)
	result, err = verifier2.verifyFile(testFile, destFile)
	if err != nil {
		t.Errorf("IgnoreMissing=trueの場合にエラーが発生しました: %v", err)
	}
}

// TestVerifyDirectory_EdgeCases はverifyDirectory関数のエッジケースをテスト
func TestVerifyDirectory_EdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	// 空のディレクトリ
	emptySourceDir := filepath.Join(sourceDir, "empty")
	os.MkdirAll(emptySourceDir, 0755)
	emptyDestDir := filepath.Join(destDir, "empty")
	os.MkdirAll(emptyDestDir, 0755)

	// シンボリックリンクを含むディレクトリ
	symlinkSourceDir := filepath.Join(sourceDir, "symlink")
	os.MkdirAll(symlinkSourceDir, 0755)
	symlinkFile := filepath.Join(symlinkSourceDir, "link.txt")
	os.Symlink(filepath.Join(sourceDir, "nonexistent.txt"), symlinkFile)

	options := DefaultOptions()
	verifier := NewVerifier(sourceDir, destDir, options, nil, nil)

	// 空のディレクトリの検証
	err := verifier.verifyDirectory(emptySourceDir, emptyDestDir)
	if err != nil {
		t.Errorf("空のディレクトリの検証が失敗: %v", err)
	}

	// シンボリックリンクを含むディレクトリの検証
	err = verifier.verifyDirectory(symlinkSourceDir, filepath.Join(destDir, "symlink"))
	if err != nil {
		t.Errorf("シンボリックリンクを含むディレクトリの検証が失敗: %v", err)
	}

	// 存在しないディレクトリ
	err = verifier.verifyDirectory(filepath.Join(sourceDir, "nonexistent"), filepath.Join(destDir, "nonexistent"))
	if err == nil {
		t.Error("存在しないディレクトリでエラーが発生しませんでした")
	}

	// 非再帰モード
	options.Recursive = false
	verifier2 := NewVerifier(sourceDir, destDir, options, nil, nil)
	err = verifier2.verifyDirectory(emptySourceDir, emptyDestDir)
	if err != nil {
		t.Errorf("非再帰モードでの検証が失敗: %v", err)
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

// TestGenerateReport_EdgeCases はGenerateReport関数のエッジケースをテスト
func TestGenerateReport_EdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	options := DefaultOptions()
	verifier := NewVerifier(sourceDir, destDir, options, nil, nil)

	// 結果がない場合のレポート生成
	reportPath := filepath.Join(tempDir, "empty_report.txt")
	err := verifier.GenerateReport(reportPath)
	if err != nil {
		t.Errorf("空の結果でのレポート生成が失敗: %v", err)
	}

	// エラー結果を含むレポート生成
	errorResult := VerificationResult{
		Path:  "error.txt",
		Error: fmt.Errorf("テストエラー"),
	}
	verifier.addResult(errorResult)

	reportPath2 := filepath.Join(tempDir, "error_report.txt")
	err = verifier.GenerateReport(reportPath2)
	if err != nil {
		t.Errorf("エラー結果を含むレポート生成が失敗: %v", err)
	}

	// 無効なパスでのレポート生成
	err = verifier.GenerateReport("/invalid/path/report.txt")
	if err == nil {
		t.Error("無効なパスでエラーが発生しませんでした")
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

// TestVerify_SingleFile は単一ファイル検証のテスト
func TestVerify_SingleFile(t *testing.T) {
	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "source.txt")
	destDir := filepath.Join(tempDir, "dest")
	destFile := filepath.Join(destDir, "source.txt")

	// テストファイルを作成
	os.WriteFile(sourceFile, []byte("test content"), 0644)
	os.MkdirAll(destDir, 0755)
	os.WriteFile(destFile, []byte("test content"), 0644)

	options := DefaultOptions()
	verifier := NewVerifier(sourceFile, destDir, options, nil, nil)

	// 単一ファイルの検証を直接テスト
	result, err := verifier.verifyFile(sourceFile, destFile)
	if err != nil {
		t.Errorf("単一ファイル検証が失敗: %v", err)
	}

	if result == nil {
		t.Fatal("結果がnilです")
	}

	if !result.HashMatch {
		t.Error("ハッシュが一致すべきです")
	}
}

// TestVerifyFile_SourceNotExists はソースファイルが存在しない場合のテスト
func TestVerifyFile_SourceNotExists(t *testing.T) {
	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "nonexistent.txt")
	destFile := filepath.Join(tempDir, "dest.txt")

	options := DefaultOptions()
	verifier := NewVerifier("/source", "/dest", options, nil, nil)

	result, err := verifier.verifyFile(sourceFile, destFile)
	if err != nil {
		t.Errorf("エラーが発生すべきではありません: %v", err)
	}

	if result == nil {
		t.Fatal("結果がnilです")
	}

	if result.SourceExists {
		t.Error("ソースファイルは存在しないと判定されるべきです")
	}

	if result.Error == nil {
		t.Error("エラーが設定されるべきです")
	}
}

// TestVerifyFile_DestNotExists は宛先ファイルが存在しない場合のテスト
func TestVerifyFile_DestNotExists(t *testing.T) {
	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "source.txt")
	destFile := filepath.Join(tempDir, "nonexistent.txt")

	// ソースファイルのみ作成
	os.WriteFile(sourceFile, []byte("test content"), 0644)

	options := DefaultOptions()
	verifier := NewVerifier(tempDir, "/dest", options, nil, nil)

	result, err := verifier.verifyFile(sourceFile, destFile)
	if err != nil {
		t.Errorf("エラーが発生すべきではありません: %v", err)
	}

	if result == nil {
		t.Fatal("結果がnilです")
	}

	if result.DestExists {
		t.Error("宛先ファイルは存在しないと判定されるべきです")
	}

	if result.Error == nil {
		t.Error("エラーが設定されるべきです")
	}
}

// TestVerifyFile_DestNotExistsWithIgnoreMissing はIgnoreMissingオプションのテスト
func TestVerifyFile_DestNotExistsWithIgnoreMissing(t *testing.T) {
	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "source.txt")
	destFile := filepath.Join(tempDir, "nonexistent.txt")

	// ソースファイルのみ作成
	os.WriteFile(sourceFile, []byte("test content"), 0644)

	options := DefaultOptions()
	options.IgnoreMissing = true
	verifier := NewVerifier(tempDir, "/dest", options, nil, nil)

	result, err := verifier.verifyFile(sourceFile, destFile)
	if err != nil {
		t.Errorf("エラーが発生すべきではありません: %v", err)
	}

	if result != nil {
		t.Error("結果はnilであるべきです（IgnoreMissingの場合）")
	}
}

// TestVerifyFile_SizeMismatch はサイズ不一致のテスト
func TestVerifyFile_SizeMismatch(t *testing.T) {
	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "source.txt")
	destFile := filepath.Join(tempDir, "dest.txt")

	// 異なるサイズのファイルを作成
	os.WriteFile(sourceFile, []byte("test content"), 0644)
	os.WriteFile(destFile, []byte("different content"), 0644)

	options := DefaultOptions()
	verifier := NewVerifier(tempDir, "/dest", options, nil, nil)

	result, err := verifier.verifyFile(sourceFile, destFile)
	if err != nil {
		t.Errorf("エラーが発生すべきではありません: %v", err)
	}

	if result == nil {
		t.Fatal("結果がnilです")
	}

	if result.SizeMatch {
		t.Error("サイズは一致しないと判定されるべきです")
	}

	if result.Error == nil {
		t.Error("エラーが設定されるべきです")
	}
}

// TestVerifyFile_HashMismatch はハッシュ不一致のテスト
func TestVerifyFile_HashMismatch(t *testing.T) {
	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "source.txt")
	destFile := filepath.Join(tempDir, "dest.txt")

	// 同じサイズだが異なる内容のファイルを作成
	os.WriteFile(sourceFile, []byte("test content 1"), 0644)
	os.WriteFile(destFile, []byte("test content 2"), 0644)

	options := DefaultOptions()
	verifier := NewVerifier(tempDir, "/dest", options, nil, nil)

	result, err := verifier.verifyFile(sourceFile, destFile)
	if err != nil {
		t.Errorf("エラーが発生すべきではありません: %v", err)
	}

	if result == nil {
		t.Fatal("結果がnilです")
	}

	if result.HashMatch {
		t.Error("ハッシュは一致しないと判定されるべきです")
	}

	if result.Error == nil {
		t.Error("エラーが設定されるべきです")
	}
}

// TestVerifyFile_WithDatabase はデータベース連携のテスト
func TestVerifyFile_WithDatabase(t *testing.T) {
	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "source.txt")
	destFile := filepath.Join(tempDir, "dest.txt")

	// テストファイルを作成
	os.WriteFile(sourceFile, []byte("test content"), 0644)
	os.WriteFile(destFile, []byte("test content"), 0644)

	// データベースを作成
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := database.NewSyncDB(dbPath, database.NormalSync)
	if err != nil {
		t.Fatalf("データベース作成エラー: %v", err)
	}
	defer os.Remove(dbPath)

	options := DefaultOptions()
	verifier := NewVerifier(tempDir, "/dest", options, nil, db)

	result, err := verifier.verifyFile(sourceFile, destFile)
	if err != nil {
		t.Errorf("エラーが発生すべきではありません: %v", err)
	}

	if result == nil {
		t.Fatal("結果がnilです")
	}

	if !result.HashMatch {
		t.Error("ハッシュが一致すべきです")
	}
}

// TestCheckExtraFiles_Recursive は再帰的な余分ファイルチェックのテスト
func TestCheckExtraFiles_Recursive(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")

	// ソースディレクトリ構造を作成
	os.MkdirAll(filepath.Join(sourceDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(sourceDir, "file1.txt"), []byte("content1"), 0644)
	os.WriteFile(filepath.Join(sourceDir, "subdir", "file2.txt"), []byte("content2"), 0644)

	// 宛先ディレクトリ構造を作成（余分なファイルを含む）
	os.MkdirAll(filepath.Join(destDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(destDir, "file1.txt"), []byte("content1"), 0644)
	os.WriteFile(filepath.Join(destDir, "extra.txt"), []byte("extra"), 0644) // 余分なファイル
	os.WriteFile(filepath.Join(destDir, "subdir", "file2.txt"), []byte("content2"), 0644)
	os.WriteFile(filepath.Join(destDir, "subdir", "extra2.txt"), []byte("extra2"), 0644) // 余分なファイル

	options := DefaultOptions()
	options.Recursive = true
	verifier := NewVerifier(sourceDir, destDir, options, nil, nil)

	err := verifier.checkExtraFiles(sourceDir, destDir)
	if err != nil {
		t.Errorf("余分ファイルチェックが失敗: %v", err)
	}

	results := verifier.GetResults()
	if len(results) < 2 {
		t.Errorf("期待される結果数: 2以上, 実際: %d", len(results))
	}

	// 余分なファイルが検出されているか確認
	extraFound := false
	for _, result := range results {
		if strings.Contains(result.Path, "extra") {
			extraFound = true
			break
		}
	}

	if !extraFound {
		t.Error("余分なファイルが検出されていません")
	}
}

// TestCheckExtraFiles_WithFilter はフィルタ付き余分ファイルチェックのテスト
func TestCheckExtraFiles_WithFilter(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")

	// ソースディレクトリを作成
	os.MkdirAll(sourceDir, 0755)

	// 宛先ディレクトリに余分なファイルを作成
	os.MkdirAll(destDir, 0755)
	os.WriteFile(filepath.Join(destDir, "extra.txt"), []byte("extra"), 0644) // フィルタに一致
	os.WriteFile(filepath.Join(destDir, "extra.log"), []byte("extra"), 0644) // フィルタに一致しない

	options := DefaultOptions()
	fileFilter := filter.NewFilter("*.txt", "") // .txtファイルのみ含める
	verifier := NewVerifier(sourceDir, destDir, options, fileFilter, nil)

	err := verifier.checkExtraFiles(sourceDir, destDir)
	if err != nil {
		t.Errorf("余分ファイルチェックが失敗: %v", err)
	}

	results := verifier.GetResults()
	if len(results) != 1 {
		t.Errorf("期待される結果数: 1, 実際: %d", len(results))
	}

	if !strings.Contains(results[0].Path, "extra.txt") {
		t.Error("フィルタに一致する余分ファイルが検出されていません")
	}
}

// TestCheckExtraFiles_ExtraDirectory は余分なディレクトリのテスト
func TestCheckExtraFiles_ExtraDirectory(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")

	// ソースディレクトリを作成
	os.MkdirAll(sourceDir, 0755)

	// 宛先ディレクトリに余分なディレクトリを作成
	os.MkdirAll(filepath.Join(destDir, "extra_dir"), 0755)
	os.WriteFile(filepath.Join(destDir, "extra_dir", "file.txt"), []byte("content"), 0644)

	options := DefaultOptions()
	options.Recursive = true
	verifier := NewVerifier(sourceDir, destDir, options, nil, nil)

	err := verifier.checkExtraFiles(sourceDir, destDir)
	if err != nil {
		t.Errorf("余分ディレクトリチェックが失敗: %v", err)
	}

	results := verifier.GetResults()
	if len(results) < 1 {
		t.Error("余分なディレクトリが検出されていません")
	}
}

// TestReportProgress_ChannelClosed は進捗報告のチャンネル閉じられケースのテスト
func TestReportProgress_ChannelClosed(t *testing.T) {
	options := DefaultOptions()
	options.ProgressInterval = time.Millisecond * 10
	verifier := NewVerifier("/source", "/dest", options, nil, nil)

	// 進捗コールバックを設定
	verifier.SetProgressCallback(func(current, total int64, currentFile string) {
		// 何もしない
	})

	// 進捗チャンネルを閉じる
	close(verifier.progressChan)

	// 進捗報告を開始
	go verifier.reportProgress()

	// 少し待ってからキャンセル
	time.Sleep(time.Millisecond * 50)
	verifier.Cancel()

	// 進捗報告が正常に終了することを確認
	verifier.wg.Wait()
}

// TestGenerateReport_WriteError はレポート書き込みエラーのテスト
func TestGenerateReport_WriteError(t *testing.T) {
	tempDir := t.TempDir()

	// 書き込み権限のないディレクトリを作成
	readOnlyDir := filepath.Join(tempDir, "readonly")
	os.MkdirAll(readOnlyDir, 0444) // 読み取り専用
	reportPath := filepath.Join(readOnlyDir, "report.csv")

	verifier := NewVerifier("/source", "/dest", DefaultOptions(), nil, nil)

	// 結果を追加
	result := VerificationResult{
		Path:         "test.txt",
		SourceExists: true,
		DestExists:   true,
		SizeMatch:    true,
		HashMatch:    true,
	}
	verifier.addResult(result)

	err := verifier.GenerateReport(reportPath)
	if err == nil {
		t.Error("書き込み権限がない場合、エラーが発生すべきです")
		return
	}
	if !strings.Contains(err.Error(), "レポートファイル作成エラー") {
		t.Errorf("期待されるエラーメッセージに'レポートファイル作成エラー'が含まれていません: %v", err)
	}
}

// TestVerify_ContextCancel はコンテキストキャンセルのテスト
func TestVerify_ContextCancel(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")

	// テストディレクトリを作成
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)
	os.WriteFile(filepath.Join(sourceDir, "test.txt"), []byte("content"), 0644)
	os.WriteFile(filepath.Join(destDir, "test.txt"), []byte("content"), 0644)

	options := DefaultOptions()
	verifier := NewVerifier(sourceDir, destDir, options, nil, nil)

	// 検証を開始する前にキャンセル
	verifier.Cancel()

	err := verifier.Verify()
	if err == nil {
		t.Error("キャンセルされた場合、エラーが発生すべきです")
		return
	}
	if !strings.Contains(err.Error(), "キャンセル") {
		t.Errorf("期待されるエラーメッセージに'キャンセル'が含まれていません: %v", err)
	}
}

// TestVerifyFile_ProgressChannelFull は進捗チャンネルが一杯の場合のテスト
func TestVerifyFile_ProgressChannelFull(t *testing.T) {
	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "source.txt")
	destFile := filepath.Join(tempDir, "dest.txt")

	// テストファイルを作成
	os.WriteFile(sourceFile, []byte("test content"), 0644)
	os.WriteFile(destFile, []byte("test content"), 0644)

	options := DefaultOptions()
	verifier := NewVerifier(tempDir, "/dest", options, nil, nil)

	// 進捗チャンネルを一杯にする
	for i := 0; i < 100; i++ {
		select {
		case verifier.progressChan <- "test":
		default:
			// チャンネルが一杯になったら停止
			break
		}
	}

	// 進捗コールバックを設定
	verifier.SetProgressCallback(func(current, total int64, currentFile string) {
		// 何もしない
	})

	result, err := verifier.verifyFile(sourceFile, destFile)
	if err != nil {
		t.Errorf("エラーが発生すべきではありません: %v", err)
	}

	if result == nil {
		t.Fatal("結果がnilです")
	}

	// 進捗チャンネルが一杯でも検証は正常に動作することを確認
	if !result.HashMatch {
		t.Error("ハッシュが一致すべきです")
	}
}

// TestVerifyFile_FileInfoError はファイル情報取得エラーのテスト
func TestVerifyFile_FileInfoError(t *testing.T) {
	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "source.txt")
	destFile := filepath.Join(tempDir, "dest.txt")

	// テストファイルを作成
	os.WriteFile(sourceFile, []byte("test content"), 0644)
	os.WriteFile(destFile, []byte("test content"), 0644)

	options := DefaultOptions()
	verifier := NewVerifier(tempDir, "/dest", options, nil, nil)

	// ファイルを削除してファイル情報取得エラーを発生させる
	os.Remove(sourceFile)

	result, err := verifier.verifyFile(sourceFile, destFile)
	if err != nil {
		t.Errorf("エラーが発生すべきではありません: %v", err)
	}

	if result == nil {
		t.Fatal("結果がnilです")
	}

	if result.SourceExists {
		t.Error("ソースファイルは存在しないと判定されるべきです")
	}

	if result.Error == nil {
		t.Error("エラーが設定されるべきです")
	}
}

// TestVerifyDirectory_ReadDirError はディレクトリ読み込みエラーのテスト
func TestVerifyDirectory_ReadDirError(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")

	// ソースディレクトリを作成
	os.MkdirAll(sourceDir, 0755)

	options := DefaultOptions()
	verifier := NewVerifier(sourceDir, destDir, options, nil, nil)

	// ソースディレクトリを削除して読み込みエラーを発生させる
	os.RemoveAll(sourceDir)

	err := verifier.verifyDirectory(sourceDir, destDir)
	if err == nil {
		t.Error("ディレクトリ読み込みエラーが発生すべきです")
	}

	if !strings.Contains(err.Error(), "ディレクトリ読み込みエラー") {
		t.Errorf("期待されるエラーメッセージに'ディレクトリ読み込みエラー'が含まれていません: %v", err)
	}
}

// TestCheckExtraFiles_ReadDirError は余分ファイルチェックのディレクトリ読み込みエラーテスト
func TestCheckExtraFiles_ReadDirError(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")

	// ソースディレクトリを作成
	os.MkdirAll(sourceDir, 0755)

	options := DefaultOptions()
	verifier := NewVerifier(sourceDir, destDir, options, nil, nil)

	// 宛先ディレクトリを削除して読み込みエラーを発生させる
	os.RemoveAll(destDir)

	err := verifier.checkExtraFiles(sourceDir, destDir)
	if err == nil {
		t.Error("ディレクトリ読み込みエラーが発生すべきです")
	}

	if !strings.Contains(err.Error(), "宛先ディレクトリ読み込みエラー") {
		t.Errorf("期待されるエラーメッセージに'宛先ディレクトリ読み込みエラー'が含まれていません: %v", err)
	}
}

// TestGenerateReport_HeaderWriteError はレポートヘッダー書き込みエラーのテスト
func TestGenerateReport_HeaderWriteError(t *testing.T) {
	tempDir := t.TempDir()
	reportPath := filepath.Join(tempDir, "report.csv")

	verifier := NewVerifier("/source", "/dest", DefaultOptions(), nil, nil)

	// 結果を追加
	result := VerificationResult{
		Path:         "test.txt",
		SourceExists: true,
		DestExists:   true,
		SizeMatch:    true,
		HashMatch:    true,
	}
	verifier.addResult(result)

	// ファイルを作成して書き込み権限を削除
	file, _ := os.Create(reportPath)
	file.Close()
	os.Chmod(reportPath, 0444) // 読み取り専用

	err := verifier.GenerateReport(reportPath)
	if err == nil {
		t.Error("書き込み権限がない場合、エラーが発生すべきです")
	}

	// 権限を元に戻す
	os.Chmod(reportPath, 0644)
}

// TestVerify_SessionError は同期セッションエラーのテスト
func TestVerify_SessionError(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")

	// テストディレクトリを作成
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)
	os.WriteFile(filepath.Join(sourceDir, "test.txt"), []byte("content"), 0644)
	os.WriteFile(filepath.Join(destDir, "test.txt"), []byte("content"), 0644)

	// データベースファイルを作成して書き込み権限を削除
	dbPath := filepath.Join(tempDir, "test.db")
	file, err := os.Create(dbPath)
	if err != nil {
		t.Fatal("データベースファイルの作成に失敗しました")
	}
	file.Close()
	os.Chmod(dbPath, 0444) // 読み取り専用

	// データベースを作成（エラーが発生するはず）
	db, err := database.NewSyncDB(dbPath, database.NormalSync)
	if err != nil {
		// データベース作成エラーが発生した場合、テストをスキップ
		t.Skip("データベース作成エラーが発生したため、テストをスキップします")
	}

	options := DefaultOptions()
	verifier := NewVerifier(sourceDir, destDir, options, nil, db)

	err = verifier.Verify()
	if err == nil {
		t.Error("無効なデータベースパスの場合、エラーが発生すべきです")
		return
	}
	if !strings.Contains(err.Error(), "同期セッション開始エラー") {
		t.Errorf("期待されるエラーメッセージに'同期セッション開始エラー'が含まれていません: %v", err)
	}

	// 権限を元に戻す
	os.Chmod(dbPath, 0644)
}

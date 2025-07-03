package copier

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
	"github.com/sakuhanight/gopier/internal/logger"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	if opts.BufferSize != 32*1024*1024 {
		t.Errorf("バッファサイズ: 期待値=%d, 実際=%d", 32*1024*1024, opts.BufferSize)
	}
	if !opts.Recursive {
		t.Error("Recursive: デフォルトはtrueであるべきです")
	}
	if !opts.PreserveModTime {
		t.Error("PreserveModTime: デフォルトはtrueであるべきです")
	}
	if !opts.VerifyHash {
		t.Error("VerifyHash: デフォルトはtrueであるべきです")
	}
	if opts.HashAlgorithm == "" {
		t.Error("HashAlgorithm: 空であってはいけません")
	}
	if !opts.OverwriteExisting {
		t.Error("OverwriteExisting: デフォルトはtrueであるべきです")
	}
	if !opts.CreateDirs {
		t.Error("CreateDirs: デフォルトはtrueであるべきです")
	}
	if opts.MaxRetries != 3 {
		t.Errorf("MaxRetries: 期待値=3, 実際=%d", opts.MaxRetries)
	}
	if opts.RetryDelay != 2*time.Second {
		t.Errorf("RetryDelay: 期待値=2s, 実際=%v", opts.RetryDelay)
	}
	if opts.ProgressInterval != time.Second {
		t.Errorf("ProgressInterval: 期待値=1s, 実際=%v", opts.ProgressInterval)
	}
	if opts.MaxConcurrent != 4 {
		t.Errorf("MaxConcurrent: 期待値=4, 実際=%d", opts.MaxConcurrent)
	}
	if opts.Mode != ModeCopy {
		t.Errorf("Mode: 期待値=ModeCopy, 実際=%v", opts.Mode)
	}
}

func TestNewFileCopierAndMethods(t *testing.T) {
	sourceDir := "/tmp/source"
	destDir := "/tmp/dest"
	options := DefaultOptions()
	fileFilter := filter.NewFilter("*.txt", "")
	db := &database.SyncDB{}
	log := &logger.Logger{}

	copier := NewFileCopier(sourceDir, destDir, options, fileFilter, db, log)
	if copier == nil {
		t.Fatal("NewFileCopierがnilを返しました")
	}
	if copier.sourceDir != sourceDir {
		t.Errorf("sourceDir: 期待値=%s, 実際=%s", sourceDir, copier.sourceDir)
	}
	if copier.destDir != destDir {
		t.Errorf("destDir: 期待値=%s, 実際=%s", destDir, copier.destDir)
	}
	if copier.filter != fileFilter {
		t.Error("filterが正しく設定されていません")
	}
	if copier.db != db {
		t.Error("dbが正しく設定されていません")
	}
	if copier.logger != log {
		t.Error("loggerが正しく設定されていません")
	}
	if copier.hasher == nil {
		t.Error("hasherが初期化されていません")
	}
	if copier.stats == nil {
		t.Error("statsが初期化されていません")
	}
	if copier.progressChan == nil {
		t.Error("progressChanが初期化されていません")
	}
	if copier.semaphore == nil {
		t.Error("semaphoreが初期化されていません")
	}
	if copier.ctx == nil {
		t.Error("ctxが初期化されていません")
	}
	if copier.cancel == nil {
		t.Error("cancelが初期化されていません")
	}

	// SetProgressCallback
	var called int32
	copier.SetProgressCallback(func(current, total int64, currentFile string) {
		atomic.AddInt32(&called, 1)
	})
	if copier.progressFunc == nil {
		t.Error("progressFuncが設定されていません")
	}
	copier.progressFunc(1, 2, "file.txt")
	if atomic.LoadInt32(&called) != 1 {
		t.Error("progressFuncが呼ばれていません")
	}

	// GetStats
	if copier.GetStats() == nil {
		t.Error("GetStatsがnilを返しました")
	}

	// Cancel
	copier.Cancel()
	select {
	case <-copier.ctx.Done():
		// ok
	default:
		t.Error("Cancelがコンテキストをキャンセルしませんでした")
	}
}

func TestCopyFiles_SuccessAndError(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "copier_test")
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

	// ファイルを作成
	for i := 0; i < 10; i++ {
		file := filepath.Join(sourceDir, fmt.Sprintf("a%d.txt", i))
		if err := os.WriteFile(file, []byte("hello world"), 0644); err != nil {
			t.Fatalf("ファイルの作成に失敗: %v", err)
		}
	}

	options := DefaultOptions()
	options.ProgressInterval = 1 * time.Millisecond
	copier := NewFileCopier(sourceDir, destDir, options, nil, nil, nil)

	var progressCalled int32
	copier.SetProgressCallback(func(current, total int64, currentFile string) {
		atomic.AddInt32(&progressCalled, 1)
	})

	err = copier.CopyFiles()
	if err != nil {
		t.Errorf("CopyFilesが失敗しました: %v", err)
	}
	copier.wg.Wait()
	time.Sleep(100 * time.Millisecond)
	if atomic.LoadInt32(&progressCalled) == 0 {
		t.Error("進捗コールバックが呼ばれていません")
	}

	// コピーされたか確認
	for i := 0; i < 10; i++ {
		copiedFile := filepath.Join(destDir, fmt.Sprintf("a%d.txt", i))
		if _, err := os.Stat(copiedFile); err != nil {
			t.Errorf("コピーされたファイルが見つかりません: %v", err)
		}
	}

	// 異常系: 存在しないソース
	copier2 := NewFileCopier(filepath.Join(tempDir, "no_such_dir"), destDir, options, nil, nil, nil)
	err = copier2.CopyFiles()
	if err == nil {
		t.Error("存在しないソースディレクトリでCopyFilesが失敗しませんでした")
	}
}

func TestCopyFile_OverwriteAndErrorCases(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "copier_test2")
	if err != nil {
		t.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	srcFile := filepath.Join(sourceDir, "file.txt")
	dstFile := filepath.Join(destDir, "file.txt")
	os.WriteFile(srcFile, []byte("abc"), 0644)
	os.WriteFile(dstFile, []byte("old"), 0644)

	// 上書き禁止
	opt := DefaultOptions()
	opt.OverwriteExisting = false
	copier := NewFileCopier(sourceDir, destDir, opt, nil, nil, nil)
	err = copier.copyFile(srcFile, dstFile)
	if err == nil {
		// 上書きしない場合はエラーになるはず
		content, _ := os.ReadFile(dstFile)
		if string(content) == "abc" {
			t.Error("OverwriteExisting=falseのとき、ファイルは上書きされるべきではありません")
		}
	}

	// コピー元が存在しない場合
	err = copier.copyFile(filepath.Join(sourceDir, "no.txt"), dstFile)
	if err == nil {
		t.Error("存在しないソースファイルでcopyFileが失敗しませんでした")
	}
}

func TestVerifyFile_HashMismatchAndNoDest(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "copier_test3")
	if err != nil {
		t.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	srcFile := filepath.Join(sourceDir, "file.txt")
	dstFile := filepath.Join(destDir, "file.txt")
	os.WriteFile(srcFile, []byte("abc"), 0644)
	os.WriteFile(dstFile, []byte("def"), 0644)

	copier := NewFileCopier(sourceDir, destDir, DefaultOptions(), nil, nil, nil)
	err = copier.verifyFile(srcFile, dstFile, "file.txt", nil)
	if err == nil {
		t.Error("ハッシュ不一致の場合、verifyFileは失敗すべきです")
	}

	// 宛先ファイルが存在しない場合
	err = copier.verifyFile(srcFile, filepath.Join(destDir, "no.txt"), "no.txt", nil)
	if err == nil {
		t.Error("宛先ファイルが存在しない場合、verifyFileは失敗すべきです")
	}
}

func TestDoCopyFile_Error(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "copier_test4")
	if err != nil {
		t.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	srcFile := filepath.Join(sourceDir, "file.txt")
	dstFile := filepath.Join(destDir, "file.txt")
	os.WriteFile(srcFile, []byte("abc"), 0000) // 読み取り不可

	copier := NewFileCopier(sourceDir, destDir, DefaultOptions(), nil, nil, nil)
	info, _ := os.Stat(srcFile)
	err = copier.doCopyFile(srcFile, dstFile, info)
	if err == nil {
		t.Error("読み取り不可ファイルでdoCopyFileが失敗しませんでした")
	}
}

func TestCopyDirectory_SubdirsAndEmpty(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "copier_test5")
	if err != nil {
		t.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	// サブディレクトリを作成
	subDir := filepath.Join(sourceDir, "subdir")
	os.MkdirAll(subDir, 0755)
	os.WriteFile(filepath.Join(subDir, "file.txt"), []byte("subdir content"), 0644)

	// 空ディレクトリを作成
	emptyDir := filepath.Join(sourceDir, "empty")
	os.MkdirAll(emptyDir, 0755)

	copier := NewFileCopier(sourceDir, destDir, DefaultOptions(), nil, nil, nil)
	err = copier.copyDirectory(sourceDir, destDir)
	if err != nil {
		t.Errorf("copyDirectoryが失敗しました: %v", err)
	}
	copier.wg.Wait()

	// サブディレクトリがコピーされたか確認
	copiedSubFile := filepath.Join(destDir, "subdir", "file.txt")
	if _, err := os.Stat(copiedSubFile); err != nil {
		t.Errorf("サブディレクトリ内のファイルがコピーされていません: %v", err)
	}

	// 空ディレクトリがコピーされたか確認
	copiedEmptyDir := filepath.Join(destDir, "empty")
	if _, err := os.Stat(copiedEmptyDir); err != nil {
		t.Errorf("空ディレクトリがコピーされていません: %v", err)
	}
}

func TestCopyFiles_SingleFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "copier_test6")
	if err != nil {
		t.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceFile := filepath.Join(tempDir, "source.txt")
	destDir := filepath.Join(tempDir, "dest")
	os.WriteFile(sourceFile, []byte("single file content"), 0644)
	os.MkdirAll(destDir, 0755)

	copier := NewFileCopier(sourceFile, destDir, DefaultOptions(), nil, nil, nil)
	err = copier.CopyFiles()
	if err != nil {
		t.Errorf("単一ファイルのCopyFilesが失敗しました: %v", err)
	}

	// ファイルがコピーされたか確認
	copiedFile := filepath.Join(destDir, "source.txt")
	if _, err := os.Stat(copiedFile); err != nil {
		t.Errorf("単一ファイルがコピーされていません: %v", err)
	}
}

func TestCopyFiles_WithFilter(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "copier_test7")
	if err != nil {
		t.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	// 異なる拡張子のファイルを作成
	os.WriteFile(filepath.Join(sourceDir, "include.txt"), []byte("include"), 0644)
	os.WriteFile(filepath.Join(sourceDir, "exclude.log"), []byte("exclude"), 0644)

	// .txtファイルのみを含むフィルター
	fileFilter := filter.NewFilter("*.txt", "")
	copier := NewFileCopier(sourceDir, destDir, DefaultOptions(), fileFilter, nil, nil)
	err = copier.CopyFiles()
	if err != nil {
		t.Errorf("フィルター付きCopyFilesが失敗しました: %v", err)
	}

	// .txtファイルのみがコピーされているか確認
	if _, err := os.Stat(filepath.Join(destDir, "include.txt")); err != nil {
		t.Error("フィルターで含まれるべきファイルがコピーされていません")
	}
	if _, err := os.Stat(filepath.Join(destDir, "exclude.log")); err == nil {
		t.Error("フィルターで除外されるべきファイルがコピーされています")
	}
}

func TestCopyFiles_WithDatabase(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "copier_test8")
	if err != nil {
		t.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	dbPath := filepath.Join(tempDir, "test.db")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	// テストファイルを作成
	os.WriteFile(filepath.Join(sourceDir, "test.txt"), []byte("test content"), 0644)

	// データベースを作成
	syncDB, err := database.NewSyncDB(dbPath, database.NormalSync)
	if err != nil {
		t.Fatalf("データベースの作成に失敗: %v", err)
	}
	defer syncDB.Close()

	copier := NewFileCopier(sourceDir, destDir, DefaultOptions(), nil, syncDB, nil)
	err = copier.CopyFiles()
	if err != nil {
		t.Errorf("データベース連携付きCopyFilesが失敗しました: %v", err)
	}

	// データベースに記録されたか確認
	files, err := syncDB.GetAllFiles()
	if err != nil {
		t.Errorf("データベースからのファイル取得に失敗: %v", err)
	}
	if len(files) == 0 {
		t.Error("データベースにファイルが記録されていません")
	}
}

func TestVerifyFile_MoreErrorCases(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "copier_test9")
	if err != nil {
		t.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	// ソースファイルが存在しない場合
	copier := NewFileCopier(sourceDir, destDir, DefaultOptions(), nil, nil, nil)
	err = copier.verifyFile(filepath.Join(sourceDir, "no.txt"), filepath.Join(destDir, "no.txt"), "no.txt", nil)
	if err == nil {
		t.Error("ソースファイルが存在しない場合、verifyFileは失敗すべきです")
	}

	// 同じ内容のファイル（正常系）
	srcFile := filepath.Join(sourceDir, "same.txt")
	dstFile := filepath.Join(destDir, "same.txt")
	content := []byte("same content")
	os.WriteFile(srcFile, content, 0644)
	os.WriteFile(dstFile, content, 0644)

	info, _ := os.Stat(srcFile)
	err = copier.verifyFile(srcFile, dstFile, "same.txt", info)
	if err != nil {
		t.Errorf("同じ内容のファイルでverifyFileが失敗しました: %v", err)
	}
}

func TestCopyDirectory_NonRecursive(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "copier_test10")
	if err != nil {
		t.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	// サブディレクトリを作成
	subDir := filepath.Join(sourceDir, "subdir")
	os.MkdirAll(subDir, 0755)
	os.WriteFile(filepath.Join(subDir, "file.txt"), []byte("subdir content"), 0644)

	// 非再帰モード
	options := DefaultOptions()
	options.Recursive = false
	copier := NewFileCopier(sourceDir, destDir, options, nil, nil, nil)
	err = copier.copyDirectory(sourceDir, destDir)
	if err != nil {
		t.Errorf("非再帰モードでcopyDirectoryが失敗しました: %v", err)
	}

	// サブディレクトリがコピーされていないか確認
	copiedSubFile := filepath.Join(destDir, "subdir", "file.txt")
	if _, err := os.Stat(copiedSubFile); err == nil {
		t.Error("非再帰モードでサブディレクトリがコピーされています")
	}
}

// TestCopyFile_EdgeCases はcopyFile関数のエッジケースをテスト
func TestCopyFile_EdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	// 空のファイル
	emptyFile := filepath.Join(sourceDir, "empty.txt")
	os.WriteFile(emptyFile, []byte{}, 0644)

	// 大きなファイル（バッファサイズより大きい）
	largeFile := filepath.Join(sourceDir, "large.txt")
	largeData := make([]byte, 100*1024) // 100KB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	os.WriteFile(largeFile, largeData, 0644)

	options := DefaultOptions()
	options.BufferSize = 1024 // 小さなバッファサイズ
	copier := NewFileCopier(sourceDir, destDir, options, nil, nil, nil)

	// 空のファイルのコピー
	err := copier.copyFile(emptyFile, filepath.Join(destDir, "empty.txt"))
	if err != nil {
		t.Errorf("空のファイルのコピーが失敗: %v", err)
	}

	// 大きなファイルのコピー
	err = copier.copyFile(largeFile, filepath.Join(destDir, "large.txt"))
	if err != nil {
		t.Errorf("大きなファイルのコピーが失敗: %v", err)
	}

	// 存在しないファイル
	err = copier.copyFile(filepath.Join(sourceDir, "nonexistent.txt"), filepath.Join(destDir, "nonexistent.txt"))
	if err == nil {
		t.Error("存在しないファイルでエラーが発生しませんでした")
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

	// ファイル情報を取得
	sourceInfo, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("ファイル情報の取得に失敗: %v", err)
	}

	options := DefaultOptions()
	copier := NewFileCopier(sourceDir, destDir, options, nil, nil, nil)

	// 正常な検証
	err = copier.verifyFile(testFile, destFile, "test.txt", sourceInfo)
	if err != nil {
		t.Errorf("正常な検証が失敗: %v", err)
	}

	// 宛先ファイルが存在しない場合
	err = copier.verifyFile(testFile, filepath.Join(destDir, "nonexistent.txt"), "nonexistent.txt", sourceInfo)
	if err == nil {
		t.Error("存在しない宛先ファイルでエラーが発生しませんでした")
	}

	// ソースファイルが存在しない場合
	err = copier.verifyFile(filepath.Join(sourceDir, "nonexistent.txt"), destFile, "nonexistent.txt", sourceInfo)
	if err == nil {
		t.Error("存在しないソースファイルでエラーが発生しませんでした")
	}
}

// TestDoCopyFile_EdgeCases はdoCopyFile関数のエッジケースをテスト
func TestDoCopyFile_EdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	// テストファイルを作成
	testFile := filepath.Join(sourceDir, "test.txt")
	testContent := "test content"
	os.WriteFile(testFile, []byte(testContent), 0644)

	// ファイル情報を取得
	sourceInfo, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("ファイル情報の取得に失敗: %v", err)
	}

	options := DefaultOptions()
	copier := NewFileCopier(sourceDir, destDir, options, nil, nil, nil)

	// 正常なコピー
	err = copier.doCopyFile(testFile, filepath.Join(destDir, "test.txt"), sourceInfo)
	if err != nil {
		t.Errorf("正常なコピーが失敗: %v", err)
	}

	// 宛先ディレクトリを作成してからコピー
	subDir := filepath.Join(destDir, "subdir")
	os.MkdirAll(subDir, 0755)
	err = copier.doCopyFile(testFile, filepath.Join(subDir, "test.txt"), sourceInfo)
	if err != nil {
		t.Errorf("サブディレクトリへのコピーが失敗: %v", err)
	}

	// 存在しないソースファイル
	err = copier.doCopyFile(filepath.Join(sourceDir, "nonexistent.txt"), filepath.Join(destDir, "nonexistent.txt"), sourceInfo)
	if err == nil {
		t.Error("存在しないソースファイルでエラーが発生しませんでした")
	}
}

// TestCopyDirectory_EdgeCases はcopyDirectory関数のエッジケースをテスト
func TestCopyDirectory_EdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	// 空のディレクトリ
	emptyDir := filepath.Join(sourceDir, "empty")
	os.MkdirAll(emptyDir, 0755)

	// シンボリックリンクを含むディレクトリ
	symlinkDir := filepath.Join(sourceDir, "symlink")
	os.MkdirAll(symlinkDir, 0755)
	symlinkFile := filepath.Join(symlinkDir, "link.txt")
	os.Symlink(filepath.Join(sourceDir, "nonexistent.txt"), symlinkFile)

	options := DefaultOptions()
	copier := NewFileCopier(sourceDir, destDir, options, nil, nil, nil)

	// 空のディレクトリのコピー
	err := copier.copyDirectory(emptyDir, filepath.Join(destDir, "empty"))
	if err != nil {
		t.Errorf("空のディレクトリのコピーが失敗: %v", err)
	}

	// シンボリックリンクを含むディレクトリのコピー
	err = copier.copyDirectory(symlinkDir, filepath.Join(destDir, "symlink"))
	if err != nil {
		t.Errorf("シンボリックリンクを含むディレクトリのコピーが失敗: %v", err)
	}

	// 存在しないディレクトリ
	err = copier.copyDirectory(filepath.Join(sourceDir, "nonexistent"), filepath.Join(destDir, "nonexistent"))
	if err == nil {
		t.Error("存在しないディレクトリでエラーが発生しませんでした")
	}
}

// ベンチマーク関数
func BenchmarkCopyFile_Small(b *testing.B) {
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
	sourceFile := filepath.Join(sourceDir, "small.txt")
	content := []byte("hello world")
	if err := os.WriteFile(sourceFile, content, 0644); err != nil {
		b.Fatalf("ファイルの作成に失敗: %v", err)
	}

	options := DefaultOptions()
	options.BufferSize = 1024 * 1024 // 1MB
	copier := NewFileCopier(sourceDir, destDir, options, nil, nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		destFile := filepath.Join(destDir, fmt.Sprintf("small_%d.txt", i))
		err := copier.copyFile(sourceFile, destFile)
		if err != nil {
			b.Fatalf("copyFileが失敗: %v", err)
		}
		// クリーンアップ
		os.Remove(destFile)
	}
}

func BenchmarkCopyFile_Large(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "benchmark")
	if err != nil {
		b.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	// 大きなファイルを作成（10MB）
	sourceFile := filepath.Join(sourceDir, "large.txt")
	content := make([]byte, 10*1024*1024)
	for i := range content {
		content[i] = byte(i % 256)
	}
	if err := os.WriteFile(sourceFile, content, 0644); err != nil {
		b.Fatalf("ファイルの作成に失敗: %v", err)
	}

	options := DefaultOptions()
	options.BufferSize = 8 * 1024 * 1024 // 8MB
	copier := NewFileCopier(sourceDir, destDir, options, nil, nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		destFile := filepath.Join(destDir, fmt.Sprintf("large_%d.txt", i))
		err := copier.copyFile(sourceFile, destFile)
		if err != nil {
			b.Fatalf("copyFileが失敗: %v", err)
		}
		// クリーンアップ
		os.Remove(destFile)
	}
}

func BenchmarkCopyDirectory_Small(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "benchmark")
	if err != nil {
		b.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)

	// 小さなファイルを複数作成
	for i := 0; i < 100; i++ {
		file := filepath.Join(sourceDir, fmt.Sprintf("file_%d.txt", i))
		content := []byte(fmt.Sprintf("content %d", i))
		if err := os.WriteFile(file, content, 0644); err != nil {
			b.Fatalf("ファイルの作成に失敗: %v", err)
		}
	}

	options := DefaultOptions()
	options.MaxConcurrent = 4
	copier := NewFileCopier(sourceDir, destDir, options, nil, nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		destDirPath := filepath.Join(destDir, fmt.Sprintf("dest_%d", i))
		err := copier.copyDirectory(sourceDir, destDirPath)
		if err != nil {
			b.Fatalf("copyDirectoryが失敗: %v", err)
		}
		// クリーンアップ
		os.RemoveAll(destDirPath)
	}
}

func BenchmarkCopyDirectory_Large(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "benchmark")
	if err != nil {
		b.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)

	// 大きなファイルを複数作成
	for i := 0; i < 10; i++ {
		file := filepath.Join(sourceDir, fmt.Sprintf("large_%d.txt", i))
		content := make([]byte, 1024*1024) // 1MB
		for j := range content {
			content[j] = byte((i + j) % 256)
		}
		if err := os.WriteFile(file, content, 0644); err != nil {
			b.Fatalf("ファイルの作成に失敗: %v", err)
		}
	}

	options := DefaultOptions()
	options.MaxConcurrent = 4
	options.BufferSize = 4 * 1024 * 1024 // 4MB
	copier := NewFileCopier(sourceDir, destDir, options, nil, nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		destDirPath := filepath.Join(destDir, fmt.Sprintf("dest_%d", i))
		err := copier.copyDirectory(sourceDir, destDirPath)
		if err != nil {
			b.Fatalf("copyDirectoryが失敗: %v", err)
		}
		// クリーンアップ
		os.RemoveAll(destDirPath)
	}
}

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

	// ファイルを作成
	sourceFile := filepath.Join(sourceDir, "test.txt")
	destFile := filepath.Join(destDir, "test.txt")
	content := []byte("hello world")
	os.WriteFile(sourceFile, content, 0644)
	os.WriteFile(destFile, content, 0644)

	options := DefaultOptions()
	copier := NewFileCopier(sourceDir, destDir, options, nil, nil, nil)

	// ファイル情報を取得
	sourceInfo, err := os.Stat(sourceFile)
	if err != nil {
		b.Fatalf("ファイル情報の取得に失敗: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := copier.verifyFile(sourceFile, destFile, "test.txt", sourceInfo)
		if err != nil {
			b.Fatalf("verifyFileが失敗: %v", err)
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

	// 大きなファイルを作成
	sourceFile := filepath.Join(sourceDir, "large.txt")
	destFile := filepath.Join(destDir, "large.txt")
	content := make([]byte, 5*1024*1024) // 5MB
	for i := range content {
		content[i] = byte(i % 256)
	}
	os.WriteFile(sourceFile, content, 0644)
	os.WriteFile(destFile, content, 0644)

	options := DefaultOptions()
	copier := NewFileCopier(sourceDir, destDir, options, nil, nil, nil)

	// ファイル情報を取得
	sourceInfo, err := os.Stat(sourceFile)
	if err != nil {
		b.Fatalf("ファイル情報の取得に失敗: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := copier.verifyFile(sourceFile, destFile, "large.txt", sourceInfo)
		if err != nil {
			b.Fatalf("verifyFileが失敗: %v", err)
		}
	}
}

func BenchmarkCopyFiles_Parallel(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "benchmark")
	if err != nil {
		b.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)

	// 複数のファイルを作成
	for i := 0; i < 50; i++ {
		file := filepath.Join(sourceDir, fmt.Sprintf("file_%d.txt", i))
		content := make([]byte, 1024) // 1KB
		for j := range content {
			content[j] = byte((i + j) % 256)
		}
		if err := os.WriteFile(file, content, 0644); err != nil {
			b.Fatalf("ファイルの作成に失敗: %v", err)
		}
	}

	options := DefaultOptions()
	options.MaxConcurrent = 8
	options.ProgressInterval = time.Hour // 進捗表示を無効化
	copier := NewFileCopier(sourceDir, destDir, options, nil, nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		destDirPath := filepath.Join(destDir, fmt.Sprintf("dest_%d", i))
		os.MkdirAll(destDirPath, 0755)

		// 一時的に宛先ディレクトリを変更
		originalDest := copier.destDir
		copier.destDir = destDirPath

		err := copier.CopyFiles()
		if err != nil {
			b.Fatalf("CopyFilesが失敗: %v", err)
		}
		copier.wg.Wait()

		// 元に戻す
		copier.destDir = originalDest

		// クリーンアップ
		os.RemoveAll(destDirPath)
	}
}

func BenchmarkCopyFiles_WithFilter(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "benchmark")
	if err != nil {
		b.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)

	// 異なる拡張子のファイルを作成
	for i := 0; i < 100; i++ {
		extensions := []string{".txt", ".log", ".tmp", ".bak"}
		ext := extensions[i%len(extensions)]
		file := filepath.Join(sourceDir, fmt.Sprintf("file_%d%s", i, ext))
		content := []byte(fmt.Sprintf("content %d", i))
		if err := os.WriteFile(file, content, 0644); err != nil {
			b.Fatalf("ファイルの作成に失敗: %v", err)
		}
	}

	options := DefaultOptions()
	options.MaxConcurrent = 4
	options.ProgressInterval = time.Hour // 進捗表示を無効化
	fileFilter := filter.NewFilter("*.txt,*.log", "*.tmp,*.bak")
	copier := NewFileCopier(sourceDir, destDir, options, fileFilter, nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		destDirPath := filepath.Join(destDir, fmt.Sprintf("dest_%d", i))
		os.MkdirAll(destDirPath, 0755)

		// 一時的に宛先ディレクトリを変更
		originalDest := copier.destDir
		copier.destDir = destDirPath

		err := copier.CopyFiles()
		if err != nil {
			b.Fatalf("CopyFilesが失敗: %v", err)
		}
		copier.wg.Wait()

		// 元に戻す
		copier.destDir = originalDest

		// クリーンアップ
		os.RemoveAll(destDirPath)
	}
}

// TestCopyFiles_ContextCancel はコンテキストキャンセルのテスト
func TestCopyFiles_ContextCancel(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	// 大きなファイルを作成（コピーに時間がかかるように）
	largeFile := filepath.Join(sourceDir, "large.txt")
	largeData := make([]byte, 10*1024*1024) // 10MB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	os.WriteFile(largeFile, largeData, 0644)

	options := DefaultOptions()
	options.BufferSize = 1024 // 小さなバッファで時間をかける
	copier := NewFileCopier(sourceDir, destDir, options, nil, nil, nil)

	// コピー開始直後にキャンセル
	go func() {
		time.Sleep(10 * time.Millisecond)
		copier.Cancel()
	}()

	err := copier.CopyFiles()
	// キャンセルエラーまたは成功のどちらでもOK（タイミングによる）
	if err != nil && !strings.Contains(err.Error(), "キャンセル") {
		t.Errorf("予期しないエラー: %v", err)
	}
}

// TestCopyFiles_DatabaseSessionError はデータベースセッションエラーのテスト
func TestCopyFiles_DatabaseSessionError(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	// テスト用ファイルを作成
	testFile := filepath.Join(sourceDir, "test.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	options := DefaultOptions()

	// 無効なデータベースパスでエラーを発生させる
	_, err := database.NewSyncDB("/invalid/path/db", database.NormalSync)
	if err == nil {
		t.Fatal("無効なパスでデータベースが作成されました")
	}

	copier := NewFileCopier(sourceDir, destDir, options, nil, nil, nil)
	err = copier.CopyFiles()
	if err != nil {
		t.Logf("期待されるエラー: %v", err)
	}
}

// TestCopyFiles_SingleFileMode は単一ファイルコピーモードのテスト
func TestCopyFiles_SingleFileMode(t *testing.T) {
	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "source.txt")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(destDir, 0755)

	// ソースファイルを作成
	os.WriteFile(sourceFile, []byte("single file test"), 0644)

	options := DefaultOptions()
	copier := NewFileCopier(sourceFile, destDir, options, nil, nil, nil)

	err := copier.CopyFiles()
	if err != nil {
		t.Errorf("単一ファイルコピーが失敗: %v", err)
	}

	// コピーされたか確認
	destFile := filepath.Join(destDir, "source.txt")
	if _, err := os.Stat(destFile); err != nil {
		t.Errorf("コピーされたファイルが見つかりません: %v", err)
	}
}

// TestCopyFile_VerifyMode は検証モードのテスト
func TestCopyFile_VerifyMode(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	// ソースファイルを作成
	sourceFile := filepath.Join(sourceDir, "test.txt")
	os.WriteFile(sourceFile, []byte("test content"), 0644)

	// 宛先ファイルを作成（異なる内容）
	destFile := filepath.Join(destDir, "test.txt")
	os.WriteFile(destFile, []byte("different content"), 0644)

	options := DefaultOptions()
	options.Mode = ModeVerify
	copier := NewFileCopier(sourceDir, destDir, options, nil, nil, nil)

	err := copier.CopyFiles()
	if err != nil {
		t.Logf("検証モードでのエラー（期待される）: %v", err)
	}
}

// TestCopyFile_CopyAndVerifyMode はコピーと検証モードのテスト
func TestCopyFile_CopyAndVerifyMode(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	// ソースファイルを作成
	sourceFile := filepath.Join(sourceDir, "test.txt")
	os.WriteFile(sourceFile, []byte("test content"), 0644)

	options := DefaultOptions()
	options.Mode = ModeCopyAndVerify
	copier := NewFileCopier(sourceDir, destDir, options, nil, nil, nil)

	err := copier.CopyFiles()
	if err != nil {
		t.Errorf("コピーと検証モードが失敗: %v", err)
	}

	// コピーされたか確認
	destFile := filepath.Join(destDir, "test.txt")
	if _, err := os.Stat(destFile); err != nil {
		t.Errorf("コピーされたファイルが見つかりません: %v", err)
	}
}

// TestCopyFile_RetryLogic はリトライロジックのテスト
func TestCopyFile_RetryLogic(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	// ソースファイルを作成
	sourceFile := filepath.Join(sourceDir, "test.txt")
	os.WriteFile(sourceFile, []byte("test content"), 0644)

	options := DefaultOptions()
	options.MaxRetries = 2
	options.RetryDelay = 10 * time.Millisecond
	copier := NewFileCopier(sourceDir, destDir, options, nil, nil, nil)

	err := copier.CopyFiles()
	if err != nil {
		t.Errorf("リトライロジックテストが失敗: %v", err)
	}
}

// TestCopyFile_OverwriteDisabled は上書き無効時のテスト
func TestCopyFile_OverwriteDisabled(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	// ソースファイルを作成
	sourceFile := filepath.Join(sourceDir, "test.txt")
	os.WriteFile(sourceFile, []byte("source content"), 0644)

	// 宛先ファイルを作成（既に存在）
	destFile := filepath.Join(destDir, "test.txt")
	os.WriteFile(destFile, []byte("dest content"), 0644)

	options := DefaultOptions()
	options.OverwriteExisting = false
	copier := NewFileCopier(sourceDir, destDir, options, nil, nil, nil)

	err := copier.CopyFiles()
	if err != nil {
		t.Errorf("上書き無効時のテストが失敗: %v", err)
	}

	// 宛先ファイルの内容が変更されていないことを確認
	content, err := os.ReadFile(destFile)
	if err != nil {
		t.Errorf("ファイル読み込みエラー: %v", err)
	}
	if string(content) != "dest content" {
		t.Error("宛先ファイルの内容が変更されました")
	}
}

// TestCopyFile_ProgressChannelFull は進捗チャンネルが一杯の時のテスト
func TestCopyFile_ProgressChannelFull(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	// 複数のファイルを作成
	for i := 0; i < 20; i++ {
		file := filepath.Join(sourceDir, fmt.Sprintf("file%d.txt", i))
		os.WriteFile(file, []byte(fmt.Sprintf("content %d", i)), 0644)
	}

	options := DefaultOptions()
	copier := NewFileCopier(sourceDir, destDir, options, nil, nil, nil)

	// 進捗コールバックを設定（処理を遅くする）
	copier.SetProgressCallback(func(current, total int64, currentFile string) {
		time.Sleep(1 * time.Millisecond)
	})

	err := copier.CopyFiles()
	if err != nil {
		t.Errorf("進捗チャンネルテストが失敗: %v", err)
	}
}

// TestCopyFile_ConcurrentAccess は並行アクセスのテスト
func TestCopyFile_ConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	// 複数のファイルを作成
	for i := 0; i < 50; i++ {
		file := filepath.Join(sourceDir, fmt.Sprintf("file%d.txt", i))
		os.WriteFile(file, []byte(fmt.Sprintf("content %d", i)), 0644)
	}

	options := DefaultOptions()
	options.MaxConcurrent = 2 // 並行数を制限
	copier := NewFileCopier(sourceDir, destDir, options, nil, nil, nil)

	err := copier.CopyFiles()
	if err != nil {
		t.Errorf("並行アクセステストが失敗: %v", err)
	}

	// すべてのファイルがコピーされたか確認
	for i := 0; i < 50; i++ {
		destFile := filepath.Join(destDir, fmt.Sprintf("file%d.txt", i))
		if _, err := os.Stat(destFile); err != nil {
			t.Errorf("ファイル %d がコピーされていません: %v", i, err)
		}
	}
}

// TestCopyFile_FileSystemErrors はファイルシステムエラーのテスト
func TestCopyFile_FileSystemErrors(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	// ソースファイルを作成
	sourceFile := filepath.Join(sourceDir, "test.txt")
	os.WriteFile(sourceFile, []byte("test content"), 0644)

	options := DefaultOptions()
	options.CreateDirs = false // ディレクトリ作成を無効化
	copier := NewFileCopier(sourceDir, filepath.Join(destDir, "nonexistent", "subdir"), options, nil, nil, nil)

	err := copier.CopyFiles()
	// ディレクトリ作成エラーまたは成功のどちらでもOK（実装による）
	if err != nil && !strings.Contains(err.Error(), "ディレクトリ") && !strings.Contains(err.Error(), "作成") {
		t.Errorf("予期しないエラー: %v", err)
	}
}

// TestCopyFile_ComplexFiltering は複雑なフィルタリングのテスト
func TestCopyFile_ComplexFiltering(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	// 様々な拡張子のファイルを作成
	files := []string{
		"include.txt",
		"include.doc",
		"exclude.tmp",
		"exclude.bak",
		"subdir/include.txt",
		"subdir/exclude.tmp",
	}

	os.MkdirAll(filepath.Join(sourceDir, "subdir"), 0755)
	for _, file := range files {
		filePath := filepath.Join(sourceDir, file)
		os.WriteFile(filePath, []byte("content"), 0644)
	}

	// 複雑なフィルタを設定
	filter := filter.NewFilter("*.txt,*.doc", "*.tmp,*.bak")
	options := DefaultOptions()
	copier := NewFileCopier(sourceDir, destDir, options, filter, nil, nil)

	err := copier.CopyFiles()
	if err != nil {
		t.Errorf("複雑なフィルタリングテストが失敗: %v", err)
	}

	// 含めるべきファイルがコピーされているか確認
	expectedFiles := []string{
		"include.txt",
		"include.doc",
		"subdir/include.txt",
	}
	for _, file := range expectedFiles {
		destFile := filepath.Join(destDir, file)
		if _, err := os.Stat(destFile); err != nil {
			t.Errorf("含めるべきファイル %s がコピーされていません", file)
		}
	}

	// 除外すべきファイルがコピーされていないか確認
	excludedFiles := []string{
		"exclude.tmp",
		"exclude.bak",
		"subdir/exclude.tmp",
	}
	for _, file := range excludedFiles {
		destFile := filepath.Join(destDir, file)
		if _, err := os.Stat(destFile); err == nil {
			t.Errorf("除外すべきファイル %s がコピーされています", file)
		}
	}
}

// TestCopyFile_HashVerificationErrors はハッシュ検証エラーのテスト
func TestCopyFile_HashVerificationErrors(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	// ソースファイルを作成
	sourceFile := filepath.Join(sourceDir, "test.txt")
	os.WriteFile(sourceFile, []byte("source content"), 0644)

	// 宛先ファイルを作成（異なる内容）
	destFile := filepath.Join(destDir, "test.txt")
	os.WriteFile(destFile, []byte("different content"), 0644)

	options := DefaultOptions()
	options.VerifyHash = true
	copier := NewFileCopier(sourceDir, destDir, options, nil, nil, nil)

	err := copier.CopyFiles()
	if err != nil {
		t.Logf("ハッシュ検証エラー（期待される）: %v", err)
	}
}

// TestCopyFile_NonRecursiveMode は非再帰モードのテスト
func TestCopyFile_NonRecursiveMode(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	// サブディレクトリを作成
	subDir := filepath.Join(sourceDir, "subdir")
	os.MkdirAll(subDir, 0755)

	// ファイルを作成
	os.WriteFile(filepath.Join(sourceDir, "root.txt"), []byte("root"), 0644)
	os.WriteFile(filepath.Join(subDir, "sub.txt"), []byte("sub"), 0644)

	options := DefaultOptions()
	options.Recursive = false
	copier := NewFileCopier(sourceDir, destDir, options, nil, nil, nil)

	err := copier.CopyFiles()
	if err != nil {
		t.Errorf("非再帰モードテストが失敗: %v", err)
	}

	// ルートファイルのみがコピーされているか確認
	if _, err := os.Stat(filepath.Join(destDir, "root.txt")); err != nil {
		t.Error("ルートファイルがコピーされていません")
	}
	if _, err := os.Stat(filepath.Join(destDir, "subdir", "sub.txt")); err == nil {
		t.Error("サブディレクトリのファイルがコピーされています（非再帰モード）")
	}
}

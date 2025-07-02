package copier

import (
	"fmt"
	"os"
	"path/filepath"
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

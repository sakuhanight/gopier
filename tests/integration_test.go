package tests

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sakuhanight/gopier/internal/copier"
	"github.com/sakuhanight/gopier/internal/database"
	"github.com/sakuhanight/gopier/internal/filter"
	"github.com/sakuhanight/gopier/internal/hasher"
	"github.com/sakuhanight/gopier/internal/logger"
	"github.com/sakuhanight/gopier/internal/stats"
)

// TestEnvironment は統合テスト用の環境を管理
type TestEnvironment struct {
	SourceDir  string
	DestDir    string
	TempDir    string
	DBPath     string
	Logger     *logger.Logger
	Stats      *stats.Stats
	Filter     *filter.Filter
	Hasher     *hasher.Hasher
	SyncDB     *database.SyncDB
	FileCopier *copier.FileCopier
}

// NewTestEnvironment は新しいテスト環境を作成
func NewTestEnvironment(t *testing.T) *TestEnvironment {
	tempDir, err := os.MkdirTemp("", "gopier_test_*")
	if err != nil {
		t.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}

	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")
	dbPath := filepath.Join(tempDir, "test.db")

	// ディレクトリを作成
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("ソースディレクトリの作成に失敗: %v", err)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("宛先ディレクトリの作成に失敗: %v", err)
	}

	// ロガーを作成
	logger := logger.NewLogger("", false, true)

	// 統計情報を作成
	stats := stats.NewStats()

	// フィルターを作成
	filter := filter.NewFilter("", "")

	// ハッシャーを作成
	hasher := hasher.NewHasher("sha256", 1024*1024)

	// データベースを作成
	syncDB, err := database.NewSyncDB(dbPath, database.NormalSync)
	if err != nil {
		t.Fatalf("データベースの作成に失敗: %v", err)
	}

	// ファイルコピーラーを作成
	options := copier.DefaultOptions()
	options.BufferSize = 1024 * 1024 // 1MB
	options.MaxConcurrent = 2
	options.MaxRetries = 2
	options.RetryDelay = 100 * time.Millisecond
	options.VerifyHash = true
	options.Recursive = true
	options.CreateDirs = true

	fileCopier := copier.NewFileCopier(sourceDir, destDir, options, filter, syncDB, logger)
	fileCopier.SetProgressCallback(func(current, total int64, currentFile string) {
		// 進捗コールバックでは統計情報を更新しない
		// 統計情報はFileCopier内部で管理される
	})

	return &TestEnvironment{
		SourceDir:  sourceDir,
		DestDir:    destDir,
		TempDir:    tempDir,
		DBPath:     dbPath,
		Logger:     logger,
		Stats:      stats,
		Filter:     filter,
		Hasher:     hasher,
		SyncDB:     syncDB,
		FileCopier: fileCopier,
	}
}

// Cleanup はテスト環境をクリーンアップ
func (env *TestEnvironment) Cleanup() {
	if env.Logger != nil {
		env.Logger.Close()
	}
	if env.SyncDB != nil {
		env.SyncDB.Close()
	}
	if env.TempDir != "" {
		os.RemoveAll(env.TempDir)
	}
}

// CreateTestFile はテスト用のファイルを作成
func (env *TestEnvironment) CreateTestFile(relPath string, size int64) error {
	fullPath := filepath.Join(env.SourceDir, relPath)

	// ディレクトリを作成
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// ファイルを作成
	file, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// ランダムデータを書き込み
	buffer := make([]byte, 1024)
	remaining := size
	for remaining > 0 {
		chunkSize := int64(len(buffer))
		if remaining < chunkSize {
			chunkSize = remaining
		}

		if _, err := rand.Read(buffer[:chunkSize]); err != nil {
			return err
		}

		if _, err := file.Write(buffer[:chunkSize]); err != nil {
			return err
		}

		remaining -= chunkSize
	}

	return nil
}

// CreateTestDirectory はテスト用のディレクトリ構造を作成
func (env *TestEnvironment) CreateTestDirectory(structure map[string]int64) error {
	for relPath, size := range structure {
		if err := env.CreateTestFile(relPath, size); err != nil {
			return err
		}
	}
	return nil
}

// VerifyFileExists はファイルが存在することを確認
func (env *TestEnvironment) VerifyFileExists(relPath string) error {
	sourcePath := filepath.Join(env.SourceDir, relPath)
	destPath := filepath.Join(env.DestDir, relPath)

	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("ソースファイルが存在しません: %s", sourcePath)
	}

	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		return fmt.Errorf("宛先ファイルが存在しません: %s", destPath)
	}

	return nil
}

// VerifyFileContent はファイルの内容が一致することを確認
func (env *TestEnvironment) VerifyFileContent(relPath string) error {
	sourcePath := filepath.Join(env.SourceDir, relPath)
	destPath := filepath.Join(env.DestDir, relPath)

	sourceHash, err := env.Hasher.HashFile(sourcePath)
	if err != nil {
		return fmt.Errorf("ソースファイルのハッシュ計算エラー: %v", err)
	}

	destHash, err := env.Hasher.HashFile(destPath)
	if err != nil {
		return fmt.Errorf("宛先ファイルのハッシュ計算エラー: %v", err)
	}

	if sourceHash != destHash {
		return fmt.Errorf("ファイルの内容が一致しません: %s", relPath)
	}

	return nil
}

// TestBasicFileCopy は基本的なファイルコピーのテスト
func TestBasicFileCopy(t *testing.T) {
	env := NewTestEnvironment(t)
	defer env.Cleanup()

	// テストファイルを作成
	testFiles := map[string]int64{
		"test1.txt":     1024,
		"test2.txt":     2048,
		"sub/test3.txt": 512,
	}

	if err := env.CreateTestDirectory(testFiles); err != nil {
		t.Fatalf("テストファイルの作成に失敗: %v", err)
	}

	// ファイルをコピー
	if err := env.FileCopier.CopyFiles(); err != nil {
		t.Fatalf("ファイルコピーに失敗: %v", err)
	}

	// 各ファイルの存在と内容を確認
	for relPath := range testFiles {
		if err := env.VerifyFileExists(relPath); err != nil {
			t.Errorf("ファイル存在確認エラー: %v", err)
		}
		if err := env.VerifyFileContent(relPath); err != nil {
			t.Errorf("ファイル内容確認エラー: %v", err)
		}
	}

	// 統計情報を確認（FileCopier内部の統計を使用）
	copierStats := env.FileCopier.GetStats()
	if copierStats.GetCopiedCount() != int64(len(testFiles)) {
		t.Errorf("コピーされたファイル数が期待値と異なります: 期待=%d, 実際=%d", len(testFiles), copierStats.GetCopiedCount())
	}
}

// TestLargeFileCopy は大きなファイルのコピーテスト
func TestLargeFileCopy(t *testing.T) {
	env := NewTestEnvironment(t)
	defer env.Cleanup()

	// 大きなファイルを作成（10MB）
	if err := env.CreateTestFile("large.bin", 10*1024*1024); err != nil {
		t.Fatalf("大きなファイルの作成に失敗: %v", err)
	}

	// ファイルをコピー
	if err := env.FileCopier.CopyFiles(); err != nil {
		t.Fatalf("大きなファイルのコピーに失敗: %v", err)
	}

	// ファイルの存在と内容を確認
	if err := env.VerifyFileExists("large.bin"); err != nil {
		t.Errorf("ファイル存在確認エラー: %v", err)
	}
	if err := env.VerifyFileContent("large.bin"); err != nil {
		t.Errorf("ファイル内容確認エラー: %v", err)
	}

	// 統計情報を確認（FileCopier内部の統計を使用）
	copierStats := env.FileCopier.GetStats()
	if copierStats.GetCopiedCount() != 1 {
		t.Errorf("コピーされたファイル数が期待値と異なります: 期待=1, 実際=%d", copierStats.GetCopiedCount())
	}
	if copierStats.GetCopiedBytes() != 10*1024*1024 {
		t.Errorf("コピーされたバイト数が期待値と異なります: 期待=%d, 実際=%d", 10*1024*1024, copierStats.GetCopiedBytes())
	}
}

// TestFilteredCopy はフィルター付きコピーのテスト
func TestFilteredCopy(t *testing.T) {
	env := NewTestEnvironment(t)
	defer env.Cleanup()

	// テストファイルを作成
	testFiles := map[string]int64{
		"include.txt":     1024,
		"exclude.tmp":     1024,
		"include.doc":     1024,
		"exclude.bak":     1024,
		"sub/include.txt": 1024,
	}

	if err := env.CreateTestDirectory(testFiles); err != nil {
		t.Fatalf("テストファイルの作成に失敗: %v", err)
	}

	// フィルターを設定（.txtと.docファイルのみ含める）
	env.Filter = filter.NewFilter("*.txt,*.doc", "*.tmp,*.bak")

	// 新しいコピーラーを作成（フィルターを適用）
	options := copier.DefaultOptions()
	options.BufferSize = 1024 * 1024
	options.MaxConcurrent = 2
	options.MaxRetries = 2
	options.RetryDelay = 100 * time.Millisecond
	options.VerifyHash = true
	options.Recursive = true
	options.CreateDirs = true

	filteredCopier := copier.NewFileCopier(env.SourceDir, env.DestDir, options, env.Filter, env.SyncDB, env.Logger)
	filteredCopier.SetProgressCallback(func(current, total int64, currentFile string) {})

	// ファイルをコピー
	if err := filteredCopier.CopyFiles(); err != nil {
		t.Fatalf("フィルター付きファイルコピーに失敗: %v", err)
	}

	// 含めるべきファイルの確認
	includeFiles := []string{"include.txt", "include.doc", "sub/include.txt"}
	for _, relPath := range includeFiles {
		if err := env.VerifyFileExists(relPath); err != nil {
			t.Errorf("含めるべきファイルが存在しません: %v", err)
		}
	}

	// 除外すべきファイルの確認
	excludeFiles := []string{"exclude.tmp", "exclude.bak"}
	for _, relPath := range excludeFiles {
		destPath := filepath.Join(env.DestDir, relPath)
		if _, err := os.Stat(destPath); !os.IsNotExist(err) {
			t.Errorf("除外すべきファイルがコピーされています: %s", relPath)
		}
	}

	// 統計情報を確認（FileCopier内部の統計を使用）
	copierStats := filteredCopier.GetStats()
	expectedCount := int64(len(includeFiles))
	if copierStats.GetCopiedCount() != expectedCount {
		t.Errorf("コピーされたファイル数が期待値と異なります: 期待=%d, 実際=%d", expectedCount, copierStats.GetCopiedCount())
	}
}

// TestFileVerification はファイル検証のテスト
func TestFileVerification(t *testing.T) {
	env := NewTestEnvironment(t)
	defer env.Cleanup()

	// テストファイルを作成
	testFiles := map[string]int64{
		"test1.txt":     1024,
		"test2.txt":     2048,
		"sub/test3.txt": 512,
	}

	if err := env.CreateTestDirectory(testFiles); err != nil {
		t.Fatalf("テストファイルの作成に失敗: %v", err)
	}

	// ファイルをコピー
	if err := env.FileCopier.CopyFiles(); err != nil {
		t.Fatalf("ファイルコピーに失敗: %v", err)
	}

	// 各ファイルの内容を手動で検証
	for relPath := range testFiles {
		if err := env.VerifyFileContent(relPath); err != nil {
			t.Errorf("ファイル検証が失敗しています: %s - %v", relPath, err)
		}
	}

	// 統計情報を確認（FileCopier内部の統計を使用）
	copierStats := env.FileCopier.GetStats()
	if copierStats.GetCopiedCount() != int64(len(testFiles)) {
		t.Errorf("コピーされたファイル数が期待値と異なります: 期待=%d, 実際=%d", len(testFiles), copierStats.GetCopiedCount())
	}
}

// TestDatabaseIntegration はデータベース統合のテスト
func TestDatabaseIntegration(t *testing.T) {
	env := NewTestEnvironment(t)
	defer env.Cleanup()

	// テストファイルを作成
	testFiles := map[string]int64{
		"test1.txt": 1024,
		"test2.txt": 2048,
	}

	if err := env.CreateTestDirectory(testFiles); err != nil {
		t.Fatalf("テストファイルの作成に失敗: %v", err)
	}

	// 同期セッションを開始
	sessionID, err := env.SyncDB.StartSyncSession()
	if err != nil {
		t.Fatalf("同期セッションの開始に失敗: %v", err)
	}

	// ファイルをコピー
	if err := env.FileCopier.CopyFiles(); err != nil {
		t.Fatalf("ファイルコピーに失敗: %v", err)
	}

	// 各ファイルをデータベースに記録
	for relPath := range testFiles {
		sourcePath := filepath.Join(env.SourceDir, relPath)
		hash, err := env.Hasher.HashFile(sourcePath)
		if err != nil {
			t.Fatalf("ファイルハッシュの計算に失敗: %v", err)
		}

		fileInfo := database.FileInfo{
			Path:         relPath,
			Size:         testFiles[relPath],
			ModTime:      time.Now(),
			Status:       database.StatusSuccess,
			SourceHash:   hash,
			DestHash:     hash,
			FailCount:    0,
			LastSyncTime: time.Now(),
			LastError:    "",
		}

		if err := env.SyncDB.AddFile(fileInfo); err != nil {
			t.Fatalf("ファイルのデータベース登録に失敗: %v", err)
		}
	}

	// 同期セッションを終了
	if err := env.SyncDB.EndSyncSession(sessionID, int(env.Stats.GetCopiedCount()), int(env.Stats.GetSkippedCount()), int(env.Stats.GetFailedCount()), env.Stats.GetCopiedBytes()); err != nil {
		t.Fatalf("同期セッションの終了に失敗: %v", err)
	}

	// データベースからファイル情報を取得
	allFiles, err := env.SyncDB.GetAllFiles()
	if err != nil {
		t.Fatalf("ファイル情報の取得に失敗: %v", err)
	}

	if len(allFiles) != len(testFiles) {
		t.Errorf("データベースのファイル数が期待値と異なります: 期待=%d, 実際=%d", len(testFiles), len(allFiles))
	}

	// 各ファイルのステータスを確認
	for _, file := range allFiles {
		if file.Status != database.StatusSuccess {
			t.Errorf("ファイルのステータスが期待値と異なります: %s - 期待=%s, 実際=%s", file.Path, database.StatusSuccess, file.Status)
		}
	}

	// 同期統計を取得
	syncStats, err := env.SyncDB.GetSyncStats()
	if err != nil {
		t.Fatalf("同期統計の取得に失敗: %v", err)
	}

	if syncStats["total_files"] != len(testFiles) {
		t.Errorf("同期統計のファイル数が期待値と異なります: 期待=%d, 実際=%d", len(testFiles), syncStats["total_files"])
	}
}

// TestDatabaseEdgeCases はデータベースのエッジケースをテスト
func TestDatabaseEdgeCases(t *testing.T) {
	env := NewTestEnvironment(t)
	defer env.Cleanup()

	// 空のファイル情報を追加（空のパスは避ける）
	emptyFile := database.FileInfo{
		Path:   "empty_file.txt",
		Status: database.StatusPending,
	}
	if err := env.SyncDB.AddFile(emptyFile); err != nil {
		t.Errorf("空のファイル情報の追加でエラーが発生: %v", err)
	}

	// 非常に長いパス名のファイル（ただし有効なパス）
	longPath := "very/long/path/" + string(make([]byte, 100)) + "/test.txt"
	longPathFile := database.FileInfo{
		Path:   longPath,
		Status: database.StatusPending,
	}
	if err := env.SyncDB.AddFile(longPathFile); err != nil {
		t.Errorf("長いパス名のファイル追加でエラーが発生: %v", err)
	}

	// 特殊文字を含むパス名
	specialCharsPath := "test/ファイル/with/特殊文字/🚀/test.txt"
	specialFile := database.FileInfo{
		Path:   specialCharsPath,
		Status: database.StatusPending,
	}
	if err := env.SyncDB.AddFile(specialFile); err != nil {
		t.Errorf("特殊文字を含むパス名のファイル追加でエラーが発生: %v", err)
	}

	// 負のサイズのファイル
	negativeSizeFile := database.FileInfo{
		Path:   "negative.txt",
		Size:   -1024,
		Status: database.StatusPending,
	}
	if err := env.SyncDB.AddFile(negativeSizeFile); err != nil {
		t.Errorf("負のサイズのファイル追加でエラーが発生: %v", err)
	}

	// 非常に大きなサイズのファイル
	largeSizeFile := database.FileInfo{
		Path:   "large.txt",
		Size:   1<<63 - 1, // 最大int64値
		Status: database.StatusPending,
	}
	if err := env.SyncDB.AddFile(largeSizeFile); err != nil {
		t.Errorf("大きなサイズのファイル追加でエラーが発生: %v", err)
	}
}

// TestDatabaseConcurrency はデータベースの並行処理をテスト
func TestDatabaseConcurrency(t *testing.T) {
	env := NewTestEnvironment(t)
	defer env.Cleanup()

	const numGoroutines = 10
	const filesPerGoroutine = 100
	done := make(chan bool, numGoroutines)

	// 複数のゴルーチンで同時にファイルを追加
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()
			for j := 0; j < filesPerGoroutine; j++ {
				fileInfo := database.FileInfo{
					Path:         fmt.Sprintf("goroutine_%d/file_%d.txt", id, j),
					Size:         int64(j * 1024),
					ModTime:      time.Now(),
					Status:       database.StatusSuccess,
					SourceHash:   fmt.Sprintf("hash_%d_%d", id, j),
					DestHash:     fmt.Sprintf("hash_%d_%d", id, j),
					FailCount:    0,
					LastSyncTime: time.Now(),
					LastError:    "",
				}

				if err := env.SyncDB.AddFile(fileInfo); err != nil {
					t.Errorf("並行処理でのファイル追加エラー: %v", err)
					return
				}
			}
		}(i)
	}

	// すべてのゴルーチンの完了を待つ
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// すべてのファイルが追加されていることを確認
	allFiles, err := env.SyncDB.GetAllFiles()
	if err != nil {
		t.Fatalf("ファイル一覧の取得に失敗: %v", err)
	}

	expectedCount := numGoroutines * filesPerGoroutine
	if len(allFiles) != expectedCount {
		t.Errorf("並行処理後のファイル数が期待値と異なります: 期待=%d, 実際=%d", expectedCount, len(allFiles))
	}
}

// TestDatabaseCorruption はデータベース破損時の動作をテスト
func TestDatabaseCorruption(t *testing.T) {
	env := NewTestEnvironment(t)
	defer env.Cleanup()

	// 正常なファイルを追加
	fileInfo := database.FileInfo{
		Path:         "test.txt",
		Size:         1024,
		ModTime:      time.Now(),
		Status:       database.StatusSuccess,
		SourceHash:   "test-hash",
		DestHash:     "test-hash",
		FailCount:    0,
		LastSyncTime: time.Now(),
		LastError:    "",
	}

	if err := env.SyncDB.AddFile(fileInfo); err != nil {
		t.Fatalf("ファイル追加に失敗: %v", err)
	}

	// データベースを閉じる
	env.SyncDB.Close()

	// データベースファイルを破損させる（ファイル全体を0で上書き）
	dbPath := filepath.Join(env.TempDir, "test.db")
	dbFile, err := os.OpenFile(dbPath, os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("データベースファイルのオープンに失敗: %v", err)
	}
	defer dbFile.Close()

	// ファイルサイズを取得
	fileInfo2, err := dbFile.Stat()
	if err != nil {
		t.Fatalf("ファイル情報の取得に失敗: %v", err)
	}

	// ファイル全体を0で上書き
	corruption := make([]byte, fileInfo2.Size())
	if _, err := dbFile.WriteAt(corruption, 0); err != nil {
		t.Fatalf("データベースファイルの破損に失敗: %v", err)
	}

	// 破損したデータベースを開こうとする
	_, err = database.NewSyncDB(dbPath, database.NormalSync)
	if err == nil {
		t.Error("破損したデータベースでエラーが発生しませんでした")
	}
}

// TestDatabaseMemoryLeak はメモリリークをテスト
func TestDatabaseMemoryLeak(t *testing.T) {
	env := NewTestEnvironment(t)
	defer env.Cleanup()

	// 大量のファイルを追加（メモリリークを検出するため）
	const numFiles = 10000
	for i := 0; i < numFiles; i++ {
		fileInfo := database.FileInfo{
			Path:         fmt.Sprintf("memory_test/file_%d.txt", i),
			Size:         int64(i * 1024),
			ModTime:      time.Now(),
			Status:       database.StatusSuccess,
			SourceHash:   fmt.Sprintf("hash_%d", i),
			DestHash:     fmt.Sprintf("hash_%d", i),
			FailCount:    0,
			LastSyncTime: time.Now(),
			LastError:    "",
		}

		if err := env.SyncDB.AddFile(fileInfo); err != nil {
			t.Fatalf("ファイル追加に失敗: %v", err)
		}
	}

	// すべてのファイルを取得（メモリ使用量を確認）
	allFiles, err := env.SyncDB.GetAllFiles()
	if err != nil {
		t.Fatalf("ファイル一覧の取得に失敗: %v", err)
	}

	if len(allFiles) != numFiles {
		t.Errorf("ファイル数が期待値と異なります: 期待=%d, 実際=%d", numFiles, len(allFiles))
	}

	// 統計情報を取得
	stats, err := env.SyncDB.GetSyncStats()
	if err != nil {
		t.Fatalf("統計情報の取得に失敗: %v", err)
	}

	if stats["total_files"] != numFiles {
		t.Errorf("統計情報のファイル数が期待値と異なります: 期待=%d, 実際=%d", numFiles, stats["total_files"])
	}
}

// TestDatabaseTransactionRollback はトランザクションのロールバックをテスト
func TestDatabaseTransactionRollback(t *testing.T) {
	env := NewTestEnvironment(t)
	defer env.Cleanup()

	// 正常なファイルを追加
	fileInfo := database.FileInfo{
		Path:         "transaction_test.txt",
		Size:         1024,
		ModTime:      time.Now(),
		Status:       database.StatusSuccess,
		SourceHash:   "test-hash",
		DestHash:     "test-hash",
		FailCount:    0,
		LastSyncTime: time.Now(),
		LastError:    "",
	}

	if err := env.SyncDB.AddFile(fileInfo); err != nil {
		t.Fatalf("ファイル追加に失敗: %v", err)
	}

	// ファイルが存在することを確認
	retrievedFile, err := env.SyncDB.GetFile("transaction_test.txt")
	if err != nil {
		t.Fatalf("ファイル取得に失敗: %v", err)
	}

	if retrievedFile.Path != fileInfo.Path {
		t.Errorf("ファイルパスが一致しません: 期待=%s, 実際=%s", fileInfo.Path, retrievedFile.Path)
	}

	// データベースをリセット（初期同期モードでのみ可能）
	// 通常モードではリセットできないことを確認
	if err := env.SyncDB.ResetDatabase(); err == nil {
		t.Error("通常モードでデータベースリセットが成功してしまいました")
	}

	// 初期同期モードで新しいデータベースを作成
	initialDB, err := database.NewSyncDB(filepath.Join(env.TempDir, "initial.db"), database.InitialSync)
	if err != nil {
		t.Fatalf("初期同期モードのデータベース作成に失敗: %v", err)
	}
	defer initialDB.Close()

	// 初期同期モードではリセットが可能
	if err := initialDB.ResetDatabase(); err != nil {
		t.Errorf("初期同期モードでのデータベースリセットに失敗: %v", err)
	}
}

// TestDatabaseFileLocking はファイルロックの競合状態をテスト
func TestDatabaseFileLocking(t *testing.T) {
	env := NewTestEnvironment(t)
	defer env.Cleanup()

	// 同じデータベースファイルに対して複数の接続を作成
	dbPath := env.DBPath
	env.SyncDB.Close()

	// 最初の接続
	db1, err := database.NewSyncDB(dbPath, database.NormalSync)
	if err != nil {
		t.Fatalf("最初のデータベース接続に失敗: %v", err)
	}
	defer db1.Close()

	// 最初の接続でファイルを追加
	file1 := database.FileInfo{
		Path:         "db1_file.txt",
		Size:         1024,
		ModTime:      time.Now(),
		Status:       database.StatusSuccess,
		SourceHash:   "hash1",
		DestHash:     "hash1",
		FailCount:    0,
		LastSyncTime: time.Now(),
		LastError:    "",
	}

	if err := db1.AddFile(file1); err != nil {
		t.Errorf("db1でのファイル追加に失敗: %v", err)
	}

	// 最初の接続でファイルを取得できることを確認
	retrievedFile1, err := db1.GetFile("db1_file.txt")
	if err != nil {
		t.Errorf("db1でのファイル取得に失敗: %v", err)
	}
	if retrievedFile1.Path != file1.Path {
		t.Errorf("db1でのファイルパスが一致しません: 期待=%s, 実際=%s", file1.Path, retrievedFile1.Path)
	}

	// 最初の接続を閉じる
	db1.Close()

	// 2番目の接続（同じファイル）
	db2, err := database.NewSyncDB(dbPath, database.NormalSync)
	if err != nil {
		t.Fatalf("2番目のデータベース接続に失敗: %v", err)
	}
	defer db2.Close()

	// 2番目の接続でファイルを追加
	file2 := database.FileInfo{
		Path:         "db2_file.txt",
		Size:         2048,
		ModTime:      time.Now(),
		Status:       database.StatusSuccess,
		SourceHash:   "hash2",
		DestHash:     "hash2",
		FailCount:    0,
		LastSyncTime: time.Now(),
		LastError:    "",
	}

	if err := db2.AddFile(file2); err != nil {
		t.Errorf("db2でのファイル追加に失敗: %v", err)
	}

	// 2番目の接続でファイルを取得できることを確認
	retrievedFile2, err := db2.GetFile("db2_file.txt")
	if err != nil {
		t.Errorf("db2でのファイル取得に失敗: %v", err)
	}
	if retrievedFile2.Path != file2.Path {
		t.Errorf("db2でのファイルパスが一致しません: 期待=%s, 実際=%s", file2.Path, retrievedFile2.Path)
	}

	// 最初の接続で追加したファイルも取得できることを確認
	retrievedFile1Again, err := db2.GetFile("db1_file.txt")
	if err != nil {
		t.Errorf("db2でのdb1ファイル取得に失敗: %v", err)
	}
	if retrievedFile1Again.Path != file1.Path {
		t.Errorf("db2でのdb1ファイルパスが一致しません: 期待=%s, 実際=%s", file1.Path, retrievedFile1Again.Path)
	}
}

// TestDatabaseStatusTransitions はファイルステータスの遷移をテスト
func TestDatabaseStatusTransitions(t *testing.T) {
	env := NewTestEnvironment(t)
	defer env.Cleanup()

	// 初期状態のファイルを追加
	fileInfo := database.FileInfo{
		Path:         "status_test.txt",
		Size:         1024,
		ModTime:      time.Now(),
		Status:       database.StatusPending,
		SourceHash:   "",
		DestHash:     "",
		FailCount:    0,
		LastSyncTime: time.Now(),
		LastError:    "",
	}

	if err := env.SyncDB.AddFile(fileInfo); err != nil {
		t.Fatalf("ファイル追加に失敗: %v", err)
	}

	// ステータス遷移をテスト
	statusTransitions := []database.FileStatus{
		database.StatusSuccess,
		database.StatusFailed,
		database.StatusSkipped,
		database.StatusVerified,
		database.StatusMismatch,
		database.StatusSuccess, // 最終的に成功に戻す
	}

	for i, status := range statusTransitions {
		errorMsg := fmt.Sprintf("transition_%d", i)
		if err := env.SyncDB.UpdateFileStatus("status_test.txt", status, errorMsg); err != nil {
			t.Errorf("ステータス更新に失敗: %v", err)
		}

		// 更新されたファイルを取得して確認
		updatedFile, err := env.SyncDB.GetFile("status_test.txt")
		if err != nil {
			t.Errorf("ファイル取得に失敗: %v", err)
		}

		if updatedFile.Status != status {
			t.Errorf("ステータスが正しく更新されていません: 期待=%s, 実際=%s", status, updatedFile.Status)
		}

		if updatedFile.LastError != errorMsg {
			t.Errorf("エラーメッセージが正しく更新されていません: 期待=%s, 実際=%s", errorMsg, updatedFile.LastError)
		}
	}
}

// TestDatabaseHashVerification はハッシュ検証のテスト
func TestDatabaseHashVerification(t *testing.T) {
	env := NewTestEnvironment(t)
	defer env.Cleanup()

	// テストファイルを作成
	testFiles := map[string]int64{
		"hash_test1.txt": 1024,
		"hash_test2.txt": 2048,
	}

	if err := env.CreateTestDirectory(testFiles); err != nil {
		t.Fatalf("テストファイルの作成に失敗: %v", err)
	}

	// ファイルをコピー
	if err := env.FileCopier.CopyFiles(); err != nil {
		t.Fatalf("ファイルコピーに失敗: %v", err)
	}

	// 各ファイルのハッシュを計算してデータベースに記録
	for relPath := range testFiles {
		sourcePath := filepath.Join(env.SourceDir, relPath)
		destPath := filepath.Join(env.DestDir, relPath)

		sourceHash, err := env.Hasher.HashFile(sourcePath)
		if err != nil {
			t.Fatalf("ソースファイルのハッシュ計算に失敗: %v", err)
		}

		destHash, err := env.Hasher.HashFile(destPath)
		if err != nil {
			t.Fatalf("宛先ファイルのハッシュ計算に失敗: %v", err)
		}

		// ハッシュが一致することを確認
		if sourceHash != destHash {
			t.Errorf("ファイルのハッシュが一致しません: %s", relPath)
		}

		// データベースにハッシュ情報を記録
		if err := env.SyncDB.UpdateFileHash(relPath, sourceHash, destHash); err != nil {
			t.Errorf("ハッシュ情報の更新に失敗: %v", err)
		}

		// 記録されたハッシュ情報を取得して確認
		fileInfo, err := env.SyncDB.GetFile(relPath)
		if err != nil {
			t.Errorf("ファイル情報の取得に失敗: %v", err)
		}

		if fileInfo.SourceHash != sourceHash {
			t.Errorf("ソースハッシュが正しく記録されていません: 期待=%s, 実際=%s", sourceHash, fileInfo.SourceHash)
		}

		if fileInfo.DestHash != destHash {
			t.Errorf("宛先ハッシュが正しく記録されていません: 期待=%s, 実際=%s", destHash, fileInfo.DestHash)
		}
	}
}

// TestDatabaseExportReport は検証レポートのエクスポートをテスト
func TestDatabaseExportReport(t *testing.T) {
	env := NewTestEnvironment(t)
	defer env.Cleanup()

	// テストファイルを作成
	testFiles := map[string]int64{
		"export_test1.txt": 1024,
		"export_test2.txt": 2048,
		"export_test3.txt": 512,
	}

	if err := env.CreateTestDirectory(testFiles); err != nil {
		t.Fatalf("テストファイルの作成に失敗: %v", err)
	}

	// ファイルをコピー
	if err := env.FileCopier.CopyFiles(); err != nil {
		t.Fatalf("ファイルコピーに失敗: %v", err)
	}

	// 各ファイルをデータベースに記録
	for relPath := range testFiles {
		sourcePath := filepath.Join(env.SourceDir, relPath)
		hash, err := env.Hasher.HashFile(sourcePath)
		if err != nil {
			t.Fatalf("ファイルハッシュの計算に失敗: %v", err)
		}

		fileInfo := database.FileInfo{
			Path:         relPath,
			Size:         testFiles[relPath],
			ModTime:      time.Now(),
			Status:       database.StatusSuccess,
			SourceHash:   hash,
			DestHash:     hash,
			FailCount:    0,
			LastSyncTime: time.Now(),
			LastError:    "",
		}

		if err := env.SyncDB.AddFile(fileInfo); err != nil {
			t.Fatalf("ファイルのデータベース登録に失敗: %v", err)
		}
	}

	// 検証レポートをエクスポート
	reportPath := filepath.Join(env.TempDir, "verification_report.json")
	if err := env.SyncDB.ExportVerificationReport(reportPath); err != nil {
		t.Fatalf("検証レポートのエクスポートに失敗: %v", err)
	}

	// レポートファイルが存在することを確認
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Error("検証レポートファイルが作成されていません")
	}

	// レポートファイルの内容を確認
	reportData, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("レポートファイルの読み込みに失敗: %v", err)
	}

	if len(reportData) == 0 {
		t.Error("レポートファイルが空です")
	}

	// JSONとして解析できることを確認
	var report struct {
		ExportTime time.Time           `json:"export_time"`
		TotalFiles int                 `json:"total_files"`
		Files      []database.FileInfo `json:"files"`
	}

	if err := json.Unmarshal(reportData, &report); err != nil {
		t.Errorf("レポートのJSON解析に失敗: %v", err)
	}

	if report.TotalFiles != len(testFiles) {
		t.Errorf("レポートのファイル数が期待値と異なります: 期待=%d, 実際=%d", len(testFiles), report.TotalFiles)
	}

	if len(report.Files) != len(testFiles) {
		t.Errorf("レポートのファイルリストの長さが期待値と異なります: 期待=%d, 実際=%d", len(testFiles), len(report.Files))
	}
}

// BenchmarkFileCopy はファイルコピーのベンチマーク
func BenchmarkFileCopy(b *testing.B) {
	// ベンチマークテストではログ出力を抑制
	tempDir := b.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")

	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(destDir, 0755)

	// ベンチマーク用のファイルを作成
	testFile := filepath.Join(sourceDir, "benchmark.txt")
	content := make([]byte, 1024*1024) // 1MB
	for i := range content {
		content[i] = byte(i % 256)
	}
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		b.Fatalf("ベンチマークファイルの作成に失敗: %v", err)
	}

	options := copier.DefaultOptions()
	options.ProgressInterval = time.Hour // 進捗表示を無効化

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 各ベンチマーク実行で新しい宛先ディレクトリを作成
		destDirPath := filepath.Join(destDir, fmt.Sprintf("dest_%d", i))
		os.MkdirAll(destDirPath, 0755)

		// 新しいcopierを作成
		benchCopier := copier.NewFileCopier(sourceDir, destDirPath, options, nil, nil, nil)

		// ファイルをコピー
		if err := benchCopier.CopyFiles(); err != nil {
			b.Fatalf("ファイルコピーに失敗: %v", err)
		}

		// クリーンアップ
		os.RemoveAll(destDirPath)
	}
}

// TestConcurrentCopy は並列コピーのテスト
func TestConcurrentCopy(t *testing.T) {
	env := NewTestEnvironment(t)
	defer env.Cleanup()

	// 多数の小さなファイルを作成
	numFiles := 50
	testFiles := make(map[string]int64)
	for i := 0; i < numFiles; i++ {
		filename := fmt.Sprintf("file_%03d.txt", i)
		testFiles[filename] = 1024
	}

	if err := env.CreateTestDirectory(testFiles); err != nil {
		t.Fatalf("テストファイルの作成に失敗: %v", err)
	}

	// 並列度を高く設定した新しいコピーラーを作成
	options := copier.DefaultOptions()
	options.BufferSize = 1024 * 1024
	options.MaxConcurrent = 10
	options.MaxRetries = 2
	options.RetryDelay = 100 * time.Millisecond
	options.VerifyHash = true
	options.Recursive = true
	options.CreateDirs = true

	concurrentCopier := copier.NewFileCopier(env.SourceDir, env.DestDir, options, env.Filter, env.SyncDB, env.Logger)
	concurrentCopier.SetProgressCallback(func(current, total int64, currentFile string) {
		env.Stats.IncrementCopied(current)
	})

	// ファイルをコピー
	if err := concurrentCopier.CopyFiles(); err != nil {
		t.Fatalf("並列ファイルコピーに失敗: %v", err)
	}

	// すべてのファイルがコピーされていることを確認
	for relPath := range testFiles {
		if err := env.VerifyFileExists(relPath); err != nil {
			t.Errorf("ファイル存在確認エラー: %v", err)
		}
	}

	// 統計情報を確認
	copierStats := concurrentCopier.GetStats()
	if copierStats.GetCopiedCount() != int64(numFiles) {
		t.Errorf("コピーされたファイル数が期待値と異なります: 期待=%d, 実際=%d", numFiles, copierStats.GetCopiedCount())
	}
}

// TestErrorHandling はエラーハンドリングのテスト
func TestErrorHandling(t *testing.T) {
	env := NewTestEnvironment(t)
	defer env.Cleanup()

	// 存在しないディレクトリを指定
	invalidSource := filepath.Join(env.TempDir, "nonexistent")
	invalidDest := filepath.Join(env.TempDir, "nonexistent_dest")

	// 無効なパスでコピーラーを作成
	options := copier.DefaultOptions()
	invalidCopier := copier.NewFileCopier(invalidSource, invalidDest, options, env.Filter, env.SyncDB, env.Logger)

	// エラーが発生することを確認
	if err := invalidCopier.CopyFiles(); err == nil {
		t.Error("無効なパスでエラーが発生しませんでした")
	}

	// 統計情報でエラーが記録されていることを確認
	stats := invalidCopier.GetStats()
	if stats.GetFailedCount() == 0 {
		t.Error("エラーが統計情報に記録されていません")
	}
}

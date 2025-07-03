package tests

import (
	"crypto/rand"
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

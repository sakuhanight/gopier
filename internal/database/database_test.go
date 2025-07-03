package database

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.etcd.io/bbolt"
)

func TestNewSyncDB(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// 正常なケース
	db, err := NewSyncDB(dbPath, NormalSync)
	if err != nil {
		t.Fatalf("データベース作成が失敗: %v", err)
	}
	defer db.Close()

	// 無効なパスでのテスト
	_, err = NewSyncDB("/invalid/path/test.db", NormalSync)
	if err == nil {
		t.Error("無効なパスでエラーが発生しませんでした")
	}
}

func TestSyncDB_AddFile(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := NewSyncDB(dbPath, NormalSync)
	if err != nil {
		t.Fatalf("データベース作成が失敗: %v", err)
	}
	defer db.Close()

	// ファイル情報を追加
	fileInfo := FileInfo{
		Path:         "/test/file.txt",
		Size:         1024,
		ModTime:      time.Now(),
		SourceHash:   "test-hash",
		Status:       StatusSuccess,
		FailCount:    0,
		LastSyncTime: time.Now(),
	}

	err = db.AddFile(fileInfo)
	if err != nil {
		t.Errorf("ファイル追加が失敗: %v", err)
	}

	// 同じファイルを再度追加（更新されるはず）
	err = db.AddFile(fileInfo)
	if err != nil {
		t.Errorf("ファイル更新が失敗: %v", err)
	}
}

func TestSyncDB_GetFile(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := NewSyncDB(dbPath, NormalSync)
	if err != nil {
		t.Fatalf("データベース作成が失敗: %v", err)
	}
	defer db.Close()

	// ファイル情報を追加
	fileInfo := FileInfo{
		Path:         "/test/file.txt",
		Size:         1024,
		ModTime:      time.Now(),
		SourceHash:   "test-hash",
		Status:       StatusSuccess,
		FailCount:    0,
		LastSyncTime: time.Now(),
	}

	err = db.AddFile(fileInfo)
	if err != nil {
		t.Fatalf("ファイル追加が失敗: %v", err)
	}

	// ファイル情報を取得
	retrievedFile, err := db.GetFile("/test/file.txt")
	if err != nil {
		t.Errorf("ファイル取得が失敗: %v", err)
	}

	if retrievedFile.Path != fileInfo.Path {
		t.Errorf("パスが一致しません: 期待値=%s, 実際=%s", fileInfo.Path, retrievedFile.Path)
	}

	// 存在しないファイルを取得
	_, err = db.GetFile("/non-existent/file.txt")
	if err == nil {
		t.Error("存在しないファイルでエラーが発生しませんでした")
	}
}

func TestSyncDB_UpdateFileStatus(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := NewSyncDB(dbPath, NormalSync)
	if err != nil {
		t.Fatalf("データベース作成が失敗: %v", err)
	}
	defer db.Close()

	// ファイル情報を追加
	fileInfo := FileInfo{
		Path:         "/test/file.txt",
		Size:         1024,
		ModTime:      time.Now(),
		SourceHash:   "test-hash",
		Status:       StatusSuccess,
		FailCount:    0,
		LastSyncTime: time.Now(),
	}

	err = db.AddFile(fileInfo)
	if err != nil {
		t.Fatalf("ファイル追加が失敗: %v", err)
	}

	// ステータスを更新
	err = db.UpdateFileStatus("/test/file.txt", StatusFailed, "test error")
	if err != nil {
		t.Errorf("ステータス更新が失敗: %v", err)
	}

	// 更新されたファイル情報を取得
	updatedFile, err := db.GetFile("/test/file.txt")
	if err != nil {
		t.Fatalf("ファイル取得が失敗: %v", err)
	}

	if updatedFile.Status != StatusFailed {
		t.Errorf("ステータスが更新されていません: 期待値=%s, 実際=%s", StatusFailed, updatedFile.Status)
	}
}

func TestSyncDB_UpdateFileHash(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := NewSyncDB(dbPath, NormalSync)
	if err != nil {
		t.Fatalf("データベース作成が失敗: %v", err)
	}
	defer db.Close()

	// ファイル情報を追加
	fileInfo := FileInfo{
		Path:         "/test/file.txt",
		Size:         1024,
		ModTime:      time.Now(),
		SourceHash:   "old-hash",
		Status:       StatusSuccess,
		FailCount:    0,
		LastSyncTime: time.Now(),
	}

	err = db.AddFile(fileInfo)
	if err != nil {
		t.Fatalf("ファイル追加が失敗: %v", err)
	}

	// ハッシュを更新
	newHash := "new-hash"
	err = db.UpdateFileHash("/test/file.txt", newHash, "dest-hash")
	if err != nil {
		t.Errorf("ハッシュ更新が失敗: %v", err)
	}

	// 更新されたファイル情報を取得
	updatedFile, err := db.GetFile("/test/file.txt")
	if err != nil {
		t.Fatalf("ファイル取得が失敗: %v", err)
	}

	if updatedFile.SourceHash != newHash {
		t.Errorf("ハッシュが更新されていません: 期待値=%s, 実際=%s", newHash, updatedFile.SourceHash)
	}
}

func TestSyncDB_IncrementFailCount(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := NewSyncDB(dbPath, NormalSync)
	if err != nil {
		t.Fatalf("データベース作成が失敗: %v", err)
	}
	defer db.Close()

	// ファイル情報を追加
	fileInfo := FileInfo{
		Path:         "/test/file.txt",
		Size:         1024,
		ModTime:      time.Now(),
		SourceHash:   "test-hash",
		Status:       StatusSuccess,
		FailCount:    0,
		LastSyncTime: time.Now(),
	}

	err = db.AddFile(fileInfo)
	if err != nil {
		t.Fatalf("ファイル追加が失敗: %v", err)
	}

	// 失敗回数をインクリメント
	_, err = db.IncrementFailCount("/test/file.txt")
	if err != nil {
		t.Errorf("失敗回数インクリメントが失敗: %v", err)
	}

	// 更新されたファイル情報を取得
	updatedFile, err := db.GetFile("/test/file.txt")
	if err != nil {
		t.Fatalf("ファイル取得が失敗: %v", err)
	}

	if updatedFile.FailCount != 1 {
		t.Errorf("失敗回数が更新されていません: 期待値=1, 実際=%d", updatedFile.FailCount)
	}
}

func TestSyncDB_GetFailedFiles(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := NewSyncDB(dbPath, NormalSync)
	if err != nil {
		t.Fatalf("データベース作成が失敗: %v", err)
	}
	defer db.Close()

	// 失敗したファイルを追加
	failedFile := FileInfo{
		Path:         "/test/failed.txt",
		Size:         1024,
		ModTime:      time.Now(),
		SourceHash:   "test-hash",
		Status:       StatusFailed,
		FailCount:    3,
		LastSyncTime: time.Now(),
	}

	err = db.AddFile(failedFile)
	if err != nil {
		t.Fatalf("ファイル追加が失敗: %v", err)
	}

	// 失敗したファイルを取得
	failedFiles, err := db.GetFailedFiles(5)
	if err != nil {
		t.Errorf("失敗ファイル取得が失敗: %v", err)
	}

	if len(failedFiles) != 1 {
		t.Errorf("失敗ファイル数が一致しません: 期待値=1, 実際=%d", len(failedFiles))
	}

	if failedFiles[0].Path != failedFile.Path {
		t.Errorf("失敗ファイルのパスが一致しません: 期待値=%s, 実際=%s", failedFile.Path, failedFiles[0].Path)
	}
}

func TestSyncDB_GetFilesByStatus(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := NewSyncDB(dbPath, NormalSync)
	if err != nil {
		t.Fatalf("データベース作成が失敗: %v", err)
	}
	defer db.Close()

	// 異なるステータスのファイルを追加
	copiedFile := FileInfo{
		Path:         "/test/copied.txt",
		Size:         1024,
		ModTime:      time.Now(),
		SourceHash:   "test-hash",
		Status:       StatusSuccess,
		FailCount:    0,
		LastSyncTime: time.Now(),
	}

	failedFile := FileInfo{
		Path:         "/test/failed.txt",
		Size:         2048,
		ModTime:      time.Now(),
		SourceHash:   "test-hash-2",
		Status:       StatusFailed,
		FailCount:    1,
		LastSyncTime: time.Now(),
	}

	err = db.AddFile(copiedFile)
	if err != nil {
		t.Fatalf("ファイル追加が失敗: %v", err)
	}

	err = db.AddFile(failedFile)
	if err != nil {
		t.Fatalf("ファイル追加が失敗: %v", err)
	}

	// copiedステータスのファイルを取得
	copiedFiles, err := db.GetFilesByStatus(StatusSuccess)
	if err != nil {
		t.Errorf("copiedファイル取得が失敗: %v", err)
	}

	if len(copiedFiles) != 1 {
		t.Errorf("copiedファイル数が一致しません: 期待値=1, 実際=%d", len(copiedFiles))
	}

	// failedステータスのファイルを取得
	failedFiles, err := db.GetFilesByStatus(StatusFailed)
	if err != nil {
		t.Errorf("failedファイル取得が失敗: %v", err)
	}

	if len(failedFiles) != 1 {
		t.Errorf("failedファイル数が一致しません: 期待値=1, 実際=%d", len(failedFiles))
	}
}

func TestSyncDB_GetAllFiles(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := NewSyncDB(dbPath, NormalSync)
	if err != nil {
		t.Fatalf("データベース作成が失敗: %v", err)
	}
	defer db.Close()

	// 複数のファイルを追加
	file1 := FileInfo{
		Path:         "/test/file1.txt",
		Size:         1024,
		ModTime:      time.Now(),
		SourceHash:   "hash1",
		Status:       StatusSuccess,
		FailCount:    0,
		LastSyncTime: time.Now(),
	}

	file2 := FileInfo{
		Path:         "/test/file2.txt",
		Size:         2048,
		ModTime:      time.Now(),
		SourceHash:   "hash2",
		Status:       StatusSuccess,
		FailCount:    0,
		LastSyncTime: time.Now(),
	}

	err = db.AddFile(file1)
	if err != nil {
		t.Fatalf("ファイル1追加が失敗: %v", err)
	}

	err = db.AddFile(file2)
	if err != nil {
		t.Fatalf("ファイル2追加が失敗: %v", err)
	}

	// 全ファイルを取得
	allFiles, err := db.GetAllFiles()
	if err != nil {
		t.Errorf("全ファイル取得が失敗: %v", err)
	}

	if len(allFiles) != 2 {
		t.Errorf("ファイル数が一致しません: 期待値=2, 実際=%d", len(allFiles))
	}
}

func TestSyncDB_StartSyncSession(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := NewSyncDB(dbPath, NormalSync)
	if err != nil {
		t.Fatalf("データベース作成が失敗: %v", err)
	}
	defer db.Close()

	// 同期セッションを開始
	sessionID, err := db.StartSyncSession()
	if err != nil {
		t.Errorf("同期セッション開始が失敗: %v", err)
	}

	if sessionID == 0 {
		t.Error("セッションIDが0です")
	}
}

func TestSyncDB_EndSyncSession(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := NewSyncDB(dbPath, NormalSync)
	if err != nil {
		t.Fatalf("データベース作成が失敗: %v", err)
	}
	defer db.Close()

	// 同期セッションを開始
	sessionID, err := db.StartSyncSession()
	if err != nil {
		t.Fatalf("同期セッション開始が失敗: %v", err)
	}

	// 同期セッションを終了
	err = db.EndSyncSession(sessionID, 10, 5, 2, 1024*1024)
	if err != nil {
		t.Errorf("同期セッション終了が失敗: %v", err)
	}
}

func TestSyncDB_GetSyncStats(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := NewSyncDB(dbPath, NormalSync)
	if err != nil {
		t.Fatalf("データベース作成が失敗: %v", err)
	}
	defer db.Close()

	// 同期セッションを開始
	sessionID, err := db.StartSyncSession()
	if err != nil {
		t.Fatalf("同期セッション開始が失敗: %v", err)
	}

	// ファイルを追加
	fileInfo := FileInfo{
		Path:         "/test/file.txt",
		Size:         1024,
		ModTime:      time.Now(),
		SourceHash:   "test-hash",
		Status:       StatusSuccess,
		FailCount:    0,
		LastSyncTime: time.Now(),
	}

	err = db.AddFile(fileInfo)
	if err != nil {
		t.Fatalf("ファイル追加が失敗: %v", err)
	}

	// 同期セッションを終了
	err = db.EndSyncSession(sessionID, 1, 0, 0, 1024)
	if err != nil {
		t.Fatalf("同期セッション終了が失敗: %v", err)
	}

	// 同期統計を取得
	stats, err := db.GetSyncStats()
	if err != nil {
		t.Errorf("同期統計取得が失敗: %v", err)
	}

	if stats == nil {
		t.Error("同期統計がnilです")
	}
}

func TestSyncDB_ResetDatabase(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := NewSyncDB(dbPath, InitialSync)
	if err != nil {
		t.Fatalf("データベース作成が失敗: %v", err)
	}
	defer db.Close()

	// ファイルを追加
	fileInfo := FileInfo{
		Path:         "/test/file.txt",
		Size:         1024,
		ModTime:      time.Now(),
		SourceHash:   "test-hash",
		Status:       StatusSuccess,
		FailCount:    0,
		LastSyncTime: time.Now(),
	}

	err = db.AddFile(fileInfo)
	if err != nil {
		t.Fatalf("ファイル追加が失敗: %v", err)
	}

	// データベースをリセット
	err = db.ResetDatabase()
	if err != nil {
		t.Errorf("データベースリセットが失敗: %v", err)
	}

	// ファイルが削除されているか確認
	allFiles, err := db.GetAllFiles()
	if err != nil {
		t.Fatalf("全ファイル取得が失敗: %v", err)
	}

	if len(allFiles) != 0 {
		t.Errorf("ファイルが削除されていません: 期待値=0, 実際=%d", len(allFiles))
	}
}

func TestSyncDB_ExportVerificationReport(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := NewSyncDB(dbPath, NormalSync)
	if err != nil {
		t.Fatalf("データベース作成が失敗: %v", err)
	}
	defer db.Close()

	// ファイルを追加
	fileInfo := FileInfo{
		Path:         "/test/file.txt",
		Size:         1024,
		ModTime:      time.Now(),
		SourceHash:   "test-hash",
		Status:       StatusSuccess,
		FailCount:    0,
		LastSyncTime: time.Now(),
	}

	err = db.AddFile(fileInfo)
	if err != nil {
		t.Fatalf("ファイル追加が失敗: %v", err)
	}

	// 検証レポートをエクスポート
	reportPath := filepath.Join(tempDir, "verification_report.txt")
	err = db.ExportVerificationReport(reportPath)
	if err != nil {
		t.Errorf("検証レポートエクスポートが失敗: %v", err)
	}

	// レポートファイルが作成されているか確認
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Error("検証レポートファイルが作成されていません")
	}
}

func BenchmarkSyncDB_AddFile(b *testing.B) {
	tempDir := b.TempDir()
	dbPath := filepath.Join(tempDir, "bench.db")
	db, err := NewSyncDB(dbPath, NormalSync)
	if err != nil {
		b.Fatalf("データベース作成が失敗: %v", err)
	}
	defer db.Close()

	fileInfo := FileInfo{
		Path:         "/test/file.txt",
		Size:         1024,
		ModTime:      time.Now(),
		SourceHash:   "test-hash",
		Status:       StatusSuccess,
		FailCount:    0,
		LastSyncTime: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fileInfo.Path = filepath.Join("/test", "file", fmt.Sprintf("%d.txt", i))
		err := db.AddFile(fileInfo)
		if err != nil {
			b.Fatalf("ファイル追加が失敗: %v", err)
		}
	}
}

func BenchmarkSyncDB_GetFile(b *testing.B) {
	tempDir := b.TempDir()
	dbPath := filepath.Join(tempDir, "bench.db")
	db, err := NewSyncDB(dbPath, NormalSync)
	if err != nil {
		b.Fatalf("データベース作成が失敗: %v", err)
	}
	defer db.Close()

	// テストファイルを追加
	fileInfo := FileInfo{
		Path:         "/test/file.txt",
		Size:         1024,
		ModTime:      time.Now(),
		SourceHash:   "test-hash",
		Status:       StatusSuccess,
		FailCount:    0,
		LastSyncTime: time.Now(),
	}

	err = db.AddFile(fileInfo)
	if err != nil {
		b.Fatalf("ファイル追加が失敗: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := db.GetFile("/test/file.txt")
		if err != nil {
			b.Fatalf("ファイル取得が失敗: %v", err)
		}
	}
}

// TestInitBuckets_EdgeCases はinitBuckets関数のエッジケースをテスト
func TestInitBuckets_EdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// 正常なケース
	db, err := NewSyncDB(dbPath, NormalSync)
	if err != nil {
		t.Fatalf("データベース作成が失敗: %v", err)
	}
	defer db.Close()

	// バケットが正しく作成されているか確認
	err = db.AddFile(FileInfo{
		Path:   "/test/file.txt",
		Status: StatusSuccess,
	})
	if err != nil {
		t.Errorf("バケット初期化後のファイル追加が失敗: %v", err)
	}

	// 無効なパスでのテスト
	_, err = NewSyncDB("/invalid/path/test.db", NormalSync)
	if err == nil {
		t.Error("無効なパスでエラーが発生しませんでした")
	}
}

// TestResetDatabase_EdgeCases はResetDatabase関数のエッジケースをテスト
func TestResetDatabase_EdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// データベースを作成してファイルを追加
	db, err := NewSyncDB(dbPath, InitialSync)
	if err != nil {
		t.Fatalf("データベース作成が失敗: %v", err)
	}

	// ファイルを追加
	fileInfo := FileInfo{
		Path:         "/test/file.txt",
		Size:         1024,
		ModTime:      time.Now(),
		SourceHash:   "test-hash",
		Status:       StatusSuccess,
		FailCount:    0,
		LastSyncTime: time.Now(),
	}
	err = db.AddFile(fileInfo)
	if err != nil {
		t.Fatalf("ファイル追加が失敗: %v", err)
	}

	// データベースをリセット
	err = db.ResetDatabase()
	if err != nil {
		t.Errorf("データベースリセットが失敗: %v", err)
	}

	// リセット後はファイルが存在しないことを確認
	_, err = db.GetFile("/test/file.txt")
	if err == nil {
		t.Error("リセット後にファイルが残っています")
	}

	// リセット後に新しいファイルを追加できることを確認
	err = db.AddFile(FileInfo{
		Path:   "/new/file.txt",
		Status: StatusSuccess,
	})
	if err != nil {
		t.Errorf("リセット後のファイル追加が失敗: %v", err)
	}

	db.Close()
}

// TestExportVerificationReport_EdgeCases はExportVerificationReport関数のエッジケースをテスト
func TestExportVerificationReport_EdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	reportPath := filepath.Join(tempDir, "report.csv")

	db, err := NewSyncDB(dbPath, NormalSync)
	if err != nil {
		t.Fatalf("データベース作成が失敗: %v", err)
	}
	defer db.Close()

	// 空のデータベースでのレポート生成
	err = db.ExportVerificationReport(reportPath)
	if err != nil {
		t.Errorf("空のデータベースでのレポート生成が失敗: %v", err)
	}

	// ファイルが作成されたか確認
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Error("レポートファイルが作成されていません")
	}

	// 複数のファイルを含むレポート生成
	files := []FileInfo{
		{
			Path:         "/test/file1.txt",
			Size:         1024,
			ModTime:      time.Now(),
			SourceHash:   "hash1",
			Status:       StatusSuccess,
			FailCount:    0,
			LastSyncTime: time.Now(),
		},
		{
			Path:         "/test/file2.txt",
			Size:         2048,
			ModTime:      time.Now(),
			SourceHash:   "hash2",
			Status:       StatusFailed,
			FailCount:    1,
			LastSyncTime: time.Now(),
			LastError:    "test error",
		},
	}

	for _, file := range files {
		err = db.AddFile(file)
		if err != nil {
			t.Fatalf("ファイル追加が失敗: %v", err)
		}
	}

	reportPath2 := filepath.Join(tempDir, "report2.csv")
	err = db.ExportVerificationReport(reportPath2)
	if err != nil {
		t.Errorf("複数ファイルでのレポート生成が失敗: %v", err)
	}

	// 無効なパスでのレポート生成
	err = db.ExportVerificationReport("/invalid/path/report.csv")
	if err == nil {
		t.Error("無効なパスでエラーが発生しませんでした")
	}
}

// TestSyncSession_EdgeCases は同期セッション関連のエッジケースをテスト
func TestSyncSession_EdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := NewSyncDB(dbPath, NormalSync)
	if err != nil {
		t.Fatalf("データベース作成が失敗: %v", err)
	}
	defer db.Close()

	// セッション開始
	sessionID, err := db.StartSyncSession()
	if err != nil {
		t.Errorf("セッション開始が失敗: %v", err)
	}
	if sessionID == 0 {
		t.Error("セッションIDが0です")
	}

	// セッション終了
	err = db.EndSyncSession(sessionID, 10, 2, 1, 1024*1024)
	if err != nil {
		t.Errorf("セッション終了が失敗: %v", err)
	}

	// 存在しないセッションIDでの終了
	err = db.EndSyncSession(999999, 0, 0, 0, 0)
	if err == nil {
		t.Error("存在しないセッションIDでエラーが発生しませんでした")
	}

	// 統計情報の取得
	stats, err := db.GetSyncStats()
	if err != nil {
		t.Errorf("統計情報取得が失敗: %v", err)
	}
	if stats == nil {
		t.Error("統計情報がnilです")
	}
}

// TestNewSyncDB_InvalidPath は無効なパスでのデータベース作成をテスト
func TestNewSyncDB_InvalidPath(t *testing.T) {
	// 権限のないディレクトリでのテスト
	_, err := NewSyncDB("/root/invalid/test.db", NormalSync)
	if err == nil {
		t.Error("権限のないディレクトリでエラーが発生しませんでした")
	}

	// 空のパスでのテスト
	_, err = NewSyncDB("", NormalSync)
	if err == nil {
		t.Error("空のパスでエラーが発生しませんでした")
	}
}

// TestAddFile_InvalidData は無効なデータでのファイル追加をテスト
func TestAddFile_InvalidData(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := NewSyncDB(dbPath, NormalSync)
	if err != nil {
		t.Fatalf("データベース作成が失敗: %v", err)
	}
	defer db.Close()

	// 無効なJSONデータを直接データベースに書き込む
	err = db.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(fileSyncBucket)
		if bucket == nil {
			return fmt.Errorf("バケットが見つかりません")
		}

		// 無効なJSONデータを書き込み
		invalidJSON := []byte("invalid json data")
		return bucket.Put([]byte("invalid.txt"), invalidJSON)
	})

	if err != nil {
		t.Fatalf("無効なデータの書き込みに失敗: %v", err)
	}

	// 無効なデータを取得しようとする
	_, err = db.GetFile("invalid.txt")
	if err == nil {
		t.Error("無効なJSONデータでエラーが発生しませんでした")
	}
}

// TestUpdateFileStatus_NonExistentFile は存在しないファイルのステータス更新をテスト
func TestUpdateFileStatus_NonExistentFile(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := NewSyncDB(dbPath, NormalSync)
	if err != nil {
		t.Fatalf("データベース作成が失敗: %v", err)
	}
	defer db.Close()

	// 存在しないファイルのステータスを更新
	err = db.UpdateFileStatus("nonexistent.txt", StatusFailed, "test error")
	if err != nil {
		t.Errorf("存在しないファイルのステータス更新に失敗: %v", err)
	}

	// ファイルが作成されていることを確認
	fileInfo, err := db.GetFile("nonexistent.txt")
	if err != nil {
		t.Errorf("作成されたファイルの取得に失敗: %v", err)
	}

	if fileInfo.Status != StatusFailed {
		t.Errorf("ステータスが正しく設定されていません: 期待=%s, 実際=%s", StatusFailed, fileInfo.Status)
	}
}

// TestUpdateFileHash_NonExistentFile は存在しないファイルのハッシュ更新をテスト
func TestUpdateFileHash_NonExistentFile(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := NewSyncDB(dbPath, NormalSync)
	if err != nil {
		t.Fatalf("データベース作成が失敗: %v", err)
	}
	defer db.Close()

	// 存在しないファイルのハッシュを更新
	err = db.UpdateFileHash("nonexistent.txt", "source-hash", "dest-hash")
	if err == nil {
		t.Error("存在しないファイルのハッシュ更新でエラーが発生しませんでした")
	}
}

// TestIncrementFailCount_NonExistentFile は存在しないファイルの失敗回数増加をテスト
func TestIncrementFailCount_NonExistentFile(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := NewSyncDB(dbPath, NormalSync)
	if err != nil {
		t.Fatalf("データベース作成が失敗: %v", err)
	}
	defer db.Close()

	// 存在しないファイルの失敗回数を増加
	_, err = db.IncrementFailCount("nonexistent.txt")
	if err == nil {
		t.Error("存在しないファイルの失敗回数増加でエラーが発生しませんでした")
	}
}

// TestDatabaseConcurrentAccess は並行アクセスのテスト
func TestDatabaseConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := NewSyncDB(dbPath, NormalSync)
	if err != nil {
		t.Fatalf("データベース作成が失敗: %v", err)
	}
	defer db.Close()

	const numGoroutines = 10
	const operationsPerGoroutine = 100
	done := make(chan bool, numGoroutines)

	// 複数のゴルーチンで同時に操作を実行
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()
			for j := 0; j < operationsPerGoroutine; j++ {
				filePath := fmt.Sprintf("goroutine_%d/file_%d.txt", id, j)

				// ファイルを追加
				fileInfo := FileInfo{
					Path:         filePath,
					Size:         int64(j * 1024),
					ModTime:      time.Now(),
					Status:       StatusSuccess,
					SourceHash:   fmt.Sprintf("hash_%d_%d", id, j),
					DestHash:     fmt.Sprintf("hash_%d_%d", id, j),
					FailCount:    0,
					LastSyncTime: time.Now(),
					LastError:    "",
				}

				if err := db.AddFile(fileInfo); err != nil {
					t.Errorf("並行処理でのファイル追加エラー: %v", err)
					return
				}

				// ファイルを取得
				if _, err := db.GetFile(filePath); err != nil {
					t.Errorf("並行処理でのファイル取得エラー: %v", err)
					return
				}

				// ステータスを更新
				if err := db.UpdateFileStatus(filePath, StatusFailed, "test error"); err != nil {
					t.Errorf("並行処理でのステータス更新エラー: %v", err)
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
	allFiles, err := db.GetAllFiles()
	if err != nil {
		t.Fatalf("ファイル一覧の取得に失敗: %v", err)
	}

	expectedCount := numGoroutines * operationsPerGoroutine
	if len(allFiles) != expectedCount {
		t.Errorf("並行処理後のファイル数が期待値と異なります: 期待=%d, 実際=%d", expectedCount, len(allFiles))
	}
}

// TestDatabaseErrorRecovery はエラーからの回復をテスト
func TestDatabaseErrorRecovery(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := NewSyncDB(dbPath, NormalSync)
	if err != nil {
		t.Fatalf("データベース作成が失敗: %v", err)
	}
	defer db.Close()

	// 正常なファイルを追加
	fileInfo := FileInfo{
		Path:         "test.txt",
		Size:         1024,
		ModTime:      time.Now(),
		Status:       StatusSuccess,
		SourceHash:   "test-hash",
		DestHash:     "test-hash",
		FailCount:    0,
		LastSyncTime: time.Now(),
		LastError:    "",
	}

	if err := db.AddFile(fileInfo); err != nil {
		t.Fatalf("ファイル追加に失敗: %v", err)
	}

	// データベースを閉じる
	db.Close()

	// 同じデータベースを再オープン
	db2, err := NewSyncDB(dbPath, NormalSync)
	if err != nil {
		t.Fatalf("データベースの再オープンに失敗: %v", err)
	}
	defer db2.Close()

	// ファイルが復旧されていることを確認
	retrievedFile, err := db2.GetFile("test.txt")
	if err != nil {
		t.Fatalf("ファイルの復旧確認に失敗: %v", err)
	}

	if retrievedFile.Path != fileInfo.Path {
		t.Errorf("復旧されたファイルのパスが一致しません: 期待=%s, 実際=%s", fileInfo.Path, retrievedFile.Path)
	}
}

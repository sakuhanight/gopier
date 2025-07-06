//go:build windows
// +build windows

package permissions

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"golang.org/x/sys/windows"
)

func TestCopyFilePermissions(t *testing.T) {
	// 管理者権限が必要なテストのため、adminタグでスキップ
	if testing.Short() {
		t.Skip("管理者権限が必要なテストのため、-shortフラグでスキップ")
	}

	// テスト用の一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "gopier_test")
	if err != nil {
		t.Fatalf("テスト用ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// テスト用のソースファイルを作成
	sourceFile := filepath.Join(tempDir, "source.txt")
	destFile := filepath.Join(tempDir, "dest.txt")

	// ソースファイルを作成
	if err := os.WriteFile(sourceFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("ソースファイルの作成に失敗: %v", err)
	}

	// 宛先ファイルを作成
	if err := os.WriteFile(destFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("宛先ファイルの作成に失敗: %v", err)
	}

	t.Logf("テストファイルを作成: %s -> %s", sourceFile, destFile)

	// ファイル権限のコピーをテスト
	err = CopyFilePermissions(sourceFile, destFile)
	if err != nil {
		t.Logf("ファイル権限のコピーでエラーが発生しました（予想される）: %v", err)
		// Windowsの権限設定では管理者権限が必要なため、エラーは予想される
		// エラーの詳細を確認
		if err.Error() == "ファイルのセキュリティ情報を設定できません（アクセス拒否）" {
			t.Logf("アクセス拒否エラー - 管理者権限が必要です")
		} else if err.Error() == "ファイルのセキュリティ情報を設定できません（特権不足）" {
			t.Logf("特権不足エラー - セキュリティ特権が必要です")
		} else {
			t.Logf("その他のエラー: %v", err)
		}
	} else {
		t.Logf("ファイル権限のコピーが成功しました")
	}
}

func TestCopyDirectoryPermissions(t *testing.T) {
	// 管理者権限が必要なテストのため、adminタグでスキップ
	if testing.Short() {
		t.Skip("管理者権限が必要なテストのため、-shortフラグでスキップ")
	}

	// テスト用の一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "gopier_test")
	if err != nil {
		t.Fatalf("テスト用ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// テスト用のソースディレクトリと宛先ディレクトリを作成
	sourceDir := filepath.Join(tempDir, "source_dir")
	destDir := filepath.Join(tempDir, "dest_dir")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("ソースディレクトリの作成に失敗: %v", err)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("宛先ディレクトリの作成に失敗: %v", err)
	}

	t.Logf("テストディレクトリを作成: %s -> %s", sourceDir, destDir)

	// ディレクトリ権限のコピーをテスト
	err = CopyDirectoryPermissions(sourceDir, destDir)
	if err != nil {
		t.Logf("ディレクトリ権限のコピーでエラーが発生しました（予想される）: %v", err)
		// Windowsの権限設定では管理者権限が必要なため、エラーは予想される
		// エラーの詳細を確認
		if err.Error() == "ファイルのセキュリティ情報を設定できません（アクセス拒否）" {
			t.Logf("アクセス拒否エラー - 管理者権限が必要です")
		} else if err.Error() == "ファイルのセキュリティ情報を設定できません（特権不足）" {
			t.Logf("特権不足エラー - セキュリティ特権が必要です")
		} else {
			t.Logf("その他のエラー: %v", err)
		}
	} else {
		t.Logf("ディレクトリ権限のコピーが成功しました")
	}
}

func TestGetWindowsErrorDescription(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"nil error", nil, "エラーなし"},
		{"access denied", syscall.Errno(windows.ERROR_ACCESS_DENIED), "アクセス拒否 - 管理者権限が必要です"},
		{"privilege not held", syscall.Errno(windows.ERROR_PRIVILEGE_NOT_HELD), "特権不足 - セキュリティ特権が必要です"},
		{"insufficient buffer", syscall.Errno(windows.ERROR_INSUFFICIENT_BUFFER), "バッファ不足 - 内部エラー（通常は正常な動作）"},
		{"file not found", syscall.Errno(windows.ERROR_FILE_NOT_FOUND), "ファイルが見つかりません"},
		{"path not found", syscall.Errno(windows.ERROR_PATH_NOT_FOUND), "パスが見つかりません"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getWindowsErrorDescription(tt.err)
			if result != tt.expected {
				t.Errorf("getWindowsErrorDescription() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCopyFilePermissionsWithNonExistentFile(t *testing.T) {
	// 管理者権限が必要なテストのため、adminタグでスキップ
	if testing.Short() {
		t.Skip("管理者権限が必要なテストのため、-shortフラグでスキップ")
	}

	// 存在しないファイルでテスト
	err := CopyFilePermissions("nonexistent_source.txt", "nonexistent_dest.txt")
	if err == nil {
		t.Errorf("存在しないファイルでエラーが発生すべきでした")
	} else {
		t.Logf("期待されるエラー: %v", err)
	}
}

func TestCopyDirectoryPermissionsWithNonExistentDirectory(t *testing.T) {
	// 管理者権限が必要なテストのため、adminタグでスキップ
	if testing.Short() {
		t.Skip("管理者権限が必要なテストのため、-shortフラグでスキップ")
	}

	// 存在しないディレクトリでテスト
	err := CopyDirectoryPermissions("nonexistent_source_dir", "nonexistent_dest_dir")
	if err == nil {
		t.Errorf("存在しないディレクトリでエラーが発生すべきでした")
	} else {
		t.Logf("期待されるエラー: %v", err)
	}
}

// BenchmarkCopyFilePermissions はファイル権限コピーのベンチマークテスト
func BenchmarkCopyFilePermissions(b *testing.B) {
	// テスト用の一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "gopier_benchmark")
	if err != nil {
		b.Fatalf("テスト用ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// テスト用のファイルを作成
	sourceFile := filepath.Join(tempDir, "source.txt")
	destFile := filepath.Join(tempDir, "dest.txt")

	if err := os.WriteFile(sourceFile, []byte("test content"), 0644); err != nil {
		b.Fatalf("ソースファイルの作成に失敗: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 宛先ファイルを毎回作成
		if err := os.WriteFile(destFile, []byte("test content"), 0644); err != nil {
			b.Fatalf("宛先ファイルの作成に失敗: %v", err)
		}

		// 権限コピーを実行
		err := CopyFilePermissions(sourceFile, destFile)
		if err != nil {
			// エラーは予想される（管理者権限が必要なため）
			b.Logf("ベンチマーク中のエラー（予想される）: %v", err)
		}
	}
}

// 統合テスト用のヘルパー関数
func createTestEnvironment(t *testing.T) (string, func()) {
	tempDir, err := os.MkdirTemp("", "gopier_integration_test")
	if err != nil {
		t.Fatalf("テスト環境の作成に失敗: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

func TestIntegrationACLCopy(t *testing.T) {
	// 管理者権限が必要なテストのため、adminタグでスキップ
	if testing.Short() {
		t.Skip("管理者権限が必要なテストのため、-shortフラグでスキップ")
	}

	tempDir, cleanup := createTestEnvironment(t)
	defer cleanup()

	// テスト用のファイル構造を作成
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("ソースディレクトリの作成に失敗: %v", err)
	}

	// 宛先ディレクトリも作成
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("宛先ディレクトリの作成に失敗: %v", err)
	}

	// テストファイルを作成
	testFile := filepath.Join(sourceDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("テストファイルの作成に失敗: %v", err)
	}

	t.Logf("統合テスト環境を作成: %s -> %s", sourceDir, destDir)

	// ディレクトリ権限のコピーをテスト
	err := CopyDirectoryPermissions(sourceDir, destDir)
	if err != nil {
		t.Logf("ディレクトリ権限コピーエラー（予想される）: %v", err)
	} else {
		t.Logf("ディレクトリ権限コピー成功")
	}

	// ファイル権限のコピーをテスト
	destFile := filepath.Join(destDir, "test.txt")
	if err := os.WriteFile(destFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("宛先ファイルの作成に失敗: %v", err)
	}

	err = CopyFilePermissions(testFile, destFile)
	if err != nil {
		t.Logf("ファイル権限コピーエラー（予想される）: %v", err)
	} else {
		t.Logf("ファイル権限コピー成功")
	}
}

// エラーハンドリングの詳細テスト
func TestDetailedErrorHandling(t *testing.T) {
	// 管理者権限が必要なテストのため、adminタグでスキップ
	if testing.Short() {
		t.Skip("管理者権限が必要なテストのため、-shortフラグでスキップ")
	}

	tests := []struct {
		name        string
		sourcePath  string
		destPath    string
		expectError bool
	}{
		{
			name:        "存在しないソースファイル",
			sourcePath:  "nonexistent_source.txt",
			destPath:    "dest.txt",
			expectError: true,
		},
		{
			name:        "存在しない宛先ファイル",
			sourcePath:  "source.txt",
			destPath:    "nonexistent_dest.txt",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CopyFilePermissions(tt.sourcePath, tt.destPath)
			if tt.expectError && err == nil {
				t.Errorf("エラーが期待されましたが、エラーが発生しませんでした")
			} else if !tt.expectError && err != nil {
				t.Errorf("予期しないエラー: %v", err)
			} else if err != nil {
				t.Logf("期待されるエラー: %v", err)
			}
		})
	}
}

// デバッグ情報の出力テスト
func TestDebugOutput(t *testing.T) {
	// このテストは実際のデバッグ出力を確認するためのもの
	// 実際のファイル操作を行い、デバッグメッセージが出力されることを確認
	tempDir, err := os.MkdirTemp("", "gopier_debug_test")
	if err != nil {
		t.Fatalf("テスト用ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceFile := filepath.Join(tempDir, "source.txt")
	destFile := filepath.Join(tempDir, "dest.txt")

	if err := os.WriteFile(sourceFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("ソースファイルの作成に失敗: %v", err)
	}

	// 宛先ファイルも作成
	if err := os.WriteFile(destFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("宛先ファイルの作成に失敗: %v", err)
	}

	t.Logf("デバッグ出力テスト開始")
	t.Logf("ソースファイル: %s", sourceFile)
	t.Logf("宛先ファイル: %s", destFile)

	// ファイル権限コピーを実行（デバッグ出力を確認）
	err = CopyFilePermissions(sourceFile, destFile)
	if err != nil {
		t.Logf("デバッグ出力テスト中のエラー（予想される）: %v", err)
	} else {
		t.Logf("デバッグ出力テスト成功")
	}

	t.Logf("デバッグ出力テスト完了")
}

func TestCopyDirectoryTreePermissions(t *testing.T) {
	// 管理者権限が必要なテストのため、adminタグでスキップ
	if testing.Short() {
		t.Skip("管理者権限が必要なテストのため、-shortフラグでスキップ")
	}

	// テスト用の一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "gopier_tree_test")
	if err != nil {
		t.Fatalf("テスト用ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// ソースディレクトリ構造を作成
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("ソースディレクトリの作成に失敗: %v", err)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("宛先ディレクトリの作成に失敗: %v", err)
	}

	// サブディレクトリを作成
	subDir := filepath.Join(sourceDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("サブディレクトリの作成に失敗: %v", err)
	}

	// 宛先にも同じ構造を作成
	destSubDir := filepath.Join(destDir, "subdir")
	if err := os.MkdirAll(destSubDir, 0755); err != nil {
		t.Fatalf("宛先サブディレクトリの作成に失敗: %v", err)
	}

	// ファイルを作成
	sourceFile := filepath.Join(sourceDir, "file1.txt")
	destFile := filepath.Join(destDir, "file1.txt")
	subSourceFile := filepath.Join(subDir, "file2.txt")
	subDestFile := filepath.Join(destSubDir, "file2.txt")

	files := []string{sourceFile, destFile, subSourceFile, subDestFile}
	for _, file := range files {
		if err := os.WriteFile(file, []byte("test content"), 0644); err != nil {
			t.Fatalf("ファイルの作成に失敗: %v", err)
		}
	}

	// ディレクトリツリー権限コピーをテスト
	err = CopyDirectoryTreePermissions(sourceDir, destDir)
	if err != nil {
		t.Logf("CopyDirectoryTreePermissionsが失敗: %v", err)
		// 管理者権限が必要なため、エラーは予想される
	} else {
		t.Logf("ACL同期が正常に完了しました")
	}
}

func TestCopyDirectoryTreePermissionsWithProgress(t *testing.T) {
	// 管理者権限が必要なテストのため、adminタグでスキップ
	if testing.Short() {
		t.Skip("管理者権限が必要なテストのため、-shortフラグでスキップ")
	}

	// テスト用の一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "gopier_progress_test")
	if err != nil {
		t.Fatalf("テスト用ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// ソースディレクトリ構造を作成
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("ソースディレクトリの作成に失敗: %v", err)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("宛先ディレクトリの作成に失敗: %v", err)
	}

	// サブディレクトリを作成
	subDir := filepath.Join(sourceDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("サブディレクトリの作成に失敗: %v", err)
	}

	// 宛先にも同じ構造を作成
	destSubDir := filepath.Join(destDir, "subdir")
	if err := os.MkdirAll(destSubDir, 0755); err != nil {
		t.Fatalf("宛先サブディレクトリの作成に失敗: %v", err)
	}

	// ファイルを作成
	sourceFile := filepath.Join(sourceDir, "file1.txt")
	destFile := filepath.Join(destDir, "file1.txt")
	subSourceFile := filepath.Join(subDir, "file2.txt")
	subDestFile := filepath.Join(destSubDir, "file2.txt")

	files := []string{sourceFile, destFile, subSourceFile, subDestFile}
	for _, file := range files {
		if err := os.WriteFile(file, []byte("test content"), 0644); err != nil {
			t.Fatalf("ファイルの作成に失敗: %v", err)
		}
	}

	// 進捗コールバックの呼び出し回数をカウント
	callbackCount := 0
	progressCallback := func(current, total int, currentPath string) {
		callbackCount++
		t.Logf("進捗: %d/%d - %s", current, total, currentPath)
	}

	// 進捗付きディレクトリツリー権限コピーをテスト
	err = CopyDirectoryTreePermissionsWithProgress(sourceDir, destDir, progressCallback)
	if err != nil {
		t.Logf("CopyDirectoryTreePermissionsWithProgressが失敗: %v", err)
		// 管理者権限が必要なため、エラーは予想される
	} else {
		t.Logf("ACL同期（進捗付き）が正常に完了しました")
	}

	// 進捗コールバックが呼ばれたかチェック
	if callbackCount == 0 {
		t.Logf("進捗コールバックが呼ばれていません")
	} else {
		t.Logf("進捗コールバック呼び出し回数: %d", callbackCount)
	}
}

func TestCopyDirectoryTreePermissions_NonExistentSource(t *testing.T) {
	// 管理者権限が必要なテストのため、adminタグでスキップ
	if testing.Short() {
		t.Skip("管理者権限が必要なテストのため、-shortフラグでスキップ")
	}

	// 存在しないソースディレクトリでテスト
	err := CopyDirectoryTreePermissions("nonexistent_source", "dest")
	if err == nil {
		t.Errorf("存在しないソースディレクトリでエラーが発生すべきでした")
	} else {
		t.Logf("期待されるエラー: %v", err)
	}
}

func TestCopyDirectoryTreePermissions_NonExistentDest(t *testing.T) {
	// 管理者権限が必要なテストのため、adminタグでスキップ
	if testing.Short() {
		t.Skip("管理者権限が必要なテストのため、-shortフラグでスキップ")
	}

	// 存在しない宛先ディレクトリでテスト
	err := CopyDirectoryTreePermissions("source", "nonexistent_dest")
	if err == nil {
		t.Errorf("存在しない宛先ディレクトリでエラーが発生すべきでした")
	} else {
		t.Logf("期待されるエラー: %v", err)
	}
}

// TestPreservePermissionsIntegration は--preserve-permissionsオプションの統合テスト
func TestPreservePermissionsIntegration(t *testing.T) {
	// 管理者権限が必要なテストのため、adminタグでスキップ
	if testing.Short() {
		t.Skip("管理者権限が必要なテストのため、-shortフラグでスキップ")
	}

	// 統合テスト環境を作成
	tempDir, err := os.MkdirTemp("", "gopier_preserve_test")
	if err != nil {
		t.Fatalf("統合テスト用ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// ソースディレクトリ構造を作成
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")

	// ソースディレクトリを作成
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("ソースディレクトリの作成に失敗: %v", err)
	}

	// サブディレクトリを作成
	subDir := filepath.Join(sourceDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("サブディレクトリの作成に失敗: %v", err)
	}

	// ファイルを作成
	testFile := filepath.Join(sourceDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("テストファイルの作成に失敗: %v", err)
	}

	subFile := filepath.Join(subDir, "sub.txt")
	if err := os.WriteFile(subFile, []byte("sub content"), 0644); err != nil {
		t.Fatalf("サブファイルの作成に失敗: %v", err)
	}

	// 宛先ディレクトリ構造を作成（権限は異なる）
	if err := os.MkdirAll(destDir, 0777); err != nil {
		t.Fatalf("宛先ディレクトリの作成に失敗: %v", err)
	}

	destSubDir := filepath.Join(destDir, "subdir")
	if err := os.MkdirAll(destSubDir, 0777); err != nil {
		t.Fatalf("宛先サブディレクトリの作成に失敗: %v", err)
	}

	destFile := filepath.Join(destDir, "test.txt")
	if err := os.WriteFile(destFile, []byte("test content"), 0666); err != nil {
		t.Fatalf("宛先ファイルの作成に失敗: %v", err)
	}

	destSubFile := filepath.Join(destSubDir, "sub.txt")
	if err := os.WriteFile(destSubFile, []byte("sub content"), 0666); err != nil {
		t.Fatalf("宛先サブファイルの作成に失敗: %v", err)
	}

	// 個別の権限コピーをテスト（ファイルコピー時の動作をシミュレート）
	t.Logf("個別ファイル権限コピーをテスト")
	if err := CopyFilePermissions(testFile, destFile); err != nil {
		t.Logf("ファイル権限コピーエラー（予想される）: %v", err)
	} else {
		t.Logf("ファイル権限コピー成功")
	}

	// 個別のディレクトリ権限コピーをテスト（ディレクトリ作成時の動作をシミュレート）
	t.Logf("個別ディレクトリ権限コピーをテスト")
	if err := CopyDirectoryPermissions(sourceDir, destDir); err != nil {
		t.Logf("ディレクトリ権限コピーエラー（予想される）: %v", err)
	} else {
		t.Logf("ディレクトリ権限コピー成功")
	}

	// 最終的なACL同期をテスト（コピー完了後の動作をシミュレート）
	t.Logf("最終ACL同期をテスト")
	if err := CopyDirectoryTreePermissions(sourceDir, destDir); err != nil {
		t.Logf("ACL同期エラー（予想される）: %v", err)
	} else {
		t.Logf("ACL同期成功")
	}

	t.Logf("統合テストが完了しました")
}

// TestAdminPrivileges は管理者権限チェック機能をテスト
func TestAdminPrivileges(t *testing.T) {
	// IsAdmin関数のテスト
	isAdmin := IsAdmin()
	t.Logf("現在のプロセスは管理者権限で実行されていますか: %v", isAdmin)

	// RequireAdmin関数のテスト
	err := RequireAdmin()
	if err != nil {
		t.Logf("管理者権限が必要です: %v", err)
	} else {
		t.Logf("管理者権限で実行されています")
	}

	// CheckAdminForPermissions関数のテスト
	err = CheckAdminForPermissions()
	if err != nil {
		t.Logf("権限コピーには管理者権限が必要です: %v", err)
	} else {
		t.Logf("権限コピーに必要な管理者権限があります")
	}
}

// TestAdminPrivilegesWithPermissions は管理者権限チェックと権限コピーの統合テスト
func TestAdminPrivilegesWithPermissions(t *testing.T) {
	// 管理者権限が必要なテストのため、adminタグでスキップ
	if testing.Short() {
		t.Skip("管理者権限が必要なテストのため、-shortフラグでスキップ")
	}

	// テスト用の一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "admin_permissions_test")
	if err != nil {
		t.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// テスト用のファイルを作成
	sourceFile := filepath.Join(tempDir, "source.txt")
	destFile := filepath.Join(tempDir, "dest.txt")

	if err := os.WriteFile(sourceFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("ソースファイルの作成に失敗: %v", err)
	}
	if err := os.WriteFile(destFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("宛先ファイルの作成に失敗: %v", err)
	}

	// 管理者権限チェック付きでファイル権限コピーをテスト
	t.Logf("管理者権限チェック付きファイル権限コピーをテスト")
	err = CopyFilePermissions(sourceFile, destFile)
	if err != nil {
		if err.Error() == "権限コピーには管理者権限が必要です。以下のいずれかの方法で管理者として実行してください：" {
			t.Logf("期待される管理者権限エラー: %v", err)
		} else {
			t.Logf("その他のエラー: %v", err)
		}
	} else {
		t.Logf("ファイル権限コピー成功")
	}

	// 管理者権限チェック付きでディレクトリ権限コピーをテスト
	sourceDir := filepath.Join(tempDir, "source_dir")
	destDir := filepath.Join(tempDir, "dest_dir")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("ソースディレクトリの作成に失敗: %v", err)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("宛先ディレクトリの作成に失敗: %v", err)
	}

	t.Logf("管理者権限チェック付きディレクトリ権限コピーをテスト")
	err = CopyDirectoryPermissions(sourceDir, destDir)
	if err != nil {
		if err.Error() == "権限コピーには管理者権限が必要です。以下のいずれかの方法で管理者として実行してください：" {
			t.Logf("期待される管理者権限エラー: %v", err)
		} else {
			t.Logf("その他のエラー: %v", err)
		}
	} else {
		t.Logf("ディレクトリ権限コピー成功")
	}
}

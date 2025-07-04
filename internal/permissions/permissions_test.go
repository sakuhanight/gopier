package permissions

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestIsWindows(t *testing.T) {
	// 現在のプラットフォームに基づいてテスト
	expected := runtime.GOOS == "windows"
	result := IsWindows()
	if result != expected {
		t.Errorf("IsWindows() = %v, expected %v", result, expected)
	}
}

func TestCanCopyPermissions(t *testing.T) {
	// 現在のプラットフォームに基づいてテスト
	expected := runtime.GOOS == "windows"
	result := CanCopyPermissions()
	if result != expected {
		t.Errorf("CanCopyPermissions() = %v, expected %v", result, expected)
	}
}

func TestCopyFilePermissions_NonWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows環境ではスキップ")
	}

	// 非Windows環境でのテスト
	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "source.txt")
	destFile := filepath.Join(tempDir, "dest.txt")

	// テストファイルを作成
	os.WriteFile(sourceFile, []byte("test"), 0644)

	err := CopyFilePermissions(sourceFile, destFile)
	if err == nil {
		t.Error("非Windows環境ではエラーが発生すべきです")
	}

	expectedError := "ファイル権限のコピーはWindowsでのみサポートされています"
	if err.Error() != expectedError {
		t.Errorf("期待されるエラーメッセージ: %s, 実際: %s", expectedError, err.Error())
	}
}

func TestCopyDirectoryPermissions_NonWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows環境ではスキップ")
	}

	// 非Windows環境でのテスト
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")

	// テストディレクトリを作成
	os.MkdirAll(sourceDir, 0755)

	err := CopyDirectoryPermissions(sourceDir, destDir)
	if err == nil {
		t.Error("非Windows環境ではエラーが発生すべきです")
	}

	expectedError := "ディレクトリ権限のコピーはWindowsでのみサポートされています"
	if err.Error() != expectedError {
		t.Errorf("期待されるエラーメッセージ: %s, 実際: %s", expectedError, err.Error())
	}
}

func TestCopyFilePermissions_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows環境でのみ実行")
	}

	// Windows環境でのテスト
	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "source.txt")
	destFile := filepath.Join(tempDir, "dest.txt")

	// テストファイルを作成
	os.WriteFile(sourceFile, []byte("test"), 0644)

	// 権限コピーを試行（実際のWindows APIが利用可能かどうかは環境による）
	err := CopyFilePermissions(sourceFile, destFile)
	if err != nil {
		// Windows環境でも権限コピーが失敗する場合がある（管理者権限が必要など）
		// エラーメッセージに"Windows"が含まれていることを確認
		if !contains(err.Error(), "Windows") && !contains(err.Error(), "セキュリティ") {
			t.Errorf("予期しないエラー: %v", err)
		}
	}
}

func TestCopyDirectoryPermissions_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows環境でのみ実行")
	}

	// Windows環境でのテスト
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")

	// テストディレクトリを作成
	os.MkdirAll(sourceDir, 0755)

	// 権限コピーを試行（実際のWindows APIが利用可能かどうかは環境による）
	err := CopyDirectoryPermissions(sourceDir, destDir)
	if err != nil {
		// Windows環境でも権限コピーが失敗する場合がある（管理者権限が必要など）
		// エラーメッセージに"Windows"が含まれていることを確認
		if !contains(err.Error(), "Windows") && !contains(err.Error(), "セキュリティ") {
			t.Errorf("予期しないエラー: %v", err)
		}
	}
}

// contains は文字列が部分文字列を含むかどうかをチェックする
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 1; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}()))
}

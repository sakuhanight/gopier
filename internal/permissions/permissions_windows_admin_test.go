//go:build windows
// +build windows

package permissions

import (
	"os"
	"path/filepath"
	"testing"
)

// 管理者権限が必要なテストのみを含むファイル
// これらのテストは管理者権限で実行する必要があります

func TestCopyFilePermissionsWithAdmin(t *testing.T) {
	// 管理者権限チェック
	if !IsAdmin() {
		t.Skip("管理者権限が必要なテストのため、管理者として実行してください")
	}

	// テスト用の一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "gopier_admin_test")
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

	t.Logf("管理者権限テストファイルを作成: %s -> %s", sourceFile, destFile)

	// ファイル権限のコピーをテスト
	err = CopyFilePermissions(sourceFile, destFile)
	if err != nil {
		t.Errorf("管理者権限で実行中なのにファイル権限コピーが失敗: %v", err)
	} else {
		t.Logf("ファイル権限のコピーが成功しました")
	}
}

func TestCopyDirectoryPermissionsWithAdmin(t *testing.T) {
	// 管理者権限チェック
	if !IsAdmin() {
		t.Skip("管理者権限が必要なテストのため、管理者として実行してください")
	}

	// テスト用の一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "gopier_admin_test")
	if err != nil {
		t.Fatalf("テスト用ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// テスト用のディレクトリを作成
	sourceDir := filepath.Join(tempDir, "source_dir")
	destDir := filepath.Join(tempDir, "dest_dir")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("ソースディレクトリの作成に失敗: %v", err)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("宛先ディレクトリの作成に失敗: %v", err)
	}

	t.Logf("管理者権限テストディレクトリを作成: %s -> %s", sourceDir, destDir)

	// ディレクトリ権限のコピーをテスト
	err = CopyDirectoryPermissions(sourceDir, destDir)
	if err != nil {
		t.Errorf("管理者権限で実行中なのにディレクトリ権限コピーが失敗: %v", err)
	} else {
		t.Logf("ディレクトリ権限のコピーが成功しました")
	}
}

func TestCopyDirectoryTreePermissionsWithAdmin(t *testing.T) {
	// 管理者権限チェック
	if !IsAdmin() {
		t.Skip("管理者権限が必要なテストのため、管理者として実行してください")
	}

	// テスト用の一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "gopier_admin_tree_test")
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
		t.Errorf("管理者権限で実行中なのにディレクトリツリー権限コピーが失敗: %v", err)
	} else {
		t.Logf("ディレクトリツリー権限コピーが成功しました")
	}
}

func TestCopyDirectoryTreePermissionsWithProgressWithAdmin(t *testing.T) {
	// 管理者権限チェック
	if !IsAdmin() {
		t.Skip("管理者権限が必要なテストのため、管理者として実行してください")
	}

	// テスト用の一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "gopier_admin_progress_test")
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
		t.Errorf("管理者権限で実行中なのに進捗付きディレクトリツリー権限コピーが失敗: %v", err)
	} else {
		t.Logf("進捗付きディレクトリツリー権限コピーが成功しました")
	}

	// 進捗コールバックが呼ばれたかチェック
	if callbackCount == 0 {
		t.Logf("進捗コールバックが呼ばれていません")
	} else {
		t.Logf("進捗コールバック呼び出し回数: %d", callbackCount)
	}
}

func TestUACElevationWithAdmin(t *testing.T) {
	// 管理者権限チェック
	if !IsAdmin() {
		t.Skip("管理者権限が必要なテストのため、管理者として実行してください")
	}

	// 管理者権限で実行されている場合のUAC権限昇格テスト
	t.Logf("管理者権限で実行中: %v", IsAdmin())

	// ElevateWithUAC関数のテスト（管理者権限で実行中なので何もしないはず）
	err := ElevateWithUAC()
	if err != nil {
		t.Errorf("管理者権限で実行中なのにUAC権限昇格エラーが発生: %v", err)
	} else {
		t.Logf("UAC権限昇格テスト成功（管理者権限で実行中のため何もしない）")
	}

	// ElevateForPermissions関数のテスト
	err = ElevateForPermissions()
	if err != nil {
		t.Errorf("管理者権限で実行中なのに権限昇格エラーが発生: %v", err)
	} else {
		t.Logf("権限昇格テスト成功（管理者権限で実行中のため何もしない）")
	}

	// CheckAndElevateForPermissions関数のテスト
	err = CheckAndElevateForPermissions()
	if err != nil {
		t.Errorf("管理者権限で実行中なのに権限昇格エラーが発生: %v", err)
	} else {
		t.Logf("権限昇格確認テスト成功（管理者権限で実行中のため何もしない）")
	}
}

func TestIntegrationWithAdmin(t *testing.T) {
	// 管理者権限チェック
	if !IsAdmin() {
		t.Skip("管理者権限が必要なテストのため、管理者として実行してください")
	}

	// 統合テスト環境を作成
	tempDir, err := os.MkdirTemp("", "gopier_admin_integration_test")
	if err != nil {
		t.Fatalf("統合テスト用ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")

	// ディレクトリ構造を作成
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("ソースディレクトリの作成に失敗: %v", err)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("宛先ディレクトリの作成に失敗: %v", err)
	}

	// ファイルを作成
	sourceFile := filepath.Join(sourceDir, "test.txt")
	destFile := filepath.Join(destDir, "test.txt")

	if err := os.WriteFile(sourceFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("ソースファイルの作成に失敗: %v", err)
	}
	if err := os.WriteFile(destFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("宛先ファイルの作成に失敗: %v", err)
	}

	// 個別ファイル権限コピーをテスト
	t.Logf("個別ファイル権限コピーをテスト")
	err = CopyFilePermissions(sourceFile, destFile)
	if err != nil {
		t.Errorf("管理者権限で実行中なのにファイル権限コピーが失敗: %v", err)
	} else {
		t.Logf("ファイル権限コピー成功")
	}

	// 個別ディレクトリ権限コピーをテスト
	t.Logf("個別ディレクトリ権限コピーをテスト")
	err = CopyDirectoryPermissions(sourceDir, destDir)
	if err != nil {
		t.Errorf("管理者権限で実行中なのにディレクトリ権限コピーが失敗: %v", err)
	} else {
		t.Logf("ディレクトリ権限コピー成功")
	}

	// 最終ACL同期をテスト
	t.Logf("最終ACL同期をテスト")
	err = CopyDirectoryTreePermissions(sourceDir, destDir)
	if err != nil {
		t.Errorf("管理者権限で実行中なのにACL同期が失敗: %v", err)
	} else {
		t.Logf("ACL同期成功")
	}

	t.Logf("管理者権限統合テストが完了しました")
}

func TestAdminPrivilegesWithAdmin(t *testing.T) {
	// 管理者権限チェック
	if !IsAdmin() {
		t.Skip("管理者権限が必要なテストのため、管理者として実行してください")
	}

	// IsAdmin関数のテスト
	isAdmin := IsAdmin()
	if !isAdmin {
		t.Errorf("管理者権限で実行されているはずですが、IsAdmin()がfalseを返しました")
	}
	t.Logf("現在のプロセスは管理者権限で実行されていますか: %v", isAdmin)

	// RequireAdmin関数のテスト
	err := RequireAdmin()
	if err != nil {
		t.Errorf("管理者権限で実行中なのにRequireAdmin()がエラーを返しました: %v", err)
	} else {
		t.Logf("RequireAdmin()テスト成功")
	}

	// CheckAdminForPermissions関数のテスト
	err = CheckAdminForPermissions()
	if err != nil {
		t.Errorf("管理者権限で実行中なのにCheckAdminForPermissions()がエラーを返しました: %v", err)
	} else {
		t.Logf("CheckAdminForPermissions()テスト成功")
	}
}

//go:build windows
// +build windows

package permissions

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows API constants
const (
	OWNER_SECURITY_INFORMATION = 0x00000001
	GROUP_SECURITY_INFORMATION = 0x00000002
	DACL_SECURITY_INFORMATION  = 0x00000004
	SACL_SECURITY_INFORMATION  = 0x00000008
	LABEL_SECURITY_INFORMATION = 0x00000010

	PROTECTED_DACL_SECURITY_INFORMATION   = 0x80000000
	PROTECTED_SACL_SECURITY_INFORMATION   = 0x40000000
	UNPROTECTED_DACL_SECURITY_INFORMATION = 0x20000000
	UNPROTECTED_SACL_SECURITY_INFORMATION = 0x10000000

	SECURITY_DESCRIPTOR_MIN_LENGTH = 20
	SECURITY_DESCRIPTOR_REVISION   = 1
)

// Windows API functions
var (
	advapi32 = windows.NewLazySystemDLL("advapi32.dll")
	shell32  = windows.NewLazySystemDLL("shell32.dll")

	procGetFileSecurityW = advapi32.NewProc("GetFileSecurityW")
	procSetFileSecurityW = advapi32.NewProc("SetFileSecurityW")
	procIsUserAnAdmin    = shell32.NewProc("IsUserAnAdmin")
)

// ACL represents a Windows Access Control List
type ACL struct {
	AclRevision byte
	Sbz1        byte
	AclSize     uint16
	AceCount    uint16
	Sbz2        uint16
}

// SecurityDescriptor represents a Windows security descriptor
type SecurityDescriptor struct {
	Revision byte
	Sbz1     byte
	Control  uint16
	Owner    *windows.SID
	Group    *windows.SID
	Sacl     *ACL
	Dacl     *ACL
}

// getWindowsErrorDescription returns a detailed description of Windows API errors
func getWindowsErrorDescription(err error) string {
	if err == nil {
		return "エラーなし"
	}

	switch err {
	case windows.ERROR_ACCESS_DENIED:
		return "アクセス拒否 - 管理者権限が必要です"
	case windows.ERROR_PRIVILEGE_NOT_HELD:
		return "特権不足 - セキュリティ特権が必要です"
	case windows.ERROR_INSUFFICIENT_BUFFER:
		return "バッファ不足 - 内部エラー（通常は正常な動作）"
	case windows.ERROR_FILE_NOT_FOUND:
		return "ファイルが見つかりません"
	case windows.ERROR_PATH_NOT_FOUND:
		return "パスが見つかりません"
	default:
		return fmt.Sprintf("不明なエラー: %v", err)
	}
}

// CopyFilePermissions copies file permissions from source to destination
func CopyFilePermissions(sourcePath, destPath string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("ファイル権限のコピーはWindowsでのみサポートされています")
	}

	// 管理者権限チェック
	if err := CheckAdminForPermissions(); err != nil {
		return err
	}

	fmt.Printf("DEBUG: CopyFilePermissions called: %s -> %s\n", sourcePath, destPath)

	// ファイルの存在確認
	if _, err := os.Stat(sourcePath); err != nil {
		return fmt.Errorf("ソースファイルが存在しません: %s (エラー: %v)", sourcePath, err)
	}
	if _, err := os.Stat(destPath); err != nil {
		return fmt.Errorf("宛先ファイルが存在しません: %s (エラー: %v)", destPath, err)
	}

	// ソースファイルのセキュリティ記述子を取得
	sourceSD, err := getFileSecurity(sourcePath)
	if err != nil {
		return fmt.Errorf("ソースファイルのセキュリティ記述子取得エラー: %w", err)
	}

	fmt.Printf("DEBUG: Successfully retrieved source security descriptor for %s\n", sourcePath)

	// 宛先ファイルにセキュリティ記述子を設定
	err = setFileSecurity(destPath, sourceSD)
	if err != nil {
		return fmt.Errorf("宛先ファイルのセキュリティ記述子設定エラー: %w", err)
	}

	fmt.Printf("DEBUG: Successfully set security descriptor for %s\n", destPath)
	return nil
}

// getFileSecurity retrieves the security descriptor for a file
func getFileSecurity(filePath string) (*SecurityDescriptor, error) {
	fmt.Printf("DEBUG: getFileSecurity called for: %s\n", filePath)

	// ファイルパスをUTF16に変換
	filePathPtr, pathErr := windows.UTF16PtrFromString(filePath)
	if pathErr != nil {
		return nil, fmt.Errorf("ファイルパスの変換に失敗: %s (エラー: %v)", filePath, pathErr)
	}

	// 最初に必要なサイズを取得
	var sizeNeeded uint32
	ret, _, callErr := procGetFileSecurityW.Call(
		uintptr(unsafe.Pointer(filePathPtr)),
		uintptr(DACL_SECURITY_INFORMATION|OWNER_SECURITY_INFORMATION|GROUP_SECURITY_INFORMATION),
		0,
		0,
		uintptr(unsafe.Pointer(&sizeNeeded)),
	)

	fmt.Printf("DEBUG: GetFileSecurityW first call: ret=%d, sizeNeeded=%d, callErr=%v\n", ret, sizeNeeded, callErr)

	// ERROR_INSUFFICIENT_BUFFERの場合は正常な動作
	if ret == 0 {
		if callErr != nil && callErr.Error() == "The data area passed to a system call is too small." {
			fmt.Printf("DEBUG: ERROR_INSUFFICIENT_BUFFER detected, continuing...\n")
			// これは正常な動作なので続行
		} else {
			err := syscall.GetLastError()
			errDesc := getWindowsErrorDescription(err)
			fmt.Printf("DEBUG: GetFileSecurityW failed with error: %v (%s)\n", err, errDesc)
			return nil, fmt.Errorf("ファイルのセキュリティ情報を取得できません: %s (エラー: %v, 詳細: %s)", filePath, err, errDesc)
		}
	}

	// サイズが0の場合はエラー
	if sizeNeeded == 0 {
		return nil, fmt.Errorf("セキュリティ記述子のサイズが0です: %s", filePath)
	}

	fmt.Printf("DEBUG: Allocating buffer of size %d for %s\n", sizeNeeded, filePath)

	// 適切なサイズのバッファを確保
	buffer := make([]byte, sizeNeeded)
	ret, _, err := procGetFileSecurityW.Call(
		uintptr(unsafe.Pointer(filePathPtr)),
		uintptr(DACL_SECURITY_INFORMATION|OWNER_SECURITY_INFORMATION|GROUP_SECURITY_INFORMATION),
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(sizeNeeded),
		uintptr(unsafe.Pointer(&sizeNeeded)),
	)

	fmt.Printf("DEBUG: GetFileSecurityW second call: ret=%d, err=%v\n", ret, err)

	if ret == 0 {
		lastErr := syscall.GetLastError()
		errDesc := getWindowsErrorDescription(lastErr)
		fmt.Printf("DEBUG: GetFileSecurityW second call failed with last error: %v (%s)\n", lastErr, errDesc)
		return nil, fmt.Errorf("ファイルのセキュリティ情報を取得できません: %s (エラー: %v, 詳細: %s)", filePath, lastErr, errDesc)
	}

	// セキュリティ記述子を解析
	sd := (*SecurityDescriptor)(unsafe.Pointer(&buffer[0]))
	fmt.Printf("DEBUG: Successfully parsed security descriptor for %s\n", filePath)
	return sd, nil
}

// setFileSecurity sets the security descriptor for a file
func setFileSecurity(filePath string, sd *SecurityDescriptor) error {
	fmt.Printf("DEBUG: setFileSecurity called for: %s\n", filePath)

	// ファイルパスをUTF16に変換
	filePathPtr, pathErr := windows.UTF16PtrFromString(filePath)
	if pathErr != nil {
		return fmt.Errorf("ファイルパスの変換に失敗: %s (エラー: %v)", filePath, pathErr)
	}

	// ファイルの存在確認
	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("ファイルが存在しません: %s (エラー: %v)", filePath, err)
	}

	ret, _, callErr := procSetFileSecurityW.Call(
		uintptr(unsafe.Pointer(filePathPtr)),
		uintptr(DACL_SECURITY_INFORMATION|OWNER_SECURITY_INFORMATION|GROUP_SECURITY_INFORMATION),
		uintptr(unsafe.Pointer(sd)),
	)

	fmt.Printf("DEBUG: SetFileSecurityW call: ret=%d, callErr=%v\n", ret, callErr)

	if ret == 0 {
		lastErr := syscall.GetLastError()
		errDesc := getWindowsErrorDescription(lastErr)
		fmt.Printf("DEBUG: SetFileSecurityW failed with last error: %v (%s)\n", lastErr, errDesc)

		// 詳細なエラー情報を提供
		if lastErr == windows.ERROR_ACCESS_DENIED {
			return fmt.Errorf("ファイルのセキュリティ情報を設定できません（アクセス拒否）: %s (エラー: %v, 詳細: %s)", filePath, lastErr, errDesc)
		} else if lastErr == windows.ERROR_PRIVILEGE_NOT_HELD {
			return fmt.Errorf("ファイルのセキュリティ情報を設定できません（特権不足）: %s (エラー: %v, 詳細: %s)", filePath, lastErr, errDesc)
		} else {
			return fmt.Errorf("ファイルのセキュリティ情報を設定できません: %s (エラー: %v, 詳細: %s)", filePath, lastErr, errDesc)
		}
	}

	fmt.Printf("DEBUG: Successfully set security descriptor for %s\n", filePath)
	return nil
}

// CopyDirectoryPermissions copies directory permissions from source to destination
func CopyDirectoryPermissions(sourcePath, destPath string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("ディレクトリ権限のコピーはWindowsでのみサポートされています")
	}

	// 管理者権限チェック
	if err := CheckAdminForPermissions(); err != nil {
		return err
	}

	fmt.Printf("DEBUG: CopyDirectoryPermissions called: %s -> %s\n", sourcePath, destPath)

	// ディレクトリの存在確認
	if _, err := os.Stat(sourcePath); err != nil {
		return fmt.Errorf("ソースディレクトリが存在しません: %s (エラー: %v)", sourcePath, err)
	}
	if _, err := os.Stat(destPath); err != nil {
		return fmt.Errorf("宛先ディレクトリが存在しません: %s (エラー: %v)", destPath, err)
	}

	// ディレクトリのセキュリティ記述子を取得
	sourceSD, err := getFileSecurity(sourcePath)
	if err != nil {
		return fmt.Errorf("ソースディレクトリのセキュリティ記述子取得エラー: %w", err)
	}

	fmt.Printf("DEBUG: Successfully retrieved source directory security descriptor for %s\n", sourcePath)

	// 宛先ディレクトリにセキュリティ記述子を設定
	err = setFileSecurity(destPath, sourceSD)
	if err != nil {
		return fmt.Errorf("宛先ディレクトリのセキュリティ記述子設定エラー: %w", err)
	}

	fmt.Printf("DEBUG: Successfully set directory security descriptor for %s\n", destPath)
	return nil
}

// IsWindows returns true if running on Windows
func IsWindows() bool {
	return runtime.GOOS == "windows"
}

// CanCopyPermissions returns true if permission copying is supported
func CanCopyPermissions() bool {
	return IsWindows()
}

// CopyDirectoryTreePermissions recursively copies permissions from source directory to destination directory
// This function ensures all files and subdirectories have matching ACLs
func CopyDirectoryTreePermissions(sourcePath, destPath string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("ディレクトリツリー権限のコピーはWindowsでのみサポートされています")
	}

	// 管理者権限チェック
	if err := CheckAdminForPermissions(); err != nil {
		return err
	}

	fmt.Printf("DEBUG: CopyDirectoryTreePermissions called: %s -> %s\n", sourcePath, destPath)

	// ディレクトリの存在確認
	if _, err := os.Stat(sourcePath); err != nil {
		return fmt.Errorf("ソースディレクトリが存在しません: %s (エラー: %v)", sourcePath, err)
	}
	if _, err := os.Stat(destPath); err != nil {
		return fmt.Errorf("宛先ディレクトリが存在しません: %s (エラー: %v)", destPath, err)
	}

	// 再帰的に権限をコピー
	return copyDirectoryTreePermissionsRecursive(sourcePath, destPath)
}

// copyDirectoryTreePermissionsRecursive recursively copies permissions for all files and directories
func copyDirectoryTreePermissionsRecursive(sourcePath, destPath string) error {
	fmt.Printf("DEBUG: copyDirectoryTreePermissionsRecursive called: %s -> %s\n", sourcePath, destPath)

	// ソースディレクトリを開く
	entries, err := os.ReadDir(sourcePath)
	if err != nil {
		return fmt.Errorf("ソースディレクトリ(%s)の読み込みエラー: %w", sourcePath, err)
	}

	// 各エントリの処理
	for _, entry := range entries {
		sourceItemPath := filepath.Join(sourcePath, entry.Name())
		destItemPath := filepath.Join(destPath, entry.Name())

		// 宛先アイテムの存在確認
		if _, err := os.Stat(destItemPath); err != nil {
			fmt.Printf("DEBUG: Skipping %s (destination does not exist)\n", destItemPath)
			continue
		}

		// ディレクトリの場合
		if entry.IsDir() {
			fmt.Printf("DEBUG: Processing directory: %s\n", sourceItemPath)

			// ディレクトリの権限をコピー
			if err := CopyDirectoryPermissions(sourceItemPath, destItemPath); err != nil {
				fmt.Printf("WARNING: ディレクトリ権限のコピーに失敗: %s -> %s: %v\n", sourceItemPath, destItemPath, err)
				// エラーが発生しても処理を続行
			} else {
				fmt.Printf("DEBUG: Successfully copied directory permissions: %s\n", destItemPath)
			}

			// 再帰的にサブディレクトリを処理
			if err := copyDirectoryTreePermissionsRecursive(sourceItemPath, destItemPath); err != nil {
				fmt.Printf("WARNING: サブディレクトリ権限のコピーに失敗: %s -> %s: %v\n", sourceItemPath, destItemPath, err)
				// エラーが発生しても処理を続行
			}
		} else {
			// ファイルの場合
			fmt.Printf("DEBUG: Processing file: %s\n", sourceItemPath)

			// ファイルの権限をコピー
			if err := CopyFilePermissions(sourceItemPath, destItemPath); err != nil {
				fmt.Printf("WARNING: ファイル権限のコピーに失敗: %s -> %s: %v\n", sourceItemPath, destItemPath, err)
				// エラーが発生しても処理を続行
			} else {
				fmt.Printf("DEBUG: Successfully copied file permissions: %s\n", destItemPath)
			}
		}
	}

	return nil
}

// CopyDirectoryTreePermissionsWithProgress recursively copies permissions with progress reporting
func CopyDirectoryTreePermissionsWithProgress(sourcePath, destPath string, progressCallback func(current, total int, currentPath string)) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("ディレクトリツリー権限のコピーはWindowsでのみサポートされています")
	}

	// 管理者権限チェック
	if err := CheckAdminForPermissions(); err != nil {
		return err
	}

	fmt.Printf("DEBUG: CopyDirectoryTreePermissionsWithProgress called: %s -> %s\n", sourcePath, destPath)

	// ディレクトリの存在確認
	if _, err := os.Stat(sourcePath); err != nil {
		return fmt.Errorf("ソースディレクトリが存在しません: %s (エラー: %v)", sourcePath, err)
	}
	if _, err := os.Stat(destPath); err != nil {
		return fmt.Errorf("宛先ディレクトリが存在しません: %s (エラー: %v)", destPath, err)
	}

	// 総アイテム数をカウント
	totalItems, err := countItemsRecursive(sourcePath)
	if err != nil {
		return fmt.Errorf("アイテム数のカウントエラー: %w", err)
	}

	fmt.Printf("DEBUG: Total items to process: %d\n", totalItems)

	// 進捗付きで再帰的に権限をコピー
	return copyDirectoryTreePermissionsRecursiveWithProgress(sourcePath, destPath, 0, totalItems, progressCallback)
}

// countItemsRecursive counts the total number of files and directories recursively
func countItemsRecursive(path string) (int, error) {
	count := 0
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		count++
		return nil
	})
	return count, err
}

// copyDirectoryTreePermissionsRecursiveWithProgress recursively copies permissions with progress reporting
func copyDirectoryTreePermissionsRecursiveWithProgress(sourcePath, destPath string, current, total int, progressCallback func(current, total int, currentPath string)) error {
	fmt.Printf("DEBUG: copyDirectoryTreePermissionsRecursiveWithProgress called: %s -> %s (%d/%d)\n", sourcePath, destPath, current, total)

	// ソースディレクトリを開く
	entries, err := os.ReadDir(sourcePath)
	if err != nil {
		return fmt.Errorf("ソースディレクトリ(%s)の読み込みエラー: %w", sourcePath, err)
	}

	// 各エントリの処理
	for _, entry := range entries {
		sourceItemPath := filepath.Join(sourcePath, entry.Name())
		destItemPath := filepath.Join(destPath, entry.Name())

		// 宛先アイテムの存在確認
		if _, err := os.Stat(destItemPath); err != nil {
			fmt.Printf("DEBUG: Skipping %s (destination does not exist)\n", destItemPath)
			continue
		}

		current++

		// 進捗報告
		if progressCallback != nil {
			progressCallback(current, total, sourceItemPath)
		}

		// ディレクトリの場合
		if entry.IsDir() {
			fmt.Printf("DEBUG: Processing directory: %s (%d/%d)\n", sourceItemPath, current, total)

			// ディレクトリの権限をコピー
			if err := CopyDirectoryPermissions(sourceItemPath, destItemPath); err != nil {
				fmt.Printf("WARNING: ディレクトリ権限のコピーに失敗: %s -> %s: %v\n", sourceItemPath, destItemPath, err)
				// エラーが発生しても処理を続行
			} else {
				fmt.Printf("DEBUG: Successfully copied directory permissions: %s\n", destItemPath)
			}

			// 再帰的にサブディレクトリを処理
			if err := copyDirectoryTreePermissionsRecursiveWithProgress(sourceItemPath, destItemPath, current, total, progressCallback); err != nil {
				fmt.Printf("WARNING: サブディレクトリ権限のコピーに失敗: %s -> %s: %v\n", sourceItemPath, destItemPath, err)
				// エラーが発生しても処理を続行
			}
		} else {
			// ファイルの場合
			fmt.Printf("DEBUG: Processing file: %s (%d/%d)\n", sourceItemPath, current, total)

			// ファイルの権限をコピー
			if err := CopyFilePermissions(sourceItemPath, destItemPath); err != nil {
				fmt.Printf("WARNING: ファイル権限のコピーに失敗: %s -> %s: %v\n", sourceItemPath, destItemPath, err)
				// エラーが発生しても処理を続行
			} else {
				fmt.Printf("DEBUG: Successfully copied file permissions: %s\n", destItemPath)
			}
		}
	}

	return nil
}

// IsAdmin checks if the current process is running with administrator privileges
func IsAdmin() bool {
	ret, _, _ := procIsUserAnAdmin.Call()
	return ret != 0
}

// RequireAdmin checks if administrator privileges are required and prompts the user if needed
func RequireAdmin() error {
	if !IsAdmin() {
		return fmt.Errorf("この操作には管理者権限が必要です。管理者として実行してください。")
	}
	return nil
}

// CheckAdminForPermissions checks if administrator privileges are required for permission operations
func CheckAdminForPermissions() error {
	// 管理者権限が必要な操作のため、権限をチェック
	if !IsAdmin() {
		// UAC権限昇格を試行
		if IsElevationSupported() {
			fmt.Println("管理者権限が必要です。UACダイアログによる権限昇格を試行します...")
			return ElevateForPermissions()
		} else {
			return fmt.Errorf("権限コピーには管理者権限が必要です。以下のいずれかの方法で管理者として実行してください：\n" +
				"1. コマンドプロンプトまたはPowerShellを管理者として実行\n" +
				"2. 右クリック → 「管理者として実行」を選択\n" +
				"3. または、--preserve-permissionsオプションを指定せずに実行")
		}
	}
	return nil
}

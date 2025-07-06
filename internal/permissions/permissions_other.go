//go:build !windows
// +build !windows

package permissions

import (
	"fmt"
	"runtime"
)

// CopyFilePermissions copies file permissions from source to destination
// This is a dummy implementation for non-Windows platforms
func CopyFilePermissions(sourcePath, destPath string) error {
	return fmt.Errorf("ファイル権限のコピーはWindowsでのみサポートされています（現在のプラットフォーム: %s）", runtime.GOOS)
}

// CopyDirectoryPermissions copies directory permissions from source to destination
// This is a dummy implementation for non-Windows platforms
func CopyDirectoryPermissions(sourcePath, destPath string) error {
	return fmt.Errorf("ディレクトリ権限のコピーはWindowsでのみサポートされています（現在のプラットフォーム: %s）", runtime.GOOS)
}

// CopyDirectoryTreePermissions recursively copies permissions from source directory to destination directory
// This is a dummy implementation for non-Windows platforms
func CopyDirectoryTreePermissions(sourcePath, destPath string) error {
	return fmt.Errorf("ディレクトリツリー権限のコピーはWindowsでのみサポートされています（現在のプラットフォーム: %s）", runtime.GOOS)
}

// CopyDirectoryTreePermissionsWithProgress recursively copies permissions with progress reporting
// This is a dummy implementation for non-Windows platforms
func CopyDirectoryTreePermissionsWithProgress(sourcePath, destPath string, progressCallback func(current, total int, currentPath string)) error {
	return fmt.Errorf("ディレクトリツリー権限のコピーはWindowsでのみサポートされています（現在のプラットフォーム: %s）", runtime.GOOS)
}

// IsAdmin checks if the current process is running with administrator privileges
// This is a dummy implementation for non-Windows platforms
func IsAdmin() bool {
	return false
}

// RequireAdmin checks if administrator privileges are required and prompts the user if needed
// This is a dummy implementation for non-Windows platforms
func RequireAdmin() error {
	return fmt.Errorf("管理者権限チェックはWindowsでのみサポートされています（現在のプラットフォーム: %s）", runtime.GOOS)
}

// CheckAdminForPermissions checks if administrator privileges are required for permission operations
// This is a dummy implementation for non-Windows platforms
func CheckAdminForPermissions() error {
	return fmt.Errorf("権限コピーはWindowsでのみサポートされています（現在のプラットフォーム: %s）", runtime.GOOS)
}

// IsWindows returns true if running on Windows
func IsWindows() bool {
	return false
}

// CanCopyPermissions returns true if permission copying is supported
func CanCopyPermissions() bool {
	return IsWindows()
}

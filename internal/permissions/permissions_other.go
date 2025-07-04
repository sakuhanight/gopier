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

// IsWindows returns true if running on Windows
func IsWindows() bool {
	return runtime.GOOS == "windows"
}

// CanCopyPermissions returns true if permission copying is supported
func CanCopyPermissions() bool {
	return IsWindows()
}

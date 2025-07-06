//go:build !windows
// +build !windows

package permissions

import (
	"fmt"
	"runtime"
)

// ElevateWithUAC elevates the current process with UAC dialog
// This is a dummy implementation for non-Windows platforms
func ElevateWithUAC() error {
	return fmt.Errorf("UAC権限昇格はWindowsでのみサポートされています（現在のプラットフォーム: %s）", runtime.GOOS)
}

// ElevateIfNeeded elevates privileges if needed for permission operations
// This is a dummy implementation for non-Windows platforms
func ElevateIfNeeded() error {
	return nil // Windows以外では何もしない
}

// ElevateForPermissions elevates privileges specifically for permission copying operations
// This is a dummy implementation for non-Windows platforms
func ElevateForPermissions() error {
	return nil // Windows以外では何もしない
}

// CheckAndElevateForPermissions checks if elevation is needed and elevates if necessary
// This is a dummy implementation for non-Windows platforms
func CheckAndElevateForPermissions() error {
	return nil // Windows以外では何もしない
}

// IsElevationSupported returns true if UAC elevation is supported on this system
// This is a dummy implementation for non-Windows platforms
func IsElevationSupported() bool {
	return false
}

// GetElevationStatus returns the current elevation status
// This is a dummy implementation for non-Windows platforms
func GetElevationStatus() string {
	return fmt.Sprintf("非Windowsプラットフォーム（%s）", runtime.GOOS)
}

//go:build windows
// +build windows

package permissions

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
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
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	advapi32 = syscall.NewLazyDLL("advapi32.dll")

	procGetFileSecurityW            = advapi32.NewProc("GetFileSecurityW")
	procSetFileSecurityW            = advapi32.NewProc("SetFileSecurityW")
	procGetSecurityDescriptorLength = advapi32.NewProc("GetSecurityDescriptorLength")
)

// SecurityDescriptor represents a Windows security descriptor
type SecurityDescriptor struct {
	Revision byte
	Sbz1     byte
	Control  uint16
	Owner    *syscall.SID
	Group    *syscall.SID
	Sacl     *syscall.ACL
	Dacl     *syscall.ACL
}

// CopyFilePermissions copies file permissions from source to destination
func CopyFilePermissions(sourcePath, destPath string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("ファイル権限のコピーはWindowsでのみサポートされています")
	}

	// ソースファイルのセキュリティ記述子を取得
	sourceSD, err := getFileSecurity(sourcePath)
	if err != nil {
		return fmt.Errorf("ソースファイルのセキュリティ記述子取得エラー: %w", err)
	}

	// 宛先ファイルにセキュリティ記述子を設定
	err = setFileSecurity(destPath, sourceSD)
	if err != nil {
		return fmt.Errorf("宛先ファイルのセキュリティ記述子設定エラー: %w", err)
	}

	return nil
}

// getFileSecurity retrieves the security descriptor for a file
func getFileSecurity(filePath string) (*SecurityDescriptor, error) {
	// 最初に必要なサイズを取得
	var sizeNeeded uint32
	ret, _, _ := procGetFileSecurityW.Call(
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(filePath))),
		uintptr(DACL_SECURITY_INFORMATION|OWNER_SECURITY_INFORMATION|GROUP_SECURITY_INFORMATION),
		0,
		0,
		uintptr(unsafe.Pointer(&sizeNeeded)),
	)

	if ret == 0 && sizeNeeded == 0 {
		return nil, fmt.Errorf("ファイルのセキュリティ情報を取得できません: %s", filePath)
	}

	// 適切なサイズのバッファを確保
	buffer := make([]byte, sizeNeeded)
	ret, _, err := procGetFileSecurityW.Call(
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(filePath))),
		uintptr(DACL_SECURITY_INFORMATION|OWNER_SECURITY_INFORMATION|GROUP_SECURITY_INFORMATION),
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(sizeNeeded),
		uintptr(unsafe.Pointer(&sizeNeeded)),
	)

	if ret == 0 {
		return nil, fmt.Errorf("ファイルのセキュリティ情報を取得できません: %s (エラー: %v)", filePath, err)
	}

	// セキュリティ記述子を解析
	sd := (*SecurityDescriptor)(unsafe.Pointer(&buffer[0]))
	return sd, nil
}

// setFileSecurity sets the security descriptor for a file
func setFileSecurity(filePath string, sd *SecurityDescriptor) error {
	ret, _, err := procSetFileSecurityW.Call(
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(filePath))),
		uintptr(DACL_SECURITY_INFORMATION|OWNER_SECURITY_INFORMATION|GROUP_SECURITY_INFORMATION),
		uintptr(unsafe.Pointer(sd)),
	)

	if ret == 0 {
		return fmt.Errorf("ファイルのセキュリティ情報を設定できません: %s (エラー: %v)", filePath, err)
	}

	return nil
}

// CopyDirectoryPermissions copies directory permissions from source to destination
func CopyDirectoryPermissions(sourcePath, destPath string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("ディレクトリ権限のコピーはWindowsでのみサポートされています")
	}

	// ディレクトリのセキュリティ記述子を取得
	sourceSD, err := getFileSecurity(sourcePath)
	if err != nil {
		return fmt.Errorf("ソースディレクトリのセキュリティ記述子取得エラー: %w", err)
	}

	// 宛先ディレクトリにセキュリティ記述子を設定
	err = setFileSecurity(destPath, sourceSD)
	if err != nil {
		return fmt.Errorf("宛先ディレクトリのセキュリティ記述子設定エラー: %w", err)
	}

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

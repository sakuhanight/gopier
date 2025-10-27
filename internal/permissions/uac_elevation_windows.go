//go:build windows
// +build windows

package permissions

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows API constants for ShellExecute
const (
	SW_SHOWNORMAL = 1
)

// Windows API functions for UAC elevation
var (
	procShellExecuteW = shell32.NewProc("ShellExecuteW")
)

// ElevateWithUAC elevates the current process with UAC dialog
// If the current process is not running as administrator, it will restart itself with elevated privileges
func ElevateWithUAC() error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("UAC権限昇格はWindowsでのみサポートされています")
	}

	// 既に管理者権限で実行されている場合は何もしない
	if IsAdmin() {
		return nil
	}

	// 現在の実行ファイルのパスを取得
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("実行ファイルのパス取得エラー: %w", err)
	}

	// 現在のコマンドライン引数を取得
	args := os.Args[1:] // 最初の要素（実行ファイル名）を除外

	// 管理者権限で再実行
	return restartAsAdmin(exePath, args)
}

// restartAsAdmin restarts the application with administrator privileges
func restartAsAdmin(exePath string, args []string) error {
	// 実行ファイルのパスをUTF16に変換
	exePathPtr, err := windows.UTF16PtrFromString(exePath)
	if err != nil {
		return fmt.Errorf("実行ファイルパスの変換エラー: %w", err)
	}

	// 引数を結合
	parameters := strings.Join(args, " ")

	// パラメータをUTF16に変換
	var parametersPtr *uint16
	if parameters != "" {
		parametersPtr, err = windows.UTF16PtrFromString(parameters)
		if err != nil {
			return fmt.Errorf("パラメータの変換エラー: %w", err)
		}
	}

	// "runas" verbをUTF16に変換（管理者権限で実行）
	verbPtr, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return fmt.Errorf("verbの変換エラー: %w", err)
	}

	// ShellExecuteWを呼び出し（直接呼び出し）
	ret, _, callErr := procShellExecuteW.Call(
		0,                                      // hwnd (親ウィンドウハンドル)
		uintptr(unsafe.Pointer(verbPtr)),       // lpVerb
		uintptr(unsafe.Pointer(exePathPtr)),    // lpFile
		uintptr(unsafe.Pointer(parametersPtr)), // lpParameters
		0,                                      // lpDirectory
		uintptr(SW_SHOWNORMAL),                 // nShow
	)

	// エラーチェック
	if ret <= 32 { // 32以下の戻り値はエラーを示す
		if callErr != nil {
			return fmt.Errorf("ShellExecuteW呼び出しエラー: %w", callErr)
		}
		return fmt.Errorf("UACダイアログの表示に失敗しました（戻り値: %d）", ret)
	}

	// 新しいプロセスが開始された場合、現在のプロセスを終了
	if ret > 32 {
		fmt.Println("管理者権限で新しいプロセスを開始しました。")
		fmt.Println("現在のプロセスを終了します...")
		// 現在のプロセスを終了
		os.Exit(0)
	}

	return nil
}

// ElevateIfNeeded elevates privileges if needed for permission operations
func ElevateIfNeeded() error {
	if runtime.GOOS != "windows" {
		return nil // Windows以外では何もしない
	}

	// 管理者権限が必要な操作の場合
	if !IsAdmin() {
		fmt.Println("管理者権限が必要です。UACダイアログが表示されます...")

		err := ElevateWithUAC()
		if err != nil {
			return fmt.Errorf("権限昇格に失敗しました: %w", err)
		}

		// 権限昇格が成功した場合、新しいプロセスが開始されるため
		// ここには到達しないはず
		return nil
	}

	return nil
}

// ElevateForPermissions elevates privileges specifically for permission copying operations
func ElevateForPermissions() error {
	if runtime.GOOS != "windows" {
		return nil // Windows以外では何もしない
	}

	// 管理者権限が必要な操作の場合
	if !IsAdmin() {
		fmt.Println("権限コピーには管理者権限が必要です。UACダイアログが表示されます...")

		err := ElevateWithUAC()
		if err != nil {
			return fmt.Errorf("権限昇格に失敗しました: %w", err)
		}

		// 権限昇格が成功した場合、新しいプロセスが開始されるため
		// ここには到達しないはず
		return nil
	}

	return nil
}

// CheckAndElevateForPermissions checks if elevation is needed and elevates if necessary
func CheckAndElevateForPermissions() error {
	if runtime.GOOS != "windows" {
		return nil // Windows以外では何もしない
	}

	// 管理者権限が必要な操作の場合
	if !IsAdmin() {
		fmt.Println("権限コピーには管理者権限が必要です。")
		fmt.Print("UACダイアログが表示されます。続行しますか？ (y/N): ")

		// ユーザーの確認を求める
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("ユーザー入力の読み取りエラー: %w", err)
		}

		// 応答を正規化
		response = strings.TrimSpace(strings.ToLower(response))

		if response == "y" || response == "yes" {
			err := ElevateForPermissions()
			if err != nil {
				return fmt.Errorf("権限昇格に失敗しました: %w", err)
			}

			// 権限昇格が成功した場合、新しいプロセスが開始されるため
			// ここには到達しないはず
			return nil
		} else {
			return fmt.Errorf("ユーザーが権限昇格をキャンセルしました")
		}
	}

	return nil
}

// IsElevationSupported returns true if UAC elevation is supported on this system
func IsElevationSupported() bool {
	return runtime.GOOS == "windows"
}

// GetElevationStatus returns the current elevation status
func GetElevationStatus() string {
	if runtime.GOOS != "windows" {
		return "非Windowsプラットフォーム"
	}

	if IsAdmin() {
		return "管理者権限で実行中"
	} else {
		return "標準権限で実行中（管理者権限が必要な場合があります）"
	}
}

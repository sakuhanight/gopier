//go:build !windows
// +build !windows

package permissions

import (
	"testing"
)

func TestElevationFunctionsNonWindows(t *testing.T) {
	// IsElevationSupported関数のテスト
	isSupported := IsElevationSupported()
	t.Logf("UAC権限昇格がサポートされていますか: %v", isSupported)

	// Windows以外ではサポートされていないことを確認
	if isSupported {
		t.Errorf("非WindowsプラットフォームでUAC権限昇格がサポートされていると報告されています")
	}

	// GetElevationStatus関数のテスト
	status := GetElevationStatus()
	t.Logf("現在の権限昇格状態: %s", status)

	// 非Windowsプラットフォームであることを確認
	if status == "管理者権限で実行中" || status == "標準権限で実行中" {
		t.Errorf("非WindowsプラットフォームでWindows固有の状態が報告されています: %s", status)
	}
}

func TestElevateWithUACNonWindows(t *testing.T) {
	// ElevateWithUAC関数のテスト
	err := ElevateWithUAC()
	if err == nil {
		t.Errorf("非WindowsプラットフォームでUAC権限昇格が成功しました（エラーが発生すべき）")
	} else {
		t.Logf("期待されるエラー: %v", err)
	}
}

func TestElevateIfNeededNonWindows(t *testing.T) {
	// ElevateIfNeeded関数のテスト
	err := ElevateIfNeeded()
	if err != nil {
		t.Errorf("非WindowsプラットフォームでElevateIfNeededがエラーを返しました: %v", err)
	}
}

func TestElevateForPermissionsNonWindows(t *testing.T) {
	// ElevateForPermissions関数のテスト
	err := ElevateForPermissions()
	if err != nil {
		t.Errorf("非WindowsプラットフォームでElevateForPermissionsがエラーを返しました: %v", err)
	}
}

func TestCheckAndElevateForPermissionsNonWindows(t *testing.T) {
	// CheckAndElevateForPermissions関数のテスト
	err := CheckAndElevateForPermissions()
	if err != nil {
		t.Errorf("非WindowsプラットフォームでCheckAndElevateForPermissionsがエラーを返しました: %v", err)
	}
}

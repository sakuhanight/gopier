//go:build windows
// +build windows

package permissions

import (
	"testing"
)

func TestElevationFunctions(t *testing.T) {
	// IsElevationSupported関数のテスト
	isSupported := IsElevationSupported()
	t.Logf("UAC権限昇格がサポートされていますか: %v", isSupported)

	// GetElevationStatus関数のテスト
	status := GetElevationStatus()
	t.Logf("現在の権限昇格状態: %s", status)

	// IsAdmin関数のテスト
	isAdmin := IsAdmin()
	t.Logf("現在のプロセスは管理者権限で実行されていますか: %v", isAdmin)

	// ElevateIfNeeded関数のテスト（実際の権限昇格は行わない）
	if !isAdmin {
		t.Logf("管理者権限で実行されていないため、権限昇格が必要です")
		// 実際の権限昇格はテスト環境では実行しない
		// err := ElevateIfNeeded()
		// if err != nil {
		//     t.Logf("権限昇格エラー: %v", err)
		// }
	} else {
		t.Logf("既に管理者権限で実行されているため、権限昇格は不要です")
		err := ElevateIfNeeded()
		if err != nil {
			t.Errorf("管理者権限で実行中なのに権限昇格エラーが発生: %v", err)
		}
	}
}

func TestElevateForPermissions(t *testing.T) {
	// 管理者権限が必要なテストのため、adminタグでスキップ
	if testing.Short() {
		t.Skip("管理者権限が必要なテストのため、-shortフラグでスキップ")
	}

	// ElevateForPermissions関数のテスト
	isAdmin := IsAdmin()
	t.Logf("現在のプロセスは管理者権限で実行されていますか: %v", isAdmin)

	if !isAdmin {
		t.Logf("管理者権限で実行されていないため、権限コピーには権限昇格が必要です")
		// 実際の権限昇格はテスト環境では実行しない
		// err := ElevateForPermissions()
		// if err != nil {
		//     t.Logf("権限昇格エラー: %v", err)
		// }
	} else {
		t.Logf("既に管理者権限で実行されているため、権限昇格は不要です")
		err := ElevateForPermissions()
		if err != nil {
			t.Errorf("管理者権限で実行中なのに権限昇格エラーが発生: %v", err)
		}
	}
}

func TestCheckAndElevateForPermissions(t *testing.T) {
	// 管理者権限が必要なテストのため、adminタグでスキップ
	if testing.Short() {
		t.Skip("管理者権限が必要なテストのため、-shortフラグでスキップ")
	}

	// CheckAndElevateForPermissions関数のテスト
	isAdmin := IsAdmin()
	t.Logf("現在のプロセスは管理者権限で実行されていますか: %v", isAdmin)

	if !isAdmin {
		t.Logf("管理者権限で実行されていないため、権限コピーには権限昇格が必要です")
		// 実際の権限昇格はテスト環境では実行しない
		// err := CheckAndElevateForPermissions()
		// if err != nil {
		//     t.Logf("権限昇格エラー: %v", err)
		// }
	} else {
		t.Logf("既に管理者権限で実行されているため、権限昇格は不要です")
		err := CheckAndElevateForPermissions()
		if err != nil {
			t.Errorf("管理者権限で実行中なのに権限昇格エラーが発生: %v", err)
		}
	}
}

func TestElevateWithUAC(t *testing.T) {
	// 管理者権限が必要なテストのため、adminタグでスキップ
	if testing.Short() {
		t.Skip("管理者権限が必要なテストのため、-shortフラグでスキップ")
	}

	// ElevateWithUAC関数のテスト
	isAdmin := IsAdmin()
	t.Logf("現在のプロセスは管理者権限で実行されていますか: %v", isAdmin)

	if !isAdmin {
		t.Logf("管理者権限で実行されていないため、UAC権限昇格が必要です")
		// 実際の権限昇格はテスト環境では実行しない
		// err := ElevateWithUAC()
		// if err != nil {
		//     t.Logf("UAC権限昇格エラー: %v", err)
		// }
	} else {
		t.Logf("既に管理者権限で実行されているため、UAC権限昇格は不要です")
		err := ElevateWithUAC()
		if err != nil {
			t.Errorf("管理者権限で実行中なのにUAC権限昇格エラーが発生: %v", err)
		}
	}
}

func TestIntegrationWithPermissions(t *testing.T) {
	// 管理者権限が必要なテストのため、adminタグでスキップ
	if testing.Short() {
		t.Skip("管理者権限が必要なテストのため、-shortフラグでスキップ")
	}

	// 権限コピー機能との統合テスト
	t.Logf("権限昇格状態: %s", GetElevationStatus())

	// CheckAdminForPermissions関数のテスト
	err := CheckAdminForPermissions()
	if err != nil {
		if isAdmin := IsAdmin(); !isAdmin {
			t.Logf("期待される管理者権限エラー: %v", err)
		} else {
			t.Errorf("管理者権限で実行中なのにエラーが発生: %v", err)
		}
	} else {
		t.Logf("管理者権限チェック成功")
	}
}

// 管理者権限が不要なテスト（常に実行される）
func TestBasicElevationFunctions(t *testing.T) {
	// IsElevationSupported関数のテスト
	isSupported := IsElevationSupported()
	if !isSupported {
		t.Errorf("Windows環境でUAC権限昇格がサポートされていません")
	}

	// GetElevationStatus関数のテスト
	status := GetElevationStatus()
	if status == "" {
		t.Errorf("権限昇格状態が空です")
	}
	t.Logf("権限昇格状態: %s", status)

	// IsAdmin関数のテスト
	isAdmin := IsAdmin()
	t.Logf("管理者権限で実行中: %v", isAdmin)
}

func TestElevationErrorHandling(t *testing.T) {
	// エラーハンドリングのテスト（実際の権限昇格は行わない）
	isAdmin := IsAdmin()

	if !isAdmin {
		t.Logf("管理者権限で実行されていないため、権限昇格が必要です")
		// 実際の権限昇格はテスト環境では実行しない
	} else {
		t.Logf("管理者権限で実行されているため、権限昇格は不要です")
	}
}

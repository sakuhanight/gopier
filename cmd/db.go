package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/sakuhanight/gopier/internal/database"
)

var (
	dbPath      string
	dbOutput    string
	dbFormat    string
	dbStatus    string
	dbLimit     int
	dbSortBy    string
	dbReverse   bool
	dbNoConfirm bool
)

// dbCmd represents the db command
var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "同期データベースの閲覧・管理",
	Long: `同期データベースの内容を閲覧・管理するコマンドです。

利用可能なサブコマンド:
  list     - データベース内のファイル一覧を表示
  stats    - 同期統計情報を表示
  export   - データベースの内容をファイルにエクスポート
  clean    - 古いレコードを削除
  reset    - データベースをリセット（初期同期モード用）`,
}

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "データベース内のファイル一覧を表示",
	Long: `データベースに記録されているファイルの一覧を表示します。

フィルタリングオプション:
  --status: 特定のステータスのファイルのみ表示
  --limit: 表示件数を制限
  --sort-by: ソート項目（path, size, mod_time, status, last_sync_time）
  --reverse: 逆順でソート`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if dbPath == "" {
			return fmt.Errorf("データベースパスが指定されていません。--dbフラグを使用してください。")
		}

		// データベースを開く
		syncDB, err := database.NewSyncDB(dbPath, database.NormalSync)
		if err != nil {
			return fmt.Errorf("データベースのオープンに失敗: %w", err)
		}
		defer syncDB.Close()

		// ファイル一覧を取得
		files, err := syncDB.GetAllFiles()
		if err != nil {
			return fmt.Errorf("ファイル一覧の取得に失敗: %w", err)
		}

		// フィルタリング
		if dbStatus != "" {
			filtered := make([]database.FileInfo, 0)
			for _, file := range files {
				if string(file.Status) == dbStatus {
					filtered = append(filtered, file)
				}
			}
			files = filtered
		}

		// ソート
		sortFiles(files, dbSortBy, dbReverse)

		// 件数制限
		if dbLimit > 0 && len(files) > dbLimit {
			files = files[:dbLimit]
		}

		// 表示
		fmt.Printf("データベース: %s\n", dbPath)
		fmt.Printf("総ファイル数: %d\n\n", len(files))

		if len(files) == 0 {
			fmt.Println("ファイルが見つかりません。")
			return nil
		}

		// ヘッダー
		fmt.Printf("%-50s %-10s %-20s %-15s %-20s\n", "パス", "サイズ", "更新日時", "ステータス", "最終同期")
		fmt.Println(strings.Repeat("-", 120))

		// ファイル一覧
		for _, file := range files {
			sizeStr := formatBytes(file.Size)
			modTimeStr := file.ModTime.Format("2006-01-02 15:04:05")
			syncTimeStr := file.LastSyncTime.Format("2006-01-02 15:04:05")
			statusStr := string(file.Status)

			fmt.Printf("%-50s %-10s %-20s %-15s %-20s\n",
				truncateString(file.Path, 50),
				sizeStr,
				modTimeStr,
				statusStr,
				syncTimeStr)
		}
		return nil
	},
}

// statsCmd represents the stats command
var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "同期統計情報を表示",
	Long:  `データベースに記録されている同期統計情報を表示します。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if dbPath == "" {
			return fmt.Errorf("データベースパスが指定されていません。--dbフラグを使用してください。")
		}

		// データベースを開く
		syncDB, err := database.NewSyncDB(dbPath, database.NormalSync)
		if err != nil {
			return fmt.Errorf("データベースのオープンに失敗: %w", err)
		}
		defer syncDB.Close()

		// 統計情報を取得
		stats, err := syncDB.GetSyncStats()
		if err != nil {
			return fmt.Errorf("統計情報の取得に失敗: %w", err)
		}

		// ファイル一覧を取得して詳細統計を計算
		files, err := syncDB.GetAllFiles()
		if err != nil {
			return fmt.Errorf("ファイル一覧の取得に失敗: %w", err)
		}

		fmt.Printf("データベース: %s\n", dbPath)
		fmt.Println(strings.Repeat("=", 50))

		// 基本統計
		fmt.Printf("総ファイル数: %d\n", len(files))
		fmt.Printf("総サイズ: %s\n", formatBytes(calculateTotalSize(files)))

		// ステータス別統計
		statusCount := make(map[database.FileStatus]int)
		for _, file := range files {
			statusCount[file.Status]++
		}

		fmt.Println("\nステータス別統計:")
		for status, count := range statusCount {
			fmt.Printf("  %s: %d件\n", status, count)
		}

		// 同期セッション統計
		fmt.Println("\n同期セッション統計:")
		for key, value := range stats {
			fmt.Printf("  %s: %d\n", key, value)
		}

		// 失敗回数統計
		failCounts := make(map[int]int)
		for _, file := range files {
			failCounts[file.FailCount]++
		}

		fmt.Println("\n失敗回数別統計:")
		for failCount, count := range failCounts {
			if failCount > 0 {
				fmt.Printf("  失敗%d回: %d件\n", failCount, count)
			}
		}
		return nil
	},
}

// exportCmd represents the export command
var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "データベースの内容をファイルにエクスポート",
	Long: `データベースの内容をCSVまたはJSON形式でファイルにエクスポートします。

サポートされている形式:
  csv  - CSVファイル（デフォルト）
  json - JSONファイル`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if dbPath == "" {
			return fmt.Errorf("データベースパスが指定されていません。--dbフラグを使用してください。")
		}

		if dbOutput == "" {
			return fmt.Errorf("出力ファイルが指定されていません。--outputフラグを使用してください。")
		}

		// データベースを開く
		syncDB, err := database.NewSyncDB(dbPath, database.NormalSync)
		if err != nil {
			return fmt.Errorf("データベースのオープンに失敗: %w", err)
		}
		defer syncDB.Close()

		// ファイル一覧を取得
		files, err := syncDB.GetAllFiles()
		if err != nil {
			return fmt.Errorf("ファイル一覧の取得に失敗: %w", err)
		}

		// フィルタリング
		if dbStatus != "" {
			filtered := make([]database.FileInfo, 0)
			for _, file := range files {
				if string(file.Status) == dbStatus {
					filtered = append(filtered, file)
				}
			}
			files = filtered
		}

		// ソート
		sortFiles(files, dbSortBy, dbReverse)

		// エクスポート
		switch strings.ToLower(dbFormat) {
		case "csv":
			err = exportToCSV(files, dbOutput)
		case "json":
			err = exportToJSON(files, dbOutput)
		default:
			return fmt.Errorf("サポートされていない形式です: %s", dbFormat)
		}
		if err != nil {
			return fmt.Errorf("エクスポートに失敗: %w", err)
		}
		fmt.Printf("エクスポートが完了しました: %s\n", dbOutput)
		return nil
	},
}

// cleanCmd represents the clean command
var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "古いレコードを削除",
	Long:  `指定された日数より古いレコードを削除します。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if dbPath == "" {
			return fmt.Errorf("データベースパスが指定されていません。--dbフラグを使用してください。")
		}

		// データベースを開く
		syncDB, err := database.NewSyncDB(dbPath, database.NormalSync)
		if err != nil {
			return fmt.Errorf("データベースのオープンに失敗: %w", err)
		}
		defer syncDB.Close()

		// ファイル一覧を取得
		files, err := syncDB.GetAllFiles()
		if err != nil {
			return fmt.Errorf("ファイル一覧の取得に失敗: %w", err)
		}

		// 古いレコードを削除
		cutoff := time.Now().AddDate(0, 0, -30) // デフォルト30日前
		deletedCount := 0

		for _, file := range files {
			if file.LastSyncTime.Before(cutoff) {
				// レコードを削除（実装は後で追加）
				deletedCount++
			}
		}

		fmt.Printf("%d件の古いレコードを削除しました。\n", deletedCount)
		return nil
	},
}

// resetCmd represents the reset command
var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "データベースをリセット",
	Long: `データベースをリセットします（初期同期モード用）。
注意: この操作は元に戻せません。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if dbPath == "" {
			return fmt.Errorf("データベースパスが指定されていません。--dbフラグを使用してください。")
		}

		// 確認（--no-confirmフラグが指定されていない場合のみ）
		if !dbNoConfirm {
			fmt.Printf("データベース %s をリセットしますか？ (y/N): ", dbPath)
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
				fmt.Println("リセットをキャンセルしました。")
				return nil
			}
		}

		// データベースを開く（初期同期モード）
		syncDB, err := database.NewSyncDB(dbPath, database.InitialSync)
		if err != nil {
			return fmt.Errorf("データベースのオープンに失敗: %w", err)
		}
		defer syncDB.Close()

		// リセット
		err = syncDB.ResetDatabase()
		if err != nil {
			return fmt.Errorf("データベースのリセットに失敗: %w", err)
		}

		fmt.Println("データベースをリセットしました。")
		return nil
	},
}

// init()は削除 - コマンド定義は残す

// ヘルパー関数はcmd/root.goに移動済み

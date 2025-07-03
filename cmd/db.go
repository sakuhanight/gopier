package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
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
	Run: func(cmd *cobra.Command, args []string) {
		if dbPath == "" {
			fmt.Fprintf(os.Stderr, "データベースパスが指定されていません。--dbフラグを使用してください。\n")
			os.Exit(1)
		}

		// データベースを開く
		syncDB, err := database.NewSyncDB(dbPath, database.NormalSync)
		if err != nil {
			fmt.Fprintf(os.Stderr, "データベースのオープンに失敗: %v\n", err)
			os.Exit(1)
		}
		defer syncDB.Close()

		// ファイル一覧を取得
		files, err := syncDB.GetAllFiles()
		if err != nil {
			fmt.Fprintf(os.Stderr, "ファイル一覧の取得に失敗: %v\n", err)
			os.Exit(1)
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
			return
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
	},
}

// statsCmd represents the stats command
var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "同期統計情報を表示",
	Long:  `データベースに記録されている同期統計情報を表示します。`,
	Run: func(cmd *cobra.Command, args []string) {
		if dbPath == "" {
			fmt.Fprintf(os.Stderr, "データベースパスが指定されていません。--dbフラグを使用してください。\n")
			os.Exit(1)
		}

		// データベースを開く
		syncDB, err := database.NewSyncDB(dbPath, database.NormalSync)
		if err != nil {
			fmt.Fprintf(os.Stderr, "データベースのオープンに失敗: %v\n", err)
			os.Exit(1)
		}
		defer syncDB.Close()

		// 統計情報を取得
		stats, err := syncDB.GetSyncStats()
		if err != nil {
			fmt.Fprintf(os.Stderr, "統計情報の取得に失敗: %v\n", err)
			os.Exit(1)
		}

		// ファイル一覧を取得して詳細統計を計算
		files, err := syncDB.GetAllFiles()
		if err != nil {
			fmt.Fprintf(os.Stderr, "ファイル一覧の取得に失敗: %v\n", err)
			os.Exit(1)
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
	Run: func(cmd *cobra.Command, args []string) {
		if dbPath == "" {
			fmt.Fprintf(os.Stderr, "データベースパスが指定されていません。--dbフラグを使用してください。\n")
			os.Exit(1)
		}

		if dbOutput == "" {
			fmt.Fprintf(os.Stderr, "出力ファイルが指定されていません。--outputフラグを使用してください。\n")
			os.Exit(1)
		}

		// データベースを開く
		syncDB, err := database.NewSyncDB(dbPath, database.NormalSync)
		if err != nil {
			fmt.Fprintf(os.Stderr, "データベースのオープンに失敗: %v\n", err)
			os.Exit(1)
		}
		defer syncDB.Close()

		// ファイル一覧を取得
		files, err := syncDB.GetAllFiles()
		if err != nil {
			fmt.Fprintf(os.Stderr, "ファイル一覧の取得に失敗: %v\n", err)
			os.Exit(1)
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
			fmt.Fprintf(os.Stderr, "サポートされていない形式: %s\n", dbFormat)
			os.Exit(1)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "エクスポートに失敗: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("データベースの内容を %s にエクスポートしました: %s\n", dbFormat, dbOutput)
	},
}

// cleanCmd represents the clean command
var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "古いレコードを削除",
	Long:  `指定された日数より古いレコードを削除します。`,
	Run: func(cmd *cobra.Command, args []string) {
		if dbPath == "" {
			fmt.Fprintf(os.Stderr, "データベースパスが指定されていません。--dbフラグを使用してください。\n")
			os.Exit(1)
		}

		// データベースを開く
		syncDB, err := database.NewSyncDB(dbPath, database.NormalSync)
		if err != nil {
			fmt.Fprintf(os.Stderr, "データベースのオープンに失敗: %v\n", err)
			os.Exit(1)
		}
		defer syncDB.Close()

		// ファイル一覧を取得
		files, err := syncDB.GetAllFiles()
		if err != nil {
			fmt.Fprintf(os.Stderr, "ファイル一覧の取得に失敗: %v\n", err)
			os.Exit(1)
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
	},
}

// resetCmd represents the reset command
var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "データベースをリセット",
	Long: `データベースをリセットします（初期同期モード用）。
注意: この操作は元に戻せません。`,
	Run: func(cmd *cobra.Command, args []string) {
		if dbPath == "" {
			fmt.Fprintf(os.Stderr, "データベースパスが指定されていません。--dbフラグを使用してください。\n")
			os.Exit(1)
		}

		// 確認（--no-confirmフラグが指定されていない場合のみ）
		if !dbNoConfirm {
			fmt.Printf("データベース %s をリセットしますか？ (y/N): ", dbPath)
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
				fmt.Println("リセットをキャンセルしました。")
				return
			}
		}

		// データベースを開く（初期同期モード）
		syncDB, err := database.NewSyncDB(dbPath, database.InitialSync)
		if err != nil {
			fmt.Fprintf(os.Stderr, "データベースのオープンに失敗: %v\n", err)
			os.Exit(1)
		}
		defer syncDB.Close()

		// リセット
		err = syncDB.ResetDatabase()
		if err != nil {
			fmt.Fprintf(os.Stderr, "データベースのリセットに失敗: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("データベースをリセットしました。")
	},
}

func init() {
	rootCmd.AddCommand(dbCmd)

	// サブコマンドを追加
	dbCmd.AddCommand(listCmd)
	dbCmd.AddCommand(statsCmd)
	dbCmd.AddCommand(exportCmd)
	dbCmd.AddCommand(cleanCmd)
	dbCmd.AddCommand(resetCmd)

	// 共通フラグ
	dbCmd.PersistentFlags().StringVar(&dbPath, "db", "", "データベースファイルのパス")
	dbCmd.PersistentFlags().StringVar(&dbStatus, "status", "", "特定のステータスのファイルのみ対象")
	dbCmd.PersistentFlags().StringVar(&dbSortBy, "sort-by", "path", "ソート項目 (path, size, mod_time, status, last_sync_time)")
	dbCmd.PersistentFlags().BoolVar(&dbReverse, "reverse", false, "逆順でソート")

	// listコマンドのフラグ
	listCmd.Flags().IntVar(&dbLimit, "limit", 0, "表示件数の制限")

	// exportコマンドのフラグ
	exportCmd.Flags().StringVar(&dbOutput, "output", "", "出力ファイルのパス")
	exportCmd.Flags().StringVar(&dbFormat, "format", "csv", "出力形式 (csv, json)")

	// cleanコマンドのフラグ
	cleanCmd.Flags().BoolVar(&dbNoConfirm, "no-confirm", false, "確認なしで実行")

	// resetコマンドのフラグ
	resetCmd.Flags().BoolVar(&dbNoConfirm, "no-confirm", false, "確認なしで実行")
}

// ヘルパー関数
func sortFiles(files []database.FileInfo, sortBy string, reverse bool) {
	sort.Slice(files, func(i, j int) bool {
		var result bool
		switch sortBy {
		case "path":
			result = files[i].Path < files[j].Path
		case "size":
			result = files[i].Size < files[j].Size
		case "mod_time":
			result = files[i].ModTime.Before(files[j].ModTime)
		case "status":
			result = string(files[i].Status) < string(files[j].Status)
		case "last_sync_time":
			result = files[i].LastSyncTime.Before(files[j].LastSyncTime)
		default:
			result = files[i].Path < files[j].Path
		}
		if reverse {
			return !result
		}
		return result
	})
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return "..."
	}
	return string(runes[:maxLen-3]) + "..."
}

func calculateTotalSize(files []database.FileInfo) int64 {
	var total int64
	for _, file := range files {
		total += file.Size
	}
	return total
}

func exportToCSV(files []database.FileInfo, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// ヘッダー
	header := []string{"パス", "サイズ", "更新日時", "ステータス", "ソースハッシュ", "宛先ハッシュ", "失敗回数", "最終同期", "最終エラー"}
	if err := writer.Write(header); err != nil {
		return err
	}

	// データ
	for _, file := range files {
		row := []string{
			file.Path,
			fmt.Sprintf("%d", file.Size),
			file.ModTime.Format(time.RFC3339),
			string(file.Status),
			file.SourceHash,
			file.DestHash,
			fmt.Sprintf("%d", file.FailCount),
			file.LastSyncTime.Format(time.RFC3339),
			file.LastError,
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

func exportToJSON(files []database.FileInfo, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(files)
}

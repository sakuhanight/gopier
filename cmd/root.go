/*
Copyright © 2025 sakuhanight

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/sakuhanight/gopier/internal/copier"
	"github.com/sakuhanight/gopier/internal/database"
	"github.com/sakuhanight/gopier/internal/filter"
	"github.com/sakuhanight/gopier/internal/logger"
	"github.com/sakuhanight/gopier/internal/verifier"
)

var (
	cfgFile string

	// 基本オプション
	sourceDir      string
	destDir        string
	logFile        string
	numWorkers     int
	retryCount     int
	retryWait      int
	includePattern string
	excludePattern string
	mirror         bool
	dryRun         bool
	verbose        bool
	skipNewer      bool
	noProgress     bool
	bufferSize     int
	recursive      bool

	// 同期モード関連
	syncMode      string
	syncDBPath    string
	verifyOnly    bool
	verifyAll     bool
	verifyChanged bool
	includeFailed bool
	maxFailCount  int
	finalReport   string
)

// Config は設定ファイルの構造を定義する
type Config struct {
	// 基本設定
	Source      string `mapstructure:"source"`
	Destination string `mapstructure:"destination"`
	LogFile     string `mapstructure:"log_file"`

	// パフォーマンス設定
	Workers    int `mapstructure:"workers"`
	BufferSize int `mapstructure:"buffer_size"`
	RetryCount int `mapstructure:"retry_count"`
	RetryWait  int `mapstructure:"retry_wait"`

	// フィルタ設定
	IncludePattern string `mapstructure:"include_pattern"`
	ExcludePattern string `mapstructure:"exclude_pattern"`

	// 動作設定
	Recursive         bool `mapstructure:"recursive"`
	Mirror            bool `mapstructure:"mirror"`
	DryRun            bool `mapstructure:"dry_run"`
	Verbose           bool `mapstructure:"verbose"`
	SkipNewer         bool `mapstructure:"skip_newer"`
	NoProgress        bool `mapstructure:"no_progress"`
	PreserveModTime   bool `mapstructure:"preserve_mod_time"`
	OverwriteExisting bool `mapstructure:"overwrite_existing"`

	// 同期設定
	SyncMode      string `mapstructure:"sync_mode"`
	SyncDBPath    string `mapstructure:"sync_db_path"`
	IncludeFailed bool   `mapstructure:"include_failed"`
	MaxFailCount  int    `mapstructure:"max_fail_count"`

	// 検証設定
	VerifyOnly    bool   `mapstructure:"verify_only"`
	VerifyChanged bool   `mapstructure:"verify_changed"`
	VerifyAll     bool   `mapstructure:"verify_all"`
	FinalReport   string `mapstructure:"final_report"`

	// ハッシュ設定
	HashAlgorithm string `mapstructure:"hash_algorithm"`
	VerifyHash    bool   `mapstructure:"verify_hash"`
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gopier",
	Short: "高性能なファイル同期ツール",
	Long: `Gopierは、Goで実装された高性能なファイル同期ツールです。
初期同期と追加同期の各フェーズに対応し、失敗したファイルの再同期機能と
ハッシュ検証機能を備えています。

詳細なログ出力にはUberのZapロガーを使用しています。`,
	Run: func(cmd *cobra.Command, args []string) {
		// 設定ファイル作成フラグの確認
		if createConfig, _ := cmd.PersistentFlags().GetBool("create-config"); createConfig {
			fmt.Println("設定ファイル作成を開始します...")

			execPath, err := os.Executable()
			if err != nil {
				fmt.Fprintf(os.Stderr, "実行ファイルパスの取得エラー: %v\n", err)
				os.Exit(1)
			}
			execDir := filepath.Dir(execPath)
			configPath := filepath.Join(execDir, ".gopier.yaml")
			fmt.Printf("設定ファイルパス: %s\n", configPath)

			if err := createDefaultConfig(configPath); err != nil {
				fmt.Fprintf(os.Stderr, "設定ファイル作成エラー: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("設定ファイルを作成しました: %s\n", configPath)
			fmt.Println("このファイルを編集してデフォルト設定をカスタマイズしてください。")
			return
		}

		// 設定表示フラグの確認
		if showConfig, _ := cmd.PersistentFlags().GetBool("show-config"); showConfig {
			showCurrentConfig()
			return
		}

		if sourceDir == "" || destDir == "" {
			cmd.Help()
			return
		}

		// デフォルトのワーカー数はCPUコア数
		if numWorkers <= 0 {
			numWorkers = runtime.NumCPU()
		}

		// ロガーの初期化
		log := logger.NewLogger(logFile, verbose, !noProgress)
		defer log.Close()

		// フィルターの設定
		fileFilter := filter.NewFilter(includePattern, excludePattern)

		// コピーオプションの設定
		options := copier.DefaultOptions()
		options.BufferSize = bufferSize * 1024 * 1024 // MBからバイトに変換
		options.Recursive = recursive
		options.MaxRetries = retryCount
		options.RetryDelay = time.Duration(retryWait) * time.Second
		options.MaxConcurrent = numWorkers
		options.OverwriteExisting = !skipNewer
		options.CreateDirs = true
		options.VerifyHash = verifyChanged || verifyAll

		// データベースの初期化（同期モードが指定されている場合）
		var syncDB *database.SyncDB
		if syncMode != "" && syncDBPath != "" {
			var err error
			syncModeEnum := database.NormalSync
			switch syncMode {
			case "initial":
				syncModeEnum = database.InitialSync
			case "incremental":
				syncModeEnum = database.IncrementalSync
			}
			syncDB, err = database.NewSyncDB(syncDBPath, syncModeEnum)
			if err != nil {
				fmt.Fprintf(os.Stderr, "データベース初期化エラー: %v\n", err)
				os.Exit(1)
			}
			defer syncDB.Close()
		}

		// 検証のみモードの場合
		if verifyOnly {
			verifierOptions := verifier.DefaultOptions()
			verifierOptions.Recursive = recursive
			verifierOptions.MaxConcurrent = numWorkers
			verifierOptions.BufferSize = bufferSize * 1024 * 1024

			v := verifier.NewVerifier(sourceDir, destDir, verifierOptions, fileFilter, syncDB)

			if verifyAll {
				// すべてのファイルを検証（最終検証）
				log.Info("すべてのファイルのハッシュ検証を開始します...")
				if err := v.Verify(); err != nil {
					fmt.Fprintf(os.Stderr, "検証中にエラーが発生しました: %v\n", err)
					os.Exit(1)
				}
				// レポート生成
				if finalReport != "" {
					if err := v.GenerateReport(finalReport); err != nil {
						fmt.Fprintf(os.Stderr, "レポート生成エラー: %v\n", err)
						os.Exit(1)
					}
				}
			} else {
				// 変更されたファイルのみ検証
				log.Info("変更されたファイルのハッシュ検証を開始します...")
				if err := v.Verify(); err != nil {
					fmt.Fprintf(os.Stderr, "検証中にエラーが発生しました: %v\n", err)
					os.Exit(1)
				}
			}
			return
		}

		// コピー実行
		fileCopier := copier.NewFileCopier(sourceDir, destDir, options, fileFilter, syncDB, log)
		err := fileCopier.CopyFiles()
		if err != nil {
			fmt.Fprintf(os.Stderr, "コピー中にエラーが発生しました: %v\n", err)
			os.Exit(1)
		}

		// コピー後に変更されたファイルのみ検証
		if verifyChanged {
			log.Info("同期したファイルのハッシュ検証を開始します...")
			verifierOptions := verifier.DefaultOptions()
			verifierOptions.Recursive = recursive
			verifierOptions.MaxConcurrent = numWorkers
			verifierOptions.BufferSize = bufferSize * 1024 * 1024

			v := verifier.NewVerifier(sourceDir, destDir, verifierOptions, fileFilter, syncDB)
			if err := v.Verify(); err != nil {
				fmt.Fprintf(os.Stderr, "検証中にエラーが発生しました: %v\n", err)
				os.Exit(1)
			}
		}

		// すべてのファイルを検証（最終検証）
		if verifyAll {
			log.Info("すべてのファイルのハッシュ検証を開始します...")
			verifierOptions := verifier.DefaultOptions()
			verifierOptions.Recursive = recursive
			verifierOptions.MaxConcurrent = numWorkers
			verifierOptions.BufferSize = bufferSize * 1024 * 1024

			v := verifier.NewVerifier(sourceDir, destDir, verifierOptions, fileFilter, syncDB)
			if err := v.Verify(); err != nil {
				fmt.Fprintf(os.Stderr, "検証中にエラーが発生しました: %v\n", err)
				os.Exit(1)
			}
			// レポート生成
			if finalReport != "" {
				if err := v.GenerateReport(finalReport); err != nil {
					fmt.Fprintf(os.Stderr, "レポート生成エラー: %v\n", err)
					os.Exit(1)
				}
			}
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// グローバル設定フラグ
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "設定ファイル (デフォルト: $HOME/.gopier.yaml)")
	rootCmd.PersistentFlags().Bool("create-config", false, "デフォルトの設定ファイルを作成")
	rootCmd.PersistentFlags().Bool("show-config", false, "現在の設定値を表示")

	// 基本オプション
	rootCmd.Flags().StringVarP(&sourceDir, "source", "s", "", "コピー元ディレクトリ (必須)")
	rootCmd.Flags().StringVarP(&destDir, "destination", "d", "", "コピー先ディレクトリ (必須)")
	rootCmd.Flags().StringVarP(&logFile, "log", "l", "", "ログファイルのパス")
	rootCmd.Flags().IntVarP(&numWorkers, "workers", "w", runtime.NumCPU(), "並列ワーカー数")
	rootCmd.Flags().IntVarP(&retryCount, "retry", "r", 3, "エラー時のリトライ回数")
	rootCmd.Flags().IntVarP(&retryWait, "wait", "", 5, "リトライ間の待機時間（秒）")
	rootCmd.Flags().StringVarP(&includePattern, "include", "i", "", "含めるファイルパターン（例: *.txt,*.docx）")
	rootCmd.Flags().StringVarP(&excludePattern, "exclude", "e", "", "除外するファイルパターン（例: *.tmp,*.bak）")
	rootCmd.Flags().BoolVarP(&mirror, "mirror", "m", false, "ミラーモード（宛先にない元ファイルを削除）")
	rootCmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "ドライラン（実際にはコピーしない）")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "詳細なログ出力")
	rootCmd.Flags().BoolVarP(&skipNewer, "skip-newer", "", false, "宛先の方が新しい場合はスキップ")
	rootCmd.Flags().BoolVarP(&noProgress, "no-progress", "", false, "進捗表示を無効化")
	rootCmd.Flags().IntVarP(&bufferSize, "buffer", "b", 8, "バッファサイズ（MB）")
	rootCmd.Flags().BoolVarP(&recursive, "recursive", "R", true, "サブディレクトリを再帰的にコピー")

	// 同期モード関連のフラグ
	rootCmd.Flags().StringVarP(&syncMode, "mode", "", "normal", "同期モード (initial:初期同期, incremental:追加同期)")
	rootCmd.Flags().StringVarP(&syncDBPath, "db", "", "sync_state.db", "同期状態データベースのパス")
	rootCmd.Flags().BoolVarP(&verifyOnly, "verify-only", "", false, "コピーせずに検証のみを実行")
	rootCmd.Flags().BoolVarP(&verifyChanged, "verify-changed", "", false, "同期したファイルのみハッシュ検証を実行")
	rootCmd.Flags().BoolVarP(&verifyAll, "verify-all", "", false, "すべてのファイルのハッシュ検証を実行（最終検証）")
	rootCmd.Flags().BoolVarP(&includeFailed, "include-failed", "", true, "前回までに失敗したファイルも同期する")
	rootCmd.Flags().IntVarP(&maxFailCount, "max-fail-count", "", 5, "最大失敗回数（これを超えるとスキップ、0は無制限）")
	rootCmd.Flags().StringVarP(&finalReport, "final-report", "", "", "最終検証レポートの出力パス")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// 設定ファイル作成フラグの確認
	if createConfig, _ := rootCmd.PersistentFlags().GetBool("create-config"); createConfig {
		// 設定ファイル作成時は設定ファイルの読み込みをスキップ
		return
	}

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// まず実行ファイルと同じディレクトリを探す
		exePath, err := os.Executable()
		if err == nil {
			exeDir := filepath.Dir(exePath)
			viper.AddConfigPath(exeDir)
		}
		// さらにホームディレクトリも探す
		home, err := os.UserHomeDir()
		if err == nil {
			viper.AddConfigPath(home)
		}
		viper.SetConfigType("yaml")
		viper.SetConfigName(".gopier")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		// 設定ファイルが見つかった場合のみログ出力
	} else {
		// 設定ファイルが見つからない場合は無視（デフォルト設定を使用）
	}

	loadConfig(rootCmd)
}

// validateConfig は設定値の妥当性をチェックする
func validateConfig(config *Config) error {
	var errors []string

	// パフォーマンス設定の検証
	if config.Workers < 1 {
		errors = append(errors, "workers: 1以上の値を指定してください")
	}
	if config.BufferSize < 1 {
		errors = append(errors, "buffer_size: 1以上の値を指定してください")
	}
	if config.RetryCount < 0 {
		errors = append(errors, "retry_count: 0以上の値を指定してください")
	}
	if config.RetryWait < 0 {
		errors = append(errors, "retry_wait: 0以上の値を指定してください")
	}

	// 同期設定の検証
	if config.SyncMode != "" && config.SyncMode != "normal" && config.SyncMode != "initial" && config.SyncMode != "incremental" {
		errors = append(errors, "sync_mode: normal, initial, incrementalのいずれかを指定してください")
	}
	if config.MaxFailCount < 0 {
		errors = append(errors, "max_fail_count: 0以上の値を指定してください")
	}

	// ハッシュ設定の検証
	if config.HashAlgorithm != "" {
		validAlgorithms := []string{"md5", "sha1", "sha256", "sha512"}
		valid := false
		for _, algo := range validAlgorithms {
			if config.HashAlgorithm == algo {
				valid = true
				break
			}
		}
		if !valid {
			errors = append(errors, "hash_algorithm: md5, sha1, sha256, sha512のいずれかを指定してください")
		}
	}

	// エラーがある場合はまとめて返す
	if len(errors) > 0 {
		return fmt.Errorf("設定ファイルにエラーがあります:\n%s", strings.Join(errors, "\n"))
	}

	return nil
}

// loadConfig は設定ファイルを読み込んでフラグにバインドする
func loadConfig(cmd *cobra.Command) {
	// 設定ファイルが既に読み込まれている場合は、その内容を処理
	if viper.ConfigFileUsed() != "" {
		// 設定ファイルの値を構造体に読み込み
		var config Config
		if err := viper.Unmarshal(&config); err != nil {
			fmt.Fprintf(os.Stderr, "設定ファイルの解析エラー: %v\n", err)
			return
		}

		// 設定値の妥当性チェック
		if err := validateConfig(&config); err != nil {
			fmt.Fprintf(os.Stderr, "設定ファイルの検証エラー: %v\n", err)
			return
		}

		// 設定ファイルの値をフラグにバインド（フラグが設定されていない場合のみ）
		bindConfigToFlags(&config, cmd)
	}
}

// bindConfigToFlags は設定ファイルの値をフラグにバインドする
func bindConfigToFlags(config *Config, cmd *cobra.Command) {
	// 基本設定
	if sourceDir == "" && config.Source != "" {
		sourceDir = config.Source
	}
	if destDir == "" && config.Destination != "" {
		destDir = config.Destination
	}
	if logFile == "" && config.LogFile != "" {
		logFile = config.LogFile
	}

	// パフォーマンス設定
	if numWorkers <= 0 && config.Workers > 0 {
		numWorkers = config.Workers
	}
	if bufferSize <= 0 && config.BufferSize > 0 {
		bufferSize = config.BufferSize
	}
	if retryCount <= 0 && config.RetryCount > 0 {
		retryCount = config.RetryCount
	}
	if retryWait <= 0 && config.RetryWait > 0 {
		retryWait = config.RetryWait
	}

	// フィルタ設定
	if includePattern == "" && config.IncludePattern != "" {
		includePattern = config.IncludePattern
	}
	if excludePattern == "" && config.ExcludePattern != "" {
		excludePattern = config.ExcludePattern
	}

	// 動作設定
	if !cmd.Flags().Changed("recursive") && config.Recursive {
		recursive = config.Recursive
	}
	if !cmd.Flags().Changed("mirror") && config.Mirror {
		mirror = config.Mirror
	}
	if !cmd.Flags().Changed("dry-run") && config.DryRun {
		dryRun = config.DryRun
	}
	if !cmd.Flags().Changed("verbose") && config.Verbose {
		verbose = config.Verbose
	}
	if !cmd.Flags().Changed("skip-newer") && config.SkipNewer {
		skipNewer = config.SkipNewer
	}
	if !cmd.Flags().Changed("no-progress") && config.NoProgress {
		noProgress = config.NoProgress
	}

	// 同期設定
	if syncMode == "" && config.SyncMode != "" {
		syncMode = config.SyncMode
	}
	if syncDBPath == "" && config.SyncDBPath != "" {
		syncDBPath = config.SyncDBPath
	}
	if !cmd.Flags().Changed("include-failed") && config.IncludeFailed {
		includeFailed = config.IncludeFailed
	}
	if maxFailCount <= 0 && config.MaxFailCount > 0 {
		maxFailCount = config.MaxFailCount
	}

	// 検証設定
	if !cmd.Flags().Changed("verify-only") && config.VerifyOnly {
		verifyOnly = config.VerifyOnly
	}
	if !cmd.Flags().Changed("verify-changed") && config.VerifyChanged {
		verifyChanged = config.VerifyChanged
	}
	if !cmd.Flags().Changed("verify-all") && config.VerifyAll {
		verifyAll = config.VerifyAll
	}
	if finalReport == "" && config.FinalReport != "" {
		finalReport = config.FinalReport
	}

	// ハッシュ設定
	if !cmd.Flags().Changed("verify-hash") && config.VerifyHash {
		// verifyHashフラグは存在しないので、verifyChangedまたはverifyAllに反映
		if !verifyChanged && !verifyAll {
			verifyChanged = config.VerifyChanged
		}
	}
}

// createDefaultConfig はデフォルトの設定ファイルを作成する
func createDefaultConfig(configPath string) error {
	config := Config{
		// パフォーマンス設定
		Workers:    runtime.NumCPU(),
		BufferSize: 8,
		RetryCount: 3,
		RetryWait:  5,

		// 動作設定
		Recursive:         true,
		Mirror:            false,
		DryRun:            false,
		Verbose:           false,
		SkipNewer:         false,
		NoProgress:        false,
		PreserveModTime:   true,
		OverwriteExisting: true,

		// 同期設定
		SyncMode:      "normal",
		SyncDBPath:    "sync_state.db",
		IncludeFailed: true,
		MaxFailCount:  5,

		// 検証設定
		VerifyOnly:    false,
		VerifyChanged: false,
		VerifyAll:     false,
		FinalReport:   "",

		// ハッシュ設定
		HashAlgorithm: "sha256",
		VerifyHash:    true,
	}

	// 設定値の妥当性チェック
	if err := validateConfig(&config); err != nil {
		return fmt.Errorf("デフォルト設定の検証エラー: %w", err)
	}

	// 設定ディレクトリの作成
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("設定ディレクトリの作成に失敗: %w", err)
	}

	// YAMLファイルとして保存
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("設定のマーシャルエラー: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("設定ファイルの作成エラー: %w", err)
	}

	return nil
}

// showCurrentConfig は現在の設定値を表示する
func showCurrentConfig() {
	config := Config{
		// 基本設定
		Source:      sourceDir,
		Destination: destDir,
		LogFile:     logFile,

		// パフォーマンス設定
		Workers:    numWorkers,
		BufferSize: bufferSize,
		RetryCount: retryCount,
		RetryWait:  retryWait,

		// フィルタ設定
		IncludePattern: includePattern,
		ExcludePattern: excludePattern,

		// 動作設定
		Recursive:         recursive,
		Mirror:            mirror,
		DryRun:            dryRun,
		Verbose:           verbose,
		SkipNewer:         skipNewer,
		NoProgress:        noProgress,
		PreserveModTime:   true, // デフォルト値
		OverwriteExisting: !skipNewer,

		// 同期設定
		SyncMode:      syncMode,
		SyncDBPath:    syncDBPath,
		IncludeFailed: includeFailed,
		MaxFailCount:  maxFailCount,

		// 検証設定
		VerifyOnly:    verifyOnly,
		VerifyChanged: verifyChanged,
		VerifyAll:     verifyAll,
		FinalReport:   finalReport,

		// ハッシュ設定
		HashAlgorithm: "sha256", // デフォルト値
		VerifyHash:    verifyChanged || verifyAll,
	}

	// YAML形式で出力
	data, err := yaml.Marshal(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "設定のマーシャルエラー: %v\n", err)
		return
	}

	fmt.Println("現在の設定値:")
	fmt.Println(string(data))
}

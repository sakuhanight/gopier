package copier

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sakuhanight/gopier/internal/database"
	"github.com/sakuhanight/gopier/internal/filter"
	"github.com/sakuhanight/gopier/internal/hasher"
	"github.com/sakuhanight/gopier/internal/logger"
	"github.com/sakuhanight/gopier/internal/permissions"
	"github.com/sakuhanight/gopier/internal/stats"
)

// CopyMode はコピーモードを表す型
type CopyMode int

const (
	// ModeCopy は通常のコピーモード
	ModeCopy CopyMode = iota
	// ModeVerify は検証のみのモード
	ModeVerify
	// ModeCopyAndVerify はコピーと検証を行うモード
	ModeCopyAndVerify
)

// ProgressCallback は進捗報告のためのコールバック関数型
type ProgressCallback func(current, total int64, currentFile string)

// Options はコピーオプションを表す構造体
type Options struct {
	BufferSize          int           // コピーバッファサイズ
	Recursive           bool          // 再帰的にコピーするかどうか
	PreserveModTime     bool          // 更新日時を保持するかどうか
	PreservePermissions bool          // ファイルアクセス権限を保持するかどうか（Windowsのみ）
	VerifyHash          bool          // ハッシュ検証を行うかどうか
	HashAlgorithm       string        // ハッシュアルゴリズム
	OverwriteExisting   bool          // 既存ファイルを上書きするかどうか
	CreateDirs          bool          // 必要なディレクトリを作成するかどうか
	MaxRetries          int           // 最大再試行回数
	RetryDelay          time.Duration // 再試行の遅延時間
	ProgressInterval    time.Duration // 進捗報告の間隔
	MaxConcurrent       int           // 最大並行コピー数
	Mode                CopyMode      // コピーモード
}

// DefaultOptions はデフォルトのオプションを返す
func DefaultOptions() Options {
	return Options{
		BufferSize:          32 * 1024 * 1024, // 32MB
		Recursive:           true,
		PreserveModTime:     true,
		PreservePermissions: false, // デフォルトでは無効（セキュリティ上の理由）
		VerifyHash:          true,
		HashAlgorithm:       string(hasher.SHA256),
		OverwriteExisting:   true,
		CreateDirs:          true,
		MaxRetries:          3,
		RetryDelay:          time.Second * 2,
		ProgressInterval:    time.Second * 1,
		MaxConcurrent:       4,
		Mode:                ModeCopy,
	}
}

// FileCopier はファイルコピー処理を管理する構造体
type FileCopier struct {
	sourceDir    string
	destDir      string
	options      Options
	stats        *stats.Stats
	filter       *filter.Filter
	hasher       *hasher.Hasher
	db           *database.SyncDB
	logger       *logger.Logger
	progressChan chan string
	progressFunc ProgressCallback
	wg           sync.WaitGroup
	semaphore    chan struct{}
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewFileCopier は新しいFileCopierを作成する
func NewFileCopier(sourceDir, destDir string, options Options, fileFilter *filter.Filter, syncDB *database.SyncDB, log *logger.Logger) *FileCopier {
	ctx, cancel := context.WithCancel(context.Background())

	// セマフォの初期化
	semaphore := make(chan struct{}, options.MaxConcurrent)

	// ハッシャーの初期化
	hashAlgo := hasher.Algorithm(options.HashAlgorithm)
	fileHasher := hasher.NewHasher(hashAlgo, options.BufferSize)

	return &FileCopier{
		sourceDir:    sourceDir,
		destDir:      destDir,
		options:      options,
		stats:        stats.NewStats(),
		filter:       fileFilter,
		hasher:       fileHasher,
		db:           syncDB,
		logger:       log,
		progressChan: make(chan string, 100),
		ctx:          ctx,
		cancel:       cancel,
		semaphore:    semaphore,
	}
}

// SetProgressCallback は進捗報告のコールバック関数を設定する
func (fc *FileCopier) SetProgressCallback(callback ProgressCallback) {
	fc.progressFunc = callback
}

// GetStats は現在の統計情報を返す
func (fc *FileCopier) GetStats() *stats.Stats {
	return fc.stats
}

// Cancel はコピー処理をキャンセルする
func (fc *FileCopier) Cancel() {
	fc.cancel()
}

// CopyFiles はファイルをコピーする
func (fc *FileCopier) CopyFiles() error {
	// 同期セッションの開始
	var sessionID int64
	var err error
	if fc.db != nil {
		sessionID, err = fc.db.StartSyncSession()
		if err != nil {
			// loggerでエラー出力
			if fc.logger != nil {
				if fc.logger.Verbose {
					fc.logger.Error("同期セッション開始エラー: %v", err)
				} else {
					fc.logger.Error("セッション開始失敗")
				}
			}
			return fmt.Errorf("同期セッション開始エラー: %w", err)
		}
	}

	// 進捗報告ゴルーチンの開始
	if fc.progressFunc != nil {
		go fc.reportProgress()
	}

	// ソースディレクトリの存在確認
	sourceInfo, err := os.Stat(fc.sourceDir)
	if err != nil {
		// loggerでエラー出力
		if fc.logger != nil {
			if fc.logger.Verbose {
				fc.logger.Error("ソースディレクトリ(%s)の確認エラー: %v", fc.sourceDir, err)
			} else {
				fc.logger.Error("ソースディレクトリ確認失敗")
			}
		}
		fc.stats.IncrementFailed()
		return fmt.Errorf("ソースディレクトリ(%s)の確認エラー: %w", fc.sourceDir, err)
	}

	// ソースがディレクトリの場合
	if sourceInfo.IsDir() {
		// 宛先ディレクトリの作成
		if fc.options.CreateDirs {
			if err := os.MkdirAll(fc.destDir, 0755); err != nil {
				// loggerでエラー出力
				if fc.logger != nil && fc.logger.Verbose {
					fc.logger.Error("宛先ディレクトリ(%s)の作成エラー: %v", fc.destDir, err)
				}
				return fmt.Errorf("宛先ディレクトリ(%s)の作成エラー: %w", fc.destDir, err)
			}

			// ディレクトリアクセス権限の保持（Windowsのみ）
			if fc.options.PreservePermissions {
				if permissions.CanCopyPermissions() {
					if err = permissions.CopyDirectoryPermissions(fc.sourceDir, fc.destDir); err != nil {
						// loggerでエラー出力（権限コピーエラーは致命的ではない）
						if fc.logger != nil && fc.logger.Verbose {
							fc.logger.Warn("ディレクトリ権限のコピーエラー: %s -> %s: %v", fc.sourceDir, fc.destDir, err)
						}
						// 権限コピーエラーは警告として記録するが、コピー処理は続行
					} else {
						// loggerで成功情報を出力
						if fc.logger != nil && fc.logger.Verbose {
							fc.logger.Info("ディレクトリ権限をコピーしました: %s", fc.destDir)
						}
					}
				} else {
					// loggerで警告出力
					if fc.logger != nil && fc.logger.Verbose {
						fc.logger.Warn("ディレクトリ権限のコピーは現在のプラットフォームではサポートされていません")
					}
				}
			}
		}

		// loggerで開始情報を出力
		if fc.logger != nil {
			if fc.logger.Verbose {
				fc.logger.Info("ディレクトリコピー開始: %s -> %s", fc.sourceDir, fc.destDir)
			} else {
				fc.logger.Info("ディレクトリコピー開始")
			}
		}

		// ディレクトリのコピー
		err = fc.copyDirectory(fc.sourceDir, fc.destDir)
	} else {
		// 単一ファイルのコピー
		destPath := filepath.Join(fc.destDir, filepath.Base(fc.sourceDir))

		// loggerで開始情報を出力
		if fc.logger != nil {
			if fc.logger.Verbose {
				fc.logger.Info("ファイルコピー開始: %s -> %s", fc.sourceDir, destPath)
			} else {
				fc.logger.Info("ファイルコピー開始")
			}
		}

		err = fc.copyFile(fc.sourceDir, destPath)
	}

	// すべてのゴルーチンの完了を待つ
	fc.wg.Wait()

	// チャンネルがまだ開いている場合のみ閉じる
	select {
	case <-fc.progressChan:
		// チャンネルは既に閉じられている
	default:
		close(fc.progressChan)
	}

	// 同期セッションの終了
	if fc.db != nil {
		endErr := fc.db.EndSyncSession(
			sessionID,
			int(fc.stats.GetCopiedCount()),
			int(fc.stats.GetSkippedCount()),
			int(fc.stats.GetFailedCount()),
			fc.stats.GetCopiedBytes(),
		)
		if endErr != nil {
			// セッション終了エラーはログに記録するが、元のエラーを返す
			if fc.logger != nil {
				if fc.logger.Verbose {
					fc.logger.Warn("同期セッション終了エラー: %v", endErr)
				} else {
					fc.logger.Warn("セッション終了エラー")
				}
			}
		}
	}

	// 完了情報を出力
	if fc.logger != nil {
		copiedCount := fc.stats.GetCopiedCount()
		skippedCount := fc.stats.GetSkippedCount()
		failedCount := fc.stats.GetFailedCount()
		copiedBytes := fc.stats.GetCopiedBytes()

		if fc.logger.Verbose {
			fc.logger.Info("コピー完了: コピー=%d, スキップ=%d, 失敗=%d, バイト=%d",
				copiedCount, skippedCount, failedCount, copiedBytes)
		} else {
			fc.logger.Info("コピー完了: %dファイル", copiedCount+skippedCount+failedCount)
		}
	}

	return err
}

// copyDirectory はディレクトリを再帰的にコピーする
func (fc *FileCopier) copyDirectory(sourceDir, destDir string) error {
	// コンテキストのキャンセル確認
	select {
	case <-fc.ctx.Done():
		return fmt.Errorf("コピー処理がキャンセルされました")
	default:
	}

	// ソースディレクトリを開く
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		// loggerでエラー出力
		if fc.logger != nil && fc.logger.Verbose {
			fc.logger.Error("ディレクトリ(%s)の読み込みエラー: %v", sourceDir, err)
		}
		return fmt.Errorf("ディレクトリ(%s)の読み込みエラー: %w", sourceDir, err)
	}

	// 宛先ディレクトリの作成
	if fc.options.CreateDirs {
		if err := os.MkdirAll(destDir, 0755); err != nil {
			// loggerでエラー出力
			if fc.logger != nil && fc.logger.Verbose {
				fc.logger.Error("宛先ディレクトリ(%s)の作成エラー: %v", destDir, err)
			}
			return fmt.Errorf("宛先ディレクトリ(%s)の作成エラー: %w", destDir, err)
		}

		// ディレクトリアクセス権限の保持（Windowsのみ）
		if fc.options.PreservePermissions {
			if permissions.CanCopyPermissions() {
				if err = permissions.CopyDirectoryPermissions(sourceDir, destDir); err != nil {
					// loggerでエラー出力（権限コピーエラーは致命的ではない）
					if fc.logger != nil && fc.logger.Verbose {
						fc.logger.Warn("ディレクトリ権限のコピーエラー: %s -> %s: %v", sourceDir, destDir, err)
					}
					// 権限コピーエラーは警告として記録するが、コピー処理は続行
				} else {
					// loggerで成功情報を出力
					if fc.logger != nil && fc.logger.Verbose {
						fc.logger.Info("ディレクトリ権限をコピーしました: %s", destDir)
					}
				}
			} else {
				// loggerで警告出力
				if fc.logger != nil && fc.logger.Verbose {
					fc.logger.Warn("ディレクトリ権限のコピーは現在のプラットフォームではサポートされていません")
				}
			}
		}
	}

	// 各エントリの処理
	for _, entry := range entries {
		sourcePath := filepath.Join(sourceDir, entry.Name())
		destPath := filepath.Join(destDir, entry.Name())

		// ディレクトリの場合
		if entry.IsDir() {
			if !fc.options.Recursive {
				continue
			}

			// 再帰的にコピー
			if err := fc.copyDirectory(sourcePath, destPath); err != nil {
				// loggerでエラー出力
				if fc.logger != nil && fc.logger.Verbose {
					fc.logger.Error("サブディレクトリ(%s)のコピーエラー: %v", sourcePath, err)
				}
				return err
			}
			continue
		}

		// ファイルの場合
		info, err := entry.Info()
		if err != nil {
			fc.stats.IncrementFailed()

			// loggerでエラー出力
			if fc.logger != nil && fc.logger.Verbose {
				fc.logger.Error("ファイル情報取得エラー: %s: %v", sourcePath, err)
			}
			return fmt.Errorf("ファイル情報取得エラー: %w", err)
		}

		// フィルタリング
		if fc.filter != nil && !fc.filter.ShouldInclude(sourcePath) {
			// ファイルをスキップ
			fc.stats.IncrementSkipped(info.Size())

			// データベースに記録
			if fc.db != nil {
				relPath, _ := filepath.Rel(fc.sourceDir, sourcePath)
				fileInfo := database.FileInfo{
					Path:         relPath,
					Size:         info.Size(),
					ModTime:      info.ModTime(),
					Status:       database.StatusSkipped,
					LastSyncTime: time.Now(),
					LastError:    "フィルタによりスキップ",
				}
				fc.db.AddFile(fileInfo)
			}

			// loggerでスキップ情報を出力
			if fc.logger != nil && fc.logger.Verbose {
				relPath, _ := filepath.Rel(fc.sourceDir, sourcePath)
				fc.logger.Info("ファイルをスキップ（フィルタ）: %s", relPath)
			}

			continue
		}

		// 非同期でファイルをコピー
		fc.wg.Add(1)
		go func(src, dst string) {
			defer fc.wg.Done()

			// セマフォの取得
			fc.semaphore <- struct{}{}
			defer func() {
				<-fc.semaphore
			}()

			if err := fc.copyFile(src, dst); err != nil {
				// loggerでエラー出力（非同期処理なので詳細は出力しない）
				if fc.logger != nil {
					relPath, _ := filepath.Rel(fc.sourceDir, src)
					fc.logger.Error("ファイルコピーエラー: %s", relPath)
				}
			}
		}(sourcePath, destPath)
	}

	return nil
}

// copyFile は単一ファイルをコピーする
func (fc *FileCopier) copyFile(sourcePath, destPath string) error {
	// コンテキストのキャンセル確認
	select {
	case <-fc.ctx.Done():
		return fmt.Errorf("コピー処理がキャンセルされました")
	default:
	}

	// 相対パスの計算
	relPath, err := filepath.Rel(fc.sourceDir, sourcePath)
	if err != nil {
		relPath = filepath.Base(sourcePath)
	}

	// 進捗報告
	if fc.progressFunc != nil && fc.progressChan != nil {
		select {
		case fc.progressChan <- relPath:
			// 正常に送信
		default:
			// チャンネルが閉じられているか、バッファが一杯
		case <-fc.ctx.Done():
			// コンテキストがキャンセルされた場合
			return fc.ctx.Err()
		}
	}

	// データベース内の既存ファイル情報を確認
	var fileInfo *database.FileInfo
	if fc.db != nil {
		fileInfo, err = fc.db.GetFile(relPath)
		if err != nil {
			if fc.logger != nil {
				if fc.logger.Verbose {
					fc.logger.Warn("データベース検索エラー: %v", err)
				}
			}
		}
	}

	// ソースファイルの情報を取得
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		fc.stats.IncrementFailed()

		// データベースに記録
		if fc.db != nil {
			errInfo := database.FileInfo{
				Path:         relPath,
				Status:       database.StatusFailed,
				LastSyncTime: time.Now(),
				LastError:    fmt.Sprintf("ソースファイル確認エラー: %v", err),
			}
			fc.db.AddFile(errInfo)
		}

		// loggerでエラー出力
		if fc.logger != nil {
			if fc.logger.Verbose {
				fc.logger.Error("ソースファイル(%s)の確認エラー: %v", sourcePath, err)
			} else {
				fc.logger.Error("ファイル確認失敗: %s", relPath)
			}
		}

		return fmt.Errorf("ソースファイル(%s)の確認エラー: %w", sourcePath, err)
	}

	// 検証モードの場合
	if fc.options.Mode == ModeVerify {
		return fc.verifyFile(sourcePath, destPath, relPath, sourceInfo)
	}

	// 宛先ファイルの存在確認
	destInfo, err := os.Stat(destPath)
	if err == nil {
		// 宛先ファイルが存在する場合

		// 上書きが許可されていない場合はスキップ
		if !fc.options.OverwriteExisting {
			fc.stats.IncrementSkipped(sourceInfo.Size())

			// データベースに記録
			if fc.db != nil {
				skipInfo := database.FileInfo{
					Path:         relPath,
					Size:         sourceInfo.Size(),
					ModTime:      sourceInfo.ModTime(),
					Status:       database.StatusSkipped,
					LastSyncTime: time.Now(),
					LastError:    "宛先ファイルが既に存在します",
				}
				fc.db.AddFile(skipInfo)
			}

			// loggerでスキップ情報を出力
			if fc.logger != nil {
				if fc.logger.Verbose {
					fc.logger.Info("ファイルをスキップ（上書き無効）: %s", relPath)
				}
			}

			return nil
		}

		// サイズと更新時刻が同じ場合はスキップ
		if sourceInfo.Size() == destInfo.Size() && sourceInfo.ModTime().Equal(destInfo.ModTime()) {
			fc.stats.IncrementSkipped(sourceInfo.Size())

			// データベースに記録
			if fc.db != nil {
				skipInfo := database.FileInfo{
					Path:         relPath,
					Size:         sourceInfo.Size(),
					ModTime:      sourceInfo.ModTime(),
					Status:       database.StatusSkipped,
					LastSyncTime: time.Now(),
				}
				fc.db.AddFile(skipInfo)
			}

			// loggerでスキップ情報を出力
			if fc.logger != nil {
				if fc.logger.Verbose {
					fc.logger.Info("ファイルをスキップ（内容同一）: %s", relPath)
				}
			}

			// 検証と同時コピーモードの場合は検証も行う
			if fc.options.Mode == ModeCopyAndVerify {
				return fc.verifyFile(sourcePath, destPath, relPath, sourceInfo)
			}

			return nil
		}
	} else if !os.IsNotExist(err) {
		// 存在確認でエラーが発生した場合（存在しない以外のエラー）
		fc.stats.IncrementFailed()

		// データベースに記録
		if fc.db != nil {
			errInfo := database.FileInfo{
				Path:         relPath,
				Size:         sourceInfo.Size(),
				ModTime:      sourceInfo.ModTime(),
				Status:       database.StatusFailed,
				LastSyncTime: time.Now(),
				LastError:    fmt.Sprintf("宛先ファイル確認エラー: %v", err),
			}
			fc.db.AddFile(errInfo)
		}

		// loggerでエラー出力
		if fc.logger != nil {
			if fc.logger.Verbose {
				fc.logger.Error("宛先ファイル(%s)の確認エラー: %v", destPath, err)
			} else {
				fc.logger.Error("宛先ファイル確認失敗: %s", relPath)
			}
		}

		return fmt.Errorf("宛先ファイル(%s)の確認エラー: %w", destPath, err)
	}

	// 宛先ディレクトリの作成
	if fc.options.CreateDirs {
		destDir := filepath.Dir(destPath)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			fc.stats.IncrementFailed()

			// データベースに記録
			if fc.db != nil {
				errInfo := database.FileInfo{
					Path:         relPath,
					Size:         sourceInfo.Size(),
					ModTime:      sourceInfo.ModTime(),
					Status:       database.StatusFailed,
					LastSyncTime: time.Now(),
					LastError:    fmt.Sprintf("宛先ディレクトリ作成エラー: %v", err),
				}
				fc.db.AddFile(errInfo)
			}

			// loggerでエラー出力
			if fc.logger != nil {
				if fc.logger.Verbose {
					fc.logger.Error("宛先ディレクトリ(%s)の作成エラー: %v", destDir, err)
				} else {
					fc.logger.Error("ディレクトリ作成失敗: %s", relPath)
				}
			}

			return fmt.Errorf("宛先ディレクトリ(%s)の作成エラー: %w", destDir, err)
		}
	}

	// ファイルのコピー（リトライロジック付き）
	var copyErr error
	for retry := 0; retry <= fc.options.MaxRetries; retry++ {
		if retry > 0 {
			// リトライ前に遅延
			time.Sleep(fc.options.RetryDelay)

			// loggerでリトライ情報を出力
			if fc.logger != nil {
				if fc.logger.Verbose {
					fc.logger.Warn("ファイル '%s' のコピーをリトライします (%d/%d): エラー: %v", relPath, retry, fc.options.MaxRetries, copyErr)
				} else {
					fc.logger.Warn("ファイル '%s' のコピーをリトライします (%d/%d)", relPath, retry, fc.options.MaxRetries)
				}
			}
		}

		// ファイルのコピー
		copyErr = fc.doCopyFile(sourcePath, destPath, sourceInfo)
		if copyErr == nil {
			break
		}
	}

	// すべてのリトライが失敗した場合
	if copyErr != nil {
		fc.stats.IncrementFailed()

		// データベースに記録
		if fc.db != nil {
			failCount := 0
			if fileInfo != nil {
				failCount = fileInfo.FailCount + 1
			}

			errInfo := database.FileInfo{
				Path:         relPath,
				Size:         sourceInfo.Size(),
				ModTime:      sourceInfo.ModTime(),
				Status:       database.StatusFailed,
				FailCount:    failCount,
				LastSyncTime: time.Now(),
				LastError:    fmt.Sprintf("ファイルコピーエラー: %v", copyErr),
			}
			fc.db.AddFile(errInfo)
		}

		// loggerでエラー出力
		if fc.logger != nil {
			if fc.logger.Verbose {
				fc.logger.Error("ファイル '%s' のコピーに失敗しました: %v", relPath, copyErr)
			} else {
				fc.logger.Error("コピー失敗: %s", relPath)
			}
		}

		return fmt.Errorf("ファイル '%s' のコピーに失敗しました: %w", relPath, copyErr)
	}

	// コピー成功の記録
	fc.stats.IncrementCopied(sourceInfo.Size())

	// データベースに記録
	if fc.db != nil {
		successInfo := database.FileInfo{
			Path:         relPath,
			Size:         sourceInfo.Size(),
			ModTime:      sourceInfo.ModTime(),
			Status:       database.StatusSuccess,
			LastSyncTime: time.Now(),
		}
		fc.db.AddFile(successInfo)
	}

	// loggerで成功情報を出力
	if fc.logger != nil {
		if fc.logger.Verbose {
			fc.logger.Info("ファイルコピー成功: %s (%d bytes)", relPath, sourceInfo.Size())
		} else {
			fc.logger.Info("コピー成功: %s", relPath)
		}
	}

	// 検証と同時コピーモードの場合は検証も行う
	if fc.options.Mode == ModeCopyAndVerify {
		return fc.verifyFile(sourcePath, destPath, relPath, sourceInfo)
	}

	return nil
}

// doCopyFile は実際のファイルコピー処理を行う
func (fc *FileCopier) doCopyFile(sourcePath, destPath string, sourceInfo os.FileInfo) error {
	// ソースファイルを開く
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		// loggerでエラー出力
		if fc.logger != nil && fc.logger.Verbose {
			fc.logger.Error("ソースファイル(%s)を開けません: %v", sourcePath, err)
		}
		return fmt.Errorf("ソースファイル(%s)を開けません: %w", sourcePath, err)
	}
	defer sourceFile.Close()

	// 宛先ファイルを作成
	destFile, err := os.Create(destPath)
	if err != nil {
		// loggerでエラー出力
		if fc.logger != nil && fc.logger.Verbose {
			fc.logger.Error("宛先ファイル(%s)を作成できません: %v", destPath, err)
		}
		return fmt.Errorf("宛先ファイル(%s)を作成できません: %w", destPath, err)
	}
	defer destFile.Close()

	// バッファを作成
	buffer := make([]byte, fc.options.BufferSize)

	// ファイルをコピー
	copiedBytes, err := io.CopyBuffer(destFile, sourceFile, buffer)
	if err != nil {
		// loggerでエラー出力
		if fc.logger != nil && fc.logger.Verbose {
			fc.logger.Error("ファイルコピーエラー: %s -> %s: %v", sourcePath, destPath, err)
		}
		return fmt.Errorf("ファイルコピーエラー: %w", err)
	}

	// コピーされたバイト数の確認
	if copiedBytes != sourceInfo.Size() {
		// loggerで警告出力
		if fc.logger != nil && fc.logger.Verbose {
			fc.logger.Warn("コピーされたバイト数が一致しません: 期待値=%d, 実際=%d", sourceInfo.Size(), copiedBytes)
		}
	}

	// ファイルを閉じる（エラーチェック付き）
	if err = destFile.Close(); err != nil {
		// loggerでエラー出力
		if fc.logger != nil && fc.logger.Verbose {
			fc.logger.Error("宛先ファイル(%s)を閉じられません: %v", destPath, err)
		}
		return fmt.Errorf("宛先ファイル(%s)を閉じられません: %w", destPath, err)
	}

	// 更新日時の保持
	if fc.options.PreserveModTime {
		if err = os.Chtimes(destPath, time.Now(), sourceInfo.ModTime()); err != nil {
			// loggerでエラー出力
			if fc.logger != nil && fc.logger.Verbose {
				fc.logger.Error("更新日時の設定エラー: %s: %v", destPath, err)
			}
			return fmt.Errorf("更新日時の設定エラー: %w", err)
		}
	}

	// ファイルアクセス権限の保持（Windowsのみ）
	if fc.options.PreservePermissions {
		if permissions.CanCopyPermissions() {
			if err = permissions.CopyFilePermissions(sourcePath, destPath); err != nil {
				// loggerでエラー出力（権限コピーエラーは致命的ではない）
				if fc.logger != nil && fc.logger.Verbose {
					fc.logger.Warn("ファイル権限のコピーエラー: %s -> %s: %v", sourcePath, destPath, err)
				}
				// 権限コピーエラーは警告として記録するが、コピー処理は続行
			} else {
				// loggerで成功情報を出力
				if fc.logger != nil && fc.logger.Verbose {
					fc.logger.Info("ファイル権限をコピーしました: %s", destPath)
				}
			}
		} else {
			// loggerで警告出力
			if fc.logger != nil && fc.logger.Verbose {
				fc.logger.Warn("ファイル権限のコピーは現在のプラットフォームではサポートされていません")
			}
		}
	}

	return nil
}

// verifyFile はファイルのハッシュ検証を行う
func (fc *FileCopier) verifyFile(sourcePath, destPath, relPath string, sourceInfo os.FileInfo) error {
	// ハッシュ検証が無効の場合はスキップ
	if !fc.options.VerifyHash {
		return nil
	}

	// 宛先ファイルの存在確認
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		// データベースに記録
		if fc.db != nil {
			errInfo := database.FileInfo{
				Path:         relPath,
				Size:         sourceInfo.Size(),
				ModTime:      sourceInfo.ModTime(),
				Status:       database.StatusMismatch,
				LastSyncTime: time.Now(),
				LastError:    "宛先ファイルが存在しません",
			}
			fc.db.AddFile(errInfo)
		}

		// loggerでエラー出力
		if fc.logger != nil {
			if fc.logger.Verbose {
				fc.logger.Error("宛先ファイル(%s)が存在しません", destPath)
			} else {
				fc.logger.Error("検証失敗: %s (宛先ファイルなし)", relPath)
			}
		}

		return fmt.Errorf("宛先ファイル '%s' が存在しません", destPath)
	}

	// ソースファイルのハッシュを計算
	sourceHash, err := fc.hasher.HashFile(sourcePath)
	if err != nil {
		// データベースに記録
		if fc.db != nil {
			errInfo := database.FileInfo{
				Path:         relPath,
				Size:         sourceInfo.Size(),
				ModTime:      sourceInfo.ModTime(),
				Status:       database.StatusFailed,
				LastSyncTime: time.Now(),
				LastError:    fmt.Sprintf("ソースハッシュ計算エラー: %v", err),
			}
			fc.db.AddFile(errInfo)
		}

		// loggerでエラー出力
		if fc.logger != nil {
			if fc.logger.Verbose {
				fc.logger.Error("ソースファイル(%s)のハッシュ計算エラー: %v", sourcePath, err)
			} else {
				fc.logger.Error("ハッシュ計算失敗: %s", relPath)
			}
		}

		return fmt.Errorf("ソースファイル(%s)のハッシュ計算エラー: %w", sourcePath, err)
	}

	// 宛先ファイルのハッシュを計算
	destHash, err := fc.hasher.HashFile(destPath)
	if err != nil {
		// データベースに記録
		if fc.db != nil {
			errInfo := database.FileInfo{
				Path:         relPath,
				Size:         sourceInfo.Size(),
				ModTime:      sourceInfo.ModTime(),
				Status:       database.StatusFailed,
				LastSyncTime: time.Now(),
				LastError:    fmt.Sprintf("宛先ハッシュ計算エラー: %v", err),
			}
			fc.db.AddFile(errInfo)
		}

		// loggerでエラー出力
		if fc.logger != nil {
			if fc.logger.Verbose {
				fc.logger.Error("宛先ファイル(%s)のハッシュ計算エラー: %v", destPath, err)
			} else {
				fc.logger.Error("ハッシュ計算失敗: %s", relPath)
			}
		}

		return fmt.Errorf("宛先ファイル(%s)のハッシュ計算エラー: %w", destPath, err)
	}

	// ハッシュ値をデータベースに記録
	if fc.db != nil {
		fc.db.UpdateFileHash(relPath, sourceHash, destHash)
	}

	// ハッシュ値の比較
	if sourceHash != destHash {
		// データベースに記録
		if fc.db != nil {
			errInfo := database.FileInfo{
				Path:         relPath,
				Size:         sourceInfo.Size(),
				ModTime:      sourceInfo.ModTime(),
				Status:       database.StatusMismatch,
				SourceHash:   sourceHash,
				DestHash:     destHash,
				LastSyncTime: time.Now(),
				LastError:    "ハッシュ値が一致しません",
			}
			fc.db.AddFile(errInfo)
		}

		// loggerでエラー出力
		if fc.logger != nil {
			if fc.logger.Verbose {
				fc.logger.Error("ファイル '%s' のハッシュ値が一致しません (ソース: %s, 宛先: %s)", relPath, sourceHash, destHash)
			} else {
				fc.logger.Error("ハッシュ不一致: %s", relPath)
			}
		}

		return fmt.Errorf("ファイル '%s' のハッシュ値が一致しません (ソース: %s, 宛先: %s)", relPath, sourceHash, destHash)
	}

	// 検証成功の記録
	if fc.db != nil {
		verifyInfo := database.FileInfo{
			Path:         relPath,
			Size:         sourceInfo.Size(),
			ModTime:      sourceInfo.ModTime(),
			Status:       database.StatusVerified,
			SourceHash:   sourceHash,
			DestHash:     destHash,
			LastSyncTime: time.Now(),
		}
		fc.db.AddFile(verifyInfo)
	}

	// loggerで成功情報を出力
	if fc.logger != nil {
		if fc.logger.Verbose {
			fc.logger.Info("ファイル検証成功: %s (ハッシュ: %s)", relPath, sourceHash)
		} else {
			fc.logger.Info("検証成功: %s", relPath)
		}
	}

	return nil
}

// reportProgress は進捗報告を行うゴルーチン
func (fc *FileCopier) reportProgress() {
	ticker := time.NewTicker(fc.options.ProgressInterval)
	defer ticker.Stop()

	var currentFile string

	for {
		select {
		case <-fc.ctx.Done():
			return
		case file, ok := <-fc.progressChan:
			if !ok {
				return
			}
			currentFile = file
		case <-ticker.C:
			if fc.progressFunc != nil {
				totalFiles, _, _ := fc.stats.GetProgressStats()
				fc.progressFunc(
					fc.stats.GetCopiedCount()+fc.stats.GetSkippedCount(),
					totalFiles,
					currentFile,
				)
			}
		}
	}
}

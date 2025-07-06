package verifier

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sakuhanight/gopier/internal/database"
	"github.com/sakuhanight/gopier/internal/filter"
	"github.com/sakuhanight/gopier/internal/hasher"
	"github.com/sakuhanight/gopier/internal/stats"
)

// ProgressCallback は進捗報告のためのコールバック関数型
type ProgressCallback func(current, total int64, currentFile string)

// Options は検証オプションを表す構造体
type Options struct {
	BufferSize       int           // ハッシュ計算のバッファサイズ
	Recursive        bool          // 再帰的に検証するかどうか
	HashAlgorithm    string        // ハッシュアルゴリズム
	ProgressInterval time.Duration // 進捗報告の間隔
	MaxConcurrent    int           // 最大並行検証数
	FailFast         bool          // 最初のエラーで停止するかどうか
	IgnoreMissing    bool          // 存在しないファイルを無視するかどうか
	IgnoreExtra      bool          // 余分なファイルを無視するかどうか
}

// DefaultOptions はデフォルトのオプションを返す
func DefaultOptions() Options {
	return Options{
		BufferSize:       32 * 1024 * 1024, // 32MB
		Recursive:        true,
		HashAlgorithm:    string(hasher.SHA256),
		ProgressInterval: time.Second * 1,
		MaxConcurrent:    4,
		FailFast:         false,
		IgnoreMissing:    false,
		IgnoreExtra:      false,
	}
}

// VerificationResult は検証結果を表す構造体
type VerificationResult struct {
	Path         string    // ファイルパス（相対パス）
	SourceExists bool      // ソースファイルが存在するかどうか
	DestExists   bool      // 宛先ファイルが存在するかどうか
	SizeMatch    bool      // サイズが一致するかどうか
	HashMatch    bool      // ハッシュが一致するかどうか
	SourceHash   string    // ソースファイルのハッシュ
	DestHash     string    // 宛先ファイルのハッシュ
	SourceSize   int64     // ソースファイルのサイズ
	DestSize     int64     // 宛先ファイルのサイズ
	SourceTime   time.Time // ソースファイルの更新時間
	DestTime     time.Time // 宛先ファイルの更新時間
	Error        error     // エラー情報
}

// Verifier はファイル検証処理を管理する構造体
type Verifier struct {
	sourceDir     string
	destDir       string
	options       Options
	stats         *stats.Stats
	filter        *filter.Filter
	hasher        *hasher.Hasher
	db            *database.SyncDB
	progressChan  chan string
	progressFunc  ProgressCallback
	wg            sync.WaitGroup
	semaphore     chan struct{}
	ctx           context.Context
	cancel        context.CancelFunc
	results       []VerificationResult
	resultsMutex  sync.Mutex
	errCount      int64
	errCountMutex sync.Mutex
}

// NewVerifier は新しいVerifierを作成する
func NewVerifier(sourceDir, destDir string, options Options, fileFilter *filter.Filter, syncDB *database.SyncDB) *Verifier {
	ctx, cancel := context.WithCancel(context.Background())

	// セマフォの初期化
	semaphore := make(chan struct{}, options.MaxConcurrent)

	// ハッシャーの初期化
	hashAlgo := hasher.Algorithm(options.HashAlgorithm)
	fileHasher := hasher.NewHasher(hashAlgo, options.BufferSize)

	return &Verifier{
		sourceDir:    sourceDir,
		destDir:      destDir,
		options:      options,
		stats:        stats.NewStats(),
		filter:       fileFilter,
		hasher:       fileHasher,
		db:           syncDB,
		progressChan: make(chan string, 100),
		ctx:          ctx,
		cancel:       cancel,
		semaphore:    semaphore,
		results:      make([]VerificationResult, 0),
	}
}

// SetProgressCallback は進捗報告のコールバック関数を設定する
func (v *Verifier) SetProgressCallback(callback ProgressCallback) {
	v.progressFunc = callback
}

// GetStats は現在の統計情報を返す
func (v *Verifier) GetStats() *stats.Stats {
	return v.stats
}

// Cancel は検証処理をキャンセルする
func (v *Verifier) Cancel() {
	v.cancel()
}

// SetTimeout はタイムアウト時間を設定する
func (v *Verifier) SetTimeout(timeout time.Duration) {
	if timeout > 0 {
		// 既存のコンテキストをキャンセル
		v.cancel()
		// 新しいタイムアウト付きコンテキストを作成
		v.ctx, v.cancel = context.WithTimeout(context.Background(), timeout)
	}
}

// GetResults は検証結果を返す
func (v *Verifier) GetResults() []VerificationResult {
	return v.results
}

// GetErrorCount はエラー数を返す
func (v *Verifier) GetErrorCount() int64 {
	v.errCountMutex.Lock()
	defer v.errCountMutex.Unlock()
	return v.errCount
}

// addResult は検証結果を追加する
func (v *Verifier) addResult(result VerificationResult) {
	v.resultsMutex.Lock()
	defer v.resultsMutex.Unlock()
	v.results = append(v.results, result)

	// エラーカウントの更新
	if result.Error != nil || !result.HashMatch || !result.SourceExists || !result.DestExists {
		v.errCountMutex.Lock()
		v.errCount++
		v.errCountMutex.Unlock()

		// 即時エラー停止が有効な場合
		if v.options.FailFast {
			v.cancel()
		}
	}
}

// Verify はファイルの検証を行う
func (v *Verifier) Verify() error {
	// コンテキストのキャンセル確認
	select {
	case <-v.ctx.Done():
		return fmt.Errorf("検証処理がキャンセルされました")
	default:
	}

	// 同期セッションの開始
	var sessionID int64
	var err error
	if v.db != nil {
		sessionID, err = v.db.StartSyncSession()
		if err != nil {
			return fmt.Errorf("同期セッション開始エラー: %w", err)
		}
	}

	// 進捗報告ゴルーチンの開始
	if v.progressFunc != nil {
		go v.reportProgress()
	}

	// ソースディレクトリの存在確認
	sourceInfo, err := os.Stat(v.sourceDir)
	if err != nil {
		return fmt.Errorf("ソースディレクトリの確認エラー: %w", err)
	}

	// ソースがディレクトリの場合
	if sourceInfo.IsDir() {
		// ディレクトリの検証
		err = v.verifyDirectory(v.sourceDir, v.destDir)

		// 余分なファイルのチェック（IgnoreExtraがfalseの場合）
		if err == nil && !v.options.IgnoreExtra {
			err = v.checkExtraFiles(v.sourceDir, v.destDir)
		}
	} else {
		// 単一ファイルの検証
		destPath := filepath.Join(v.destDir, filepath.Base(v.sourceDir))
		_, err = v.verifyFile(v.sourceDir, destPath)
	}

	// すべてのゴルーチンの完了を待つ
	v.wg.Wait()

	// チャンネルがまだ開いている場合のみ閉じる
	select {
	case <-v.progressChan:
		// チャンネルは既に閉じられている
	default:
		close(v.progressChan)
	}

	// 同期セッションの終了
	if v.db != nil {
		endErr := v.db.EndSyncSession(
			sessionID,
			0, // コピーされたファイル数
			int(v.stats.GetSkippedCount()),
			int(v.errCount),
			0, // コピーされたバイト数
		)
		if endErr != nil {
			// セッション終了エラーはログに記録するが、元のエラーを返す
			fmt.Printf("同期セッション終了エラー: %v\n", endErr)
		}
	}

	// エラーが発生したかどうかを返す
	if v.GetErrorCount() > 0 {
		return fmt.Errorf("%d 個のファイルで不一致が検出されました", v.GetErrorCount())
	}

	return err
}

// verifyDirectory はディレクトリを再帰的に検証する
func (v *Verifier) verifyDirectory(sourceDir, destDir string) error {
	// コンテキストのキャンセル確認
	select {
	case <-v.ctx.Done():
		return fmt.Errorf("検証処理がキャンセルされました")
	default:
	}

	// ソースディレクトリを開く
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return fmt.Errorf("ディレクトリ読み込みエラー: %w", err)
	}

	// 宛先ディレクトリの存在確認
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		if !v.options.IgnoreMissing {
			result := VerificationResult{
				Path:         destDir,
				SourceExists: true,
				DestExists:   false,
				Error:        fmt.Errorf("宛先ディレクトリが存在しません"),
			}
			v.addResult(result)
		}
		return nil
	}

	// 各エントリの処理
	for _, entry := range entries {
		sourcePath := filepath.Join(sourceDir, entry.Name())
		destPath := filepath.Join(destDir, entry.Name())

		// ディレクトリの場合
		if entry.IsDir() {
			if !v.options.Recursive {
				continue
			}

			// 再帰的に検証
			if err := v.verifyDirectory(sourcePath, destPath); err != nil {
				return err
			}
			continue
		}

		// ファイルの場合
		info, err := entry.Info()
		if err != nil {
			v.stats.IncrementFailed()
			result := VerificationResult{
				Path:  sourcePath,
				Error: fmt.Errorf("ファイル情報取得エラー: %w", err),
			}
			v.addResult(result)
			continue
		}

		// フィルタリング
		if v.filter != nil && !v.filter.ShouldInclude(sourcePath) {
			// ファイルをスキップ
			v.stats.IncrementSkipped(info.Size())
			continue
		}

		// 非同期でファイルを検証
		v.wg.Add(1)
		go func(src, dst string) {
			defer v.wg.Done()

			// セマフォの取得
			v.semaphore <- struct{}{}
			defer func() {
				<-v.semaphore
			}()

			result, err := v.verifyFile(src, dst)
			if err != nil {
				fmt.Printf("ファイル検証エラー: %v\n", err)
			}

			// 結果を追加
			if result != nil {
				v.addResult(*result)
			}
		}(sourcePath, destPath)
	}

	return nil
}

// verifyFile は単一ファイルを検証する
func (v *Verifier) verifyFile(sourcePath, destPath string) (*VerificationResult, error) {
	// コンテキストのキャンセル確認
	select {
	case <-v.ctx.Done():
		return nil, fmt.Errorf("検証処理がキャンセルされました")
	default:
	}

	// 相対パスの計算
	relPath, err := filepath.Rel(v.sourceDir, sourcePath)
	if err != nil {
		relPath = filepath.Base(sourcePath)
	}

	// 進捗報告
	if v.progressFunc != nil {
		select {
		case v.progressChan <- relPath:
			// 正常に送信
		default:
			// チャンネルが閉じられているか、バッファが一杯
		}
	}

	// 結果の初期化
	result := &VerificationResult{
		Path:         relPath,
		SourceExists: true,
		DestExists:   true,
		SizeMatch:    false,
		HashMatch:    false,
	}

	// ソースファイルの情報を取得
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		result.SourceExists = false
		result.Error = fmt.Errorf("ソースファイル確認エラー: %w", err)
		return result, nil
	}

	result.SourceSize = sourceInfo.Size()
	result.SourceTime = sourceInfo.ModTime()

	// 宛先ファイルの情報を取得
	destInfo, err := os.Stat(destPath)
	if err != nil {
		result.DestExists = false

		// 存在しないファイルを無視する場合
		if v.options.IgnoreMissing {
			v.stats.IncrementSkipped(sourceInfo.Size())
			return nil, nil
		}

		result.Error = fmt.Errorf("宛先ファイル確認エラー: %w", err)

		// データベースに記録
		if v.db != nil {
			fileInfo := database.FileInfo{
				Path:         relPath,
				Size:         sourceInfo.Size(),
				ModTime:      sourceInfo.ModTime(),
				Status:       database.StatusMismatch,
				LastSyncTime: time.Now(),
				LastError:    "宛先ファイルが存在しません",
			}
			v.db.AddFile(fileInfo)
		}

		return result, nil
	}

	result.DestSize = destInfo.Size()
	result.DestTime = destInfo.ModTime()

	// サイズの比較
	result.SizeMatch = sourceInfo.Size() == destInfo.Size()
	if !result.SizeMatch {
		result.Error = fmt.Errorf("ファイルサイズが一致しません (ソース: %d, 宛先: %d)", sourceInfo.Size(), destInfo.Size())

		// データベースに記録
		if v.db != nil {
			fileInfo := database.FileInfo{
				Path:         relPath,
				Size:         sourceInfo.Size(),
				ModTime:      sourceInfo.ModTime(),
				Status:       database.StatusMismatch,
				LastSyncTime: time.Now(),
				LastError:    fmt.Sprintf("ファイルサイズが一致しません (ソース: %d, 宛先: %d)", sourceInfo.Size(), destInfo.Size()),
			}
			v.db.AddFile(fileInfo)
		}

		return result, nil
	}

	// ソースファイルのハッシュを計算
	sourceHash, err := v.hasher.HashFile(sourcePath)
	if err != nil {
		result.Error = fmt.Errorf("ソースファイルのハッシュ計算エラー: %w", err)

		// データベースに記録
		if v.db != nil {
			fileInfo := database.FileInfo{
				Path:         relPath,
				Size:         sourceInfo.Size(),
				ModTime:      sourceInfo.ModTime(),
				Status:       database.StatusFailed,
				LastSyncTime: time.Now(),
				LastError:    fmt.Sprintf("ソースハッシュ計算エラー: %v", err),
			}
			v.db.AddFile(fileInfo)
		}

		return result, nil
	}

	result.SourceHash = sourceHash

	// 宛先ファイルのハッシュを計算
	destHash, err := v.hasher.HashFile(destPath)
	if err != nil {
		result.Error = fmt.Errorf("宛先ファイルのハッシュ計算エラー: %w", err)

		// データベースに記録
		if v.db != nil {
			fileInfo := database.FileInfo{
				Path:         relPath,
				Size:         sourceInfo.Size(),
				ModTime:      sourceInfo.ModTime(),
				Status:       database.StatusFailed,
				LastSyncTime: time.Now(),
				LastError:    fmt.Sprintf("宛先ハッシュ計算エラー: %v", err),
			}
			v.db.AddFile(fileInfo)
		}

		return result, nil
	}

	result.DestHash = destHash

	// ハッシュ値をデータベースに記録
	if v.db != nil {
		v.db.UpdateFileHash(relPath, sourceHash, destHash)
	}

	// ハッシュ値の比較
	result.HashMatch = sourceHash == destHash
	if !result.HashMatch {
		result.Error = fmt.Errorf("ハッシュ値が一致しません (ソース: %s, 宛先: %s)", sourceHash, destHash)

		// データベースに記録
		if v.db != nil {
			fileInfo := database.FileInfo{
				Path:         relPath,
				Size:         sourceInfo.Size(),
				ModTime:      sourceInfo.ModTime(),
				Status:       database.StatusMismatch,
				SourceHash:   sourceHash,
				DestHash:     destHash,
				LastSyncTime: time.Now(),
				LastError:    "ハッシュ値が一致しません",
			}
			v.db.AddFile(fileInfo)
		}

		return result, nil
	}

	// 検証成功の記録
	if v.db != nil {
		fileInfo := database.FileInfo{
			Path:         relPath,
			Size:         sourceInfo.Size(),
			ModTime:      sourceInfo.ModTime(),
			Status:       database.StatusVerified,
			SourceHash:   sourceHash,
			DestHash:     destHash,
			LastSyncTime: time.Now(),
		}
		v.db.AddFile(fileInfo)
	}

	return result, nil
}

// checkExtraFiles は宛先ディレクトリに余分なファイルがないかチェックする
func (v *Verifier) checkExtraFiles(sourceDir, destDir string) error {
	// 宛先ディレクトリを開く
	entries, err := os.ReadDir(destDir)
	if err != nil {
		return fmt.Errorf("宛先ディレクトリ読み込みエラー: %w", err)
	}

	// 各エントリの処理
	for _, entry := range entries {
		destPath := filepath.Join(destDir, entry.Name())
		sourcePath := filepath.Join(sourceDir, entry.Name())

		// ディレクトリの場合
		if entry.IsDir() {
			if !v.options.Recursive {
				continue
			}

			// ソースディレクトリの存在確認
			if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
				// 余分なディレクトリとして報告
				result := VerificationResult{
					Path:         destPath,
					SourceExists: false,
					DestExists:   true,
					Error:        fmt.Errorf("余分なディレクトリが存在します"),
				}
				v.addResult(result)
				continue
			}

			// 再帰的にチェック
			if err := v.checkExtraFiles(sourcePath, destPath); err != nil {
				return err
			}
			continue
		}

		// ファイルの場合
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// ソースファイルの存在確認
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			// フィルタリング
			if v.filter != nil && !v.filter.ShouldInclude(destPath) {
				// ファイルをスキップ
				continue
			}

			// 余分なファイルとして報告
			result := VerificationResult{
				Path:         destPath,
				SourceExists: false,
				DestExists:   true,
				DestSize:     info.Size(),
				DestTime:     info.ModTime(),
				Error:        fmt.Errorf("余分なファイルが存在します"),
			}
			v.addResult(result)

			// データベースに記録
			if v.db != nil {
				relPath, _ := filepath.Rel(v.destDir, destPath)
				fileInfo := database.FileInfo{
					Path:         relPath,
					Size:         info.Size(),
					ModTime:      info.ModTime(),
					Status:       database.StatusMismatch,
					LastSyncTime: time.Now(),
					LastError:    "ソースに存在しない余分なファイルです",
				}
				v.db.AddFile(fileInfo)
			}
		}
	}

	return nil
}

// reportProgress は進捗報告を行うゴルーチン
func (v *Verifier) reportProgress() {
	ticker := time.NewTicker(v.options.ProgressInterval)
	defer ticker.Stop()

	var currentFile string
	var processedFiles int64

	for {
		select {
		case <-v.ctx.Done():
			return
		case file, ok := <-v.progressChan:
			if !ok {
				return
			}
			currentFile = file
			processedFiles++
		case <-ticker.C:
			if v.progressFunc != nil {
				// 総ファイル数は不明なので、処理済みファイル数を報告
				v.progressFunc(
					processedFiles,
					-1, // 総ファイル数不明
					currentFile,
				)
			}
		}
	}
}

// GenerateReport は検証結果のレポートを生成する
func (v *Verifier) GenerateReport(reportPath string) error {
	// レポートディレクトリの作成
	reportDir := filepath.Dir(reportPath)
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		return fmt.Errorf("レポートディレクトリの作成に失敗: %w", err)
	}

	// ファイルを作成
	file, err := os.Create(reportPath)
	if err != nil {
		return fmt.Errorf("レポートファイル作成エラー: %w", err)
	}
	defer file.Close()

	// ヘッダー行を書き込む
	_, err = file.WriteString("ファイルパス,ソース存在,宛先存在,サイズ一致,ハッシュ一致,ソースハッシュ,宛先ハッシュ,ソースサイズ,宛先サイズ,ソース更新日時,宛先更新日時,エラー\n")
	if err != nil {
		return fmt.Errorf("ヘッダー書き込みエラー: %w", err)
	}

	// 結果を書き込む
	for _, result := range v.results {
		// エラーメッセージの整形
		errorMsg := ""
		if result.Error != nil {
			errorMsg = result.Error.Error()
		}

		line := fmt.Sprintf(
			"%s,%t,%t,%t,%t,%s,%s,%d,%d,%s,%s,%s\n",
			result.Path,
			result.SourceExists,
			result.DestExists,
			result.SizeMatch,
			result.HashMatch,
			result.SourceHash,
			result.DestHash,
			result.SourceSize,
			result.DestSize,
			result.SourceTime.Format(time.RFC3339),
			result.DestTime.Format(time.RFC3339),
			errorMsg,
		)
		_, err = file.WriteString(line)
		if err != nil {
			return fmt.Errorf("データ書き込みエラー: %w", err)
		}
	}

	return nil
}

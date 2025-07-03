package database

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.etcd.io/bbolt"
)

// SyncMode は同期モードを表す型
type SyncMode string

const (
	// InitialSync は初期同期モード
	InitialSync SyncMode = "initial"
	// IncrementalSync は追加同期モード
	IncrementalSync SyncMode = "incremental"
	// NormalSync は通常の同期モード
	NormalSync SyncMode = "normal"
)

// FileStatus はファイルの同期状態を表す型
type FileStatus string

const (
	// StatusPending は同期待ちの状態
	StatusPending FileStatus = "pending"
	// StatusSuccess は同期成功の状態
	StatusSuccess FileStatus = "success"
	// StatusFailed は同期失敗の状態
	StatusFailed FileStatus = "failed"
	// StatusSkipped は同期スキップの状態
	StatusSkipped FileStatus = "skipped"
	// StatusVerified は検証済みの状態
	StatusVerified FileStatus = "verified"
	// StatusMismatch はハッシュ不一致の状態
	StatusMismatch FileStatus = "mismatch"
)

// FileInfo はファイル情報を表す構造体
type FileInfo struct {
	Path         string     `json:"path"`           // ファイルパス（相対パス）
	Size         int64      `json:"size"`           // ファイルサイズ
	ModTime      time.Time  `json:"mod_time"`       // 最終更新時間
	Status       FileStatus `json:"status"`         // 同期状態
	SourceHash   string     `json:"source_hash"`    // ソースファイルのハッシュ
	DestHash     string     `json:"dest_hash"`      // 宛先ファイルのハッシュ
	FailCount    int        `json:"fail_count"`     // 失敗回数
	LastSyncTime time.Time  `json:"last_sync_time"` // 最終同期時間
	LastError    string     `json:"last_error"`     // 最後のエラーメッセージ
}

// SyncSession は同期セッション情報を表す構造体
type SyncSession struct {
	ID           int64     `json:"id"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Mode         string    `json:"mode"`
	FilesCopied  int       `json:"files_copied"`
	FilesSkipped int       `json:"files_skipped"`
	FilesFailed  int       `json:"files_failed"`
	BytesCopied  int64     `json:"bytes_copied"`
	Status       string    `json:"status"`
}

// SyncDB は同期状態データベースを管理する構造体
type SyncDB struct {
	db       *bbolt.DB
	dbPath   string
	syncMode SyncMode
}

// バケット名の定数
var (
	fileSyncBucket = []byte("file_sync")
	sessionBucket  = []byte("sync_session")
	statsBucket    = []byte("sync_stats")
)

// NewSyncDB は新しい同期データベースを作成する
func NewSyncDB(dbPath string, mode SyncMode) (*SyncDB, error) {
	// データベースディレクトリの作成
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("データベースディレクトリの作成に失敗: %w", err)
	}

	// BoltDBデータベースを開く
	db, err := bbolt.Open(dbPath, 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("データベース接続エラー: %w", err)
	}

	syncDB := &SyncDB{
		db:       db,
		dbPath:   dbPath,
		syncMode: mode,
	}

	// バケットの初期化
	if err := syncDB.initBuckets(); err != nil {
		db.Close()
		return nil, err
	}

	return syncDB, nil
}

// Close はデータベース接続を閉じる
func (s *SyncDB) Close() error {
	return s.db.Close()
}

// initBuckets はデータベースバケットを初期化する
func (s *SyncDB) initBuckets() error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		// ファイル同期状態バケット
		if _, err := tx.CreateBucketIfNotExists(fileSyncBucket); err != nil {
			return fmt.Errorf("ファイル同期バケット作成エラー: %w", err)
		}

		// 同期セッションバケット
		if _, err := tx.CreateBucketIfNotExists(sessionBucket); err != nil {
			return fmt.Errorf("セッションバケット作成エラー: %w", err)
		}

		// 統計情報バケット
		if _, err := tx.CreateBucketIfNotExists(statsBucket); err != nil {
			return fmt.Errorf("統計バケット作成エラー: %w", err)
		}

		return nil
	})
}

// ResetDatabase はデータベースをリセットする（初期同期モード用）
func (s *SyncDB) ResetDatabase() error {
	if s.syncMode != InitialSync {
		return fmt.Errorf("初期同期モードでのみデータベースをリセットできます")
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		// ファイル同期バケットを削除して再作成
		if err := tx.DeleteBucket(fileSyncBucket); err != nil {
			return fmt.Errorf("ファイル同期バケット削除エラー: %w", err)
		}
		if _, err := tx.CreateBucket(fileSyncBucket); err != nil {
			return fmt.Errorf("ファイル同期バケット再作成エラー: %w", err)
		}

		// 統計情報バケットをクリア
		if err := tx.DeleteBucket(statsBucket); err != nil {
			return fmt.Errorf("統計バケット削除エラー: %w", err)
		}
		if _, err := tx.CreateBucket(statsBucket); err != nil {
			return fmt.Errorf("統計バケット再作成エラー: %w", err)
		}

		return nil
	})
}

// AddFile はファイル情報をデータベースに追加する
func (s *SyncDB) AddFile(file FileInfo) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(fileSyncBucket)
		if bucket == nil {
			return fmt.Errorf("ファイル同期バケットが見つかりません")
		}

		// ファイル情報をJSONにシリアライズ
		data, err := json.Marshal(file)
		if err != nil {
			return fmt.Errorf("ファイル情報のシリアライズエラー: %w", err)
		}

		// キーとしてファイルパスを使用
		key := []byte(file.Path)
		if err := bucket.Put(key, data); err != nil {
			return fmt.Errorf("ファイル情報の保存エラー: %w", err)
		}

		return nil
	})
}

// GetFile はファイル情報をデータベースから取得する
func (s *SyncDB) GetFile(path string) (*FileInfo, error) {
	var fileInfo FileInfo

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(fileSyncBucket)
		if bucket == nil {
			return fmt.Errorf("ファイル同期バケットが見つかりません")
		}

		key := []byte(path)
		data := bucket.Get(key)
		if data == nil {
			return fmt.Errorf("ファイルが見つかりません: %s", path)
		}

		if err := json.Unmarshal(data, &fileInfo); err != nil {
			return fmt.Errorf("ファイル情報のデシリアライズエラー: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &fileInfo, nil
}

// UpdateFileStatus はファイルの状態を更新する
func (s *SyncDB) UpdateFileStatus(path string, status FileStatus, lastError string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(fileSyncBucket)
		if bucket == nil {
			return fmt.Errorf("ファイル同期バケットが見つかりません")
		}

		key := []byte(path)
		data := bucket.Get(key)
		if data == nil {
			// ファイルが存在しない場合は新規作成
			fileInfo := FileInfo{
				Path:         path,
				Status:       status,
				LastError:    lastError,
				LastSyncTime: time.Now(),
			}

			// 現在のトランザクション内で直接保存
			newData, err := json.Marshal(fileInfo)
			if err != nil {
				return fmt.Errorf("ファイル情報のシリアライズエラー: %w", err)
			}

			if err := bucket.Put(key, newData); err != nil {
				return fmt.Errorf("ファイル情報の保存エラー: %w", err)
			}

			return nil
		}

		// 既存のファイル情報を更新
		var fileInfo FileInfo
		if err := json.Unmarshal(data, &fileInfo); err != nil {
			return fmt.Errorf("ファイル情報のデシリアライズエラー: %w", err)
		}

		fileInfo.Status = status
		fileInfo.LastError = lastError
		fileInfo.LastSyncTime = time.Now()

		// 更新された情報を保存
		newData, err := json.Marshal(fileInfo)
		if err != nil {
			return fmt.Errorf("ファイル情報のシリアライズエラー: %w", err)
		}

		if err := bucket.Put(key, newData); err != nil {
			return fmt.Errorf("ファイル情報の更新エラー: %w", err)
		}

		return nil
	})
}

// UpdateFileHash はファイルのハッシュ情報を更新する
func (s *SyncDB) UpdateFileHash(path string, sourceHash, destHash string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(fileSyncBucket)
		if bucket == nil {
			return fmt.Errorf("ファイル同期バケットが見つかりません")
		}

		key := []byte(path)
		data := bucket.Get(key)
		if data == nil {
			return fmt.Errorf("ファイルが見つかりません: %s", path)
		}

		var fileInfo FileInfo
		if err := json.Unmarshal(data, &fileInfo); err != nil {
			return fmt.Errorf("ファイル情報のデシリアライズエラー: %w", err)
		}

		fileInfo.SourceHash = sourceHash
		fileInfo.DestHash = destHash
		fileInfo.LastSyncTime = time.Now()

		newData, err := json.Marshal(fileInfo)
		if err != nil {
			return fmt.Errorf("ファイル情報のシリアライズエラー: %w", err)
		}

		if err := bucket.Put(key, newData); err != nil {
			return fmt.Errorf("ファイル情報の更新エラー: %w", err)
		}

		return nil
	})
}

// IncrementFailCount はファイルの失敗回数を増加させる
func (s *SyncDB) IncrementFailCount(path string) (int, error) {
	var failCount int

	err := s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(fileSyncBucket)
		if bucket == nil {
			return fmt.Errorf("ファイル同期バケットが見つかりません")
		}

		key := []byte(path)
		data := bucket.Get(key)
		if data == nil {
			return fmt.Errorf("ファイルが見つかりません: %s", path)
		}

		var fileInfo FileInfo
		if err := json.Unmarshal(data, &fileInfo); err != nil {
			return fmt.Errorf("ファイル情報のデシリアライズエラー: %w", err)
		}

		fileInfo.FailCount++
		failCount = fileInfo.FailCount
		fileInfo.LastSyncTime = time.Now()

		newData, err := json.Marshal(fileInfo)
		if err != nil {
			return fmt.Errorf("ファイル情報のシリアライズエラー: %w", err)
		}

		if err := bucket.Put(key, newData); err != nil {
			return fmt.Errorf("ファイル情報の更新エラー: %w", err)
		}

		return nil
	})

	return failCount, err
}

// GetFailedFiles は失敗したファイルのリストを取得する
func (s *SyncDB) GetFailedFiles(maxFailCount int) ([]FileInfo, error) {
	var failedFiles []FileInfo

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(fileSyncBucket)
		if bucket == nil {
			return fmt.Errorf("ファイル同期バケットが見つかりません")
		}

		return bucket.ForEach(func(k, v []byte) error {
			var fileInfo FileInfo
			if err := json.Unmarshal(v, &fileInfo); err != nil {
				return fmt.Errorf("ファイル情報のデシリアライズエラー: %w", err)
			}

			// 失敗状態で、かつ最大失敗回数未満のファイルを追加
			if fileInfo.Status == StatusFailed && (maxFailCount == 0 || fileInfo.FailCount < maxFailCount) {
				failedFiles = append(failedFiles, fileInfo)
			}

			return nil
		})
	})

	return failedFiles, err
}

// GetFilesByStatus は指定された状態のファイルリストを取得する
func (s *SyncDB) GetFilesByStatus(status FileStatus) ([]FileInfo, error) {
	var files []FileInfo

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(fileSyncBucket)
		if bucket == nil {
			return fmt.Errorf("ファイル同期バケットが見つかりません")
		}

		return bucket.ForEach(func(k, v []byte) error {
			var fileInfo FileInfo
			if err := json.Unmarshal(v, &fileInfo); err != nil {
				return fmt.Errorf("ファイル情報のデシリアライズエラー: %w", err)
			}

			if fileInfo.Status == status {
				files = append(files, fileInfo)
			}

			return nil
		})
	})

	return files, err
}

// GetAllFiles はすべてのファイル情報を取得する
func (s *SyncDB) GetAllFiles() ([]FileInfo, error) {
	var files []FileInfo

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(fileSyncBucket)
		if bucket == nil {
			return fmt.Errorf("ファイル同期バケットが見つかりません")
		}

		return bucket.ForEach(func(k, v []byte) error {
			var fileInfo FileInfo
			if err := json.Unmarshal(v, &fileInfo); err != nil {
				return fmt.Errorf("ファイル情報のデシリアライズエラー: %w", err)
			}

			files = append(files, fileInfo)
			return nil
		})
	})

	return files, err
}

// StartSyncSession は新しい同期セッションを開始する
func (s *SyncDB) StartSyncSession() (int64, error) {
	var sessionID int64

	err := s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(sessionBucket)
		if bucket == nil {
			return fmt.Errorf("セッションバケットが見つかりません")
		}

		// セッションIDを生成（現在のタイムスタンプを使用）
		sessionID = time.Now().UnixNano()

		session := SyncSession{
			ID:        sessionID,
			StartTime: time.Now(),
			Mode:      string(s.syncMode),
			Status:    "running",
		}

		data, err := json.Marshal(session)
		if err != nil {
			return fmt.Errorf("セッション情報のシリアライズエラー: %w", err)
		}

		key := []byte(fmt.Sprintf("%d", sessionID))
		if err := bucket.Put(key, data); err != nil {
			return fmt.Errorf("セッション情報の保存エラー: %w", err)
		}

		return nil
	})

	return sessionID, err
}

// EndSyncSession は同期セッションを終了する
func (s *SyncDB) EndSyncSession(sessionID int64, filesCopied, filesSkipped, filesFailed int, bytesCopied int64) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(sessionBucket)
		if bucket == nil {
			return fmt.Errorf("セッションバケットが見つかりません")
		}

		key := []byte(fmt.Sprintf("%d", sessionID))
		data := bucket.Get(key)
		if data == nil {
			return fmt.Errorf("セッションが見つかりません: %d", sessionID)
		}

		var session SyncSession
		if err := json.Unmarshal(data, &session); err != nil {
			return fmt.Errorf("セッション情報のデシリアライズエラー: %w", err)
		}

		session.EndTime = time.Now()
		session.FilesCopied = filesCopied
		session.FilesSkipped = filesSkipped
		session.FilesFailed = filesFailed
		session.BytesCopied = bytesCopied
		session.Status = "completed"

		newData, err := json.Marshal(session)
		if err != nil {
			return fmt.Errorf("セッション情報のシリアライズエラー: %w", err)
		}

		if err := bucket.Put(key, newData); err != nil {
			return fmt.Errorf("セッション情報の更新エラー: %w", err)
		}

		return nil
	})
}

// GetSyncStats は同期統計情報を取得する
func (s *SyncDB) GetSyncStats() (map[string]int, error) {
	stats := make(map[string]int)

	err := s.db.View(func(tx *bbolt.Tx) error {
		// ファイル同期バケットから統計を取得
		fileBucket := tx.Bucket(fileSyncBucket)
		if fileBucket == nil {
			return fmt.Errorf("ファイル同期バケットが見つかりません")
		}

		var totalFiles, successFiles, failedFiles, skippedFiles, pendingFiles int

		err := fileBucket.ForEach(func(k, v []byte) error {
			var fileInfo FileInfo
			if err := json.Unmarshal(v, &fileInfo); err != nil {
				return fmt.Errorf("ファイル情報のデシリアライズエラー: %w", err)
			}

			totalFiles++
			switch fileInfo.Status {
			case StatusSuccess:
				successFiles++
			case StatusFailed:
				failedFiles++
			case StatusSkipped:
				skippedFiles++
			case StatusPending:
				pendingFiles++
			}

			return nil
		})

		if err != nil {
			return err
		}

		stats["total_files"] = totalFiles
		stats["success_files"] = successFiles
		stats["failed_files"] = failedFiles
		stats["skipped_files"] = skippedFiles
		stats["pending_files"] = pendingFiles

		return nil
	})

	return stats, err
}

// ExportVerificationReport は検証レポートをエクスポートする
func (s *SyncDB) ExportVerificationReport(reportPath string) error {
	files, err := s.GetAllFiles()
	if err != nil {
		return fmt.Errorf("ファイル情報の取得エラー: %w", err)
	}

	// レポートデータを構造化
	report := struct {
		ExportTime time.Time  `json:"export_time"`
		TotalFiles int        `json:"total_files"`
		Files      []FileInfo `json:"files"`
	}{
		ExportTime: time.Now(),
		TotalFiles: len(files),
		Files:      files,
	}

	// JSONファイルとして保存
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("レポートのシリアライズエラー: %w", err)
	}

	if err := os.WriteFile(reportPath, data, 0644); err != nil {
		return fmt.Errorf("レポートファイルの保存エラー: %w", err)
	}

	return nil
}

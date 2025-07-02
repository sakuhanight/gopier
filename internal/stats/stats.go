package stats

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// Stats は同期処理の統計情報を管理する構造体
type Stats struct {
	FilesCopied  int64 // コピーしたファイル数
	FilesSkipped int64 // スキップしたファイル数
	FilesFailed  int64 // 失敗したファイル数
	BytesCopied  int64 // コピーしたバイト数
	BytesSkipped int64 // スキップしたバイト数
	mu           sync.Mutex
}

// NewStats は新しい統計情報オブジェクトを作成する
func NewStats() *Stats {
	return &Stats{}
}

// IncrementCopied はコピーしたファイル数とバイト数を増加させる
func (s *Stats) IncrementCopied(bytes int64) {
	atomic.AddInt64(&s.FilesCopied, 1)
	atomic.AddInt64(&s.BytesCopied, bytes)
}

// IncrementSkipped はスキップしたファイル数とバイト数を増加させる
func (s *Stats) IncrementSkipped(bytes int64) {
	atomic.AddInt64(&s.FilesSkipped, 1)
	atomic.AddInt64(&s.BytesSkipped, bytes)
}

// IncrementFailed は失敗したファイル数を増加させる
func (s *Stats) IncrementFailed() {
	atomic.AddInt64(&s.FilesFailed, 1)
}

// GetCopiedCount はコピーしたファイル数を取得する
func (s *Stats) GetCopiedCount() int64 {
	return atomic.LoadInt64(&s.FilesCopied)
}

// GetSkippedCount はスキップしたファイル数を取得する
func (s *Stats) GetSkippedCount() int64 {
	return atomic.LoadInt64(&s.FilesSkipped)
}

// GetFailedCount は失敗したファイル数を取得する
func (s *Stats) GetFailedCount() int64 {
	return atomic.LoadInt64(&s.FilesFailed)
}

// GetCopiedBytes はコピーしたバイト数を取得する
func (s *Stats) GetCopiedBytes() int64 {
	return atomic.LoadInt64(&s.BytesCopied)
}

// GetSkippedBytes はスキップしたバイト数を取得する
func (s *Stats) GetSkippedBytes() int64 {
	return atomic.LoadInt64(&s.BytesSkipped)
}

// GetTotalFiles は処理したファイルの合計数を取得する
func (s *Stats) GetTotalFiles() int64 {
	return s.GetCopiedCount() + s.GetSkippedCount() + s.GetFailedCount()
}

// GetTotalBytes は処理したバイトの合計数を取得する
func (s *Stats) GetTotalBytes() int64 {
	return s.GetCopiedBytes() + s.GetSkippedBytes()
}

// String はStats構造体の文字列表現を返す
func (s *Stats) String() string {
	return fmt.Sprintf(
		"コピー: %d ファイル (%s), スキップ: %d ファイル (%s), 失敗: %d ファイル",
		s.GetCopiedCount(), formatBytes(s.GetCopiedBytes()),
		s.GetSkippedCount(), formatBytes(s.GetSkippedBytes()),
		s.GetFailedCount(),
	)
}

// GetProgressStats は進捗表示用の統計情報を取得する
func (s *Stats) GetProgressStats() (int64, int64, float64) {
	totalFiles := s.GetTotalFiles()
	totalBytes := s.GetTotalBytes()

	var progressPercent float64
	if totalFiles > 0 {
		progressPercent = float64(s.GetCopiedCount()+s.GetSkippedCount()) / float64(totalFiles) * 100
	}

	return totalFiles, totalBytes, progressPercent
}

// Reset は統計情報をリセットする
func (s *Stats) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	atomic.StoreInt64(&s.FilesCopied, 0)
	atomic.StoreInt64(&s.FilesSkipped, 0)
	atomic.StoreInt64(&s.FilesFailed, 0)
	atomic.StoreInt64(&s.BytesCopied, 0)
	atomic.StoreInt64(&s.BytesSkipped, 0)
}

// formatBytes はバイト数を読みやすい形式にフォーマットする
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

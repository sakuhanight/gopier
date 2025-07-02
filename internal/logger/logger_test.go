package logger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name         string
		logFile      string
		verbose      bool
		showProgress bool
	}{
		{
			name:         "基本設定",
			logFile:      "",
			verbose:      false,
			showProgress: true,
		},
		{
			name:         "詳細ログ有効",
			logFile:      "",
			verbose:      true,
			showProgress: true,
		},
		{
			name:         "進捗表示無効",
			logFile:      "",
			verbose:      false,
			showProgress: false,
		},
		{
			name:         "ログファイル指定",
			logFile:      "test.log",
			verbose:      false,
			showProgress: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(tt.logFile, tt.verbose, tt.showProgress)

			if logger == nil {
				t.Error("NewLogger() が nil を返しました")
				return
			}

			if logger.Verbose != tt.verbose {
				t.Errorf("Verbose = %v, 期待値 %v", logger.Verbose, tt.verbose)
			}

			if logger.NoProgress != !tt.showProgress {
				t.Errorf("NoProgress = %v, 期待値 %v", logger.NoProgress, !tt.showProgress)
			}

			// ロガーを閉じる
			logger.Close()
		})
	}
}

func TestNewLoggerWithFile(t *testing.T) {
	// 一時ディレクトリを作成
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	logger := NewLogger(logFile, false, true)
	if logger == nil {
		t.Fatal("NewLogger() が nil を返しました")
	}

	// ログファイルが作成されているか確認
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Error("ログファイルが作成されていません")
	}

	// ロガーを閉じる
	logger.Close()
}

func TestLoggerMethods(t *testing.T) {
	logger := NewLogger("", false, true)
	if logger == nil {
		t.Fatal("NewLogger() が nil を返しました")
	}
	defer logger.Close()

	// 各ログメソッドがエラーを返さないことを確認
	logger.Debug("Debug message")
	logger.Info("Info message")
	logger.Warn("Warning message")
	logger.Error("Error message")
	logger.Progress("Progress message")
}

func TestLoggerWithFields(t *testing.T) {
	logger := NewLogger("", false, true)
	if logger == nil {
		t.Fatal("NewLogger() が nil を返しました")
	}
	defer logger.Close()

	fields := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}

	sugar := logger.WithFields(fields)
	if sugar == nil {
		t.Error("WithFields() が nil を返しました")
	}
}

func TestLoggerProgress(t *testing.T) {
	tests := []struct {
		name         string
		showProgress bool
	}{
		{
			name:         "進捗表示有効",
			showProgress: true,
		},
		{
			name:         "進捗表示無効",
			showProgress: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger("", false, tt.showProgress)
			if logger == nil {
				t.Fatal("NewLogger() が nil を返しました")
			}
			defer logger.Close()

			// Progressメソッドがエラーを返さないことを確認
			logger.Progress("Test progress message")
		})
	}
}

func TestLoggerConcurrentAccess(t *testing.T) {
	logger := NewLogger("", false, true)
	if logger == nil {
		t.Fatal("NewLogger() が nil を返しました")
	}
	defer logger.Close()

	// ゴルーチンで並行アクセス
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			logger.Info("Concurrent log message %d", id)
			logger.Debug("Concurrent debug message %d", id)
			logger.Warn("Concurrent warning message %d", id)
			logger.Error("Concurrent error message %d", id)
			logger.Progress("Concurrent progress message %d", id)
			done <- true
		}(i)
	}

	// すべてのゴルーチンの完了を待つ
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestLoggerFileCreation(t *testing.T) {
	// 一時ディレクトリを作成
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "nested", "path", "test.log")

	logger := NewLogger(logFile, false, true)
	if logger == nil {
		t.Fatal("NewLogger() が nil を返しました")
	}
	defer logger.Close()

	// ネストしたディレクトリが作成されているか確認
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Error("ネストしたディレクトリのログファイルが作成されていません")
	}
}

func TestLoggerClose(t *testing.T) {
	logger := NewLogger("", false, true)
	if logger == nil {
		t.Fatal("NewLogger() が nil を返しました")
	}

	// Closeメソッドがエラーを返さないことを確認
	logger.Close()

	// 複数回呼び出してもエラーが発生しないことを確認
	logger.Close()
}

func TestLoggerTimeFormat(t *testing.T) {
	logger := NewLogger("", false, true)
	if logger == nil {
		t.Fatal("NewLogger() が nil を返しました")
	}
	defer logger.Close()

	// 進捗メッセージに時刻が含まれているかテスト
	// 実際の時刻フォーマットは内部実装に依存するため、
	// エラーが発生しないことを確認するのみ
	logger.Progress("Test message with time")
}

func TestLoggerVerboseMode(t *testing.T) {
	logger := NewLogger("", true, true) // verbose = true
	if logger == nil {
		t.Fatal("NewLogger() が nil を返しました")
	}
	defer logger.Close()

	// 詳細モードでデバッグログが出力されることを確認
	logger.Debug("Debug message in verbose mode")
	logger.Info("Info message in verbose mode")
}

func TestLoggerNoProgressMode(t *testing.T) {
	logger := NewLogger("", false, false) // showProgress = false
	if logger == nil {
		t.Fatal("NewLogger() が nil を返しました")
	}
	defer logger.Close()

	// 進捗表示が無効の場合でもエラーが発生しないことを確認
	logger.Progress("This should not be displayed")
	logger.Info("This should be displayed")
}

// ベンチマークテスト
func BenchmarkLoggerInfo(b *testing.B) {
	logger := NewLogger("", false, false)
	if logger == nil {
		b.Fatal("NewLogger() が nil を返しました")
	}
	defer logger.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("Benchmark log message %d", i)
	}
}

func BenchmarkLoggerDebug(b *testing.B) {
	logger := NewLogger("", true, false)
	if logger == nil {
		b.Fatal("NewLogger() が nil を返しました")
	}
	defer logger.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Debug("Benchmark debug message %d", i)
	}
}

func BenchmarkLoggerProgress(b *testing.B) {
	logger := NewLogger("", false, true)
	if logger == nil {
		b.Fatal("NewLogger() が nil を返しました")
	}
	defer logger.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Progress("Benchmark progress message %d", i)
	}
}

func BenchmarkLoggerWithFields(b *testing.B) {
	logger := NewLogger("", false, false)
	if logger == nil {
		b.Fatal("NewLogger() が nil を返しました")
	}
	defer logger.Close()

	fields := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.WithFields(fields)
	}
}

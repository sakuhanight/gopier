package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger はzapロガーをラップする構造体
type Logger struct {
	zap        *zap.Logger
	sugar      *zap.SugaredLogger
	Verbose    bool
	NoProgress bool
	mu         sync.Mutex
	lastLine   string
	file       *os.File // ファイルハンドルを保持
}

// NewLogger は新しいロガーを作成する
func NewLogger(logFile string, verbose bool, showProgress bool) *Logger {
	// ログレベルの設定
	level := zapcore.InfoLevel
	if verbose {
		level = zapcore.DebugLevel
	}

	// エンコーダーの設定
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 出力先の設定
	var cores []zapcore.Core

	// コンソール出力
	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
	consoleCore := zapcore.NewCore(
		consoleEncoder,
		zapcore.Lock(os.Stdout),
		level,
	)
	cores = append(cores, consoleCore)

	// ファイル出力（指定されている場合）
	var file *os.File
	if logFile != "" {
		// ディレクトリの作成
		logDir := filepath.Dir(logFile)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "ログディレクトリの作成に失敗: %v\n", err)
		} else {
			// ファイルオープン
			var err error
			file, err = os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ログファイルのオープンに失敗: %v\n", err)
			} else {
				fileEncoder := zapcore.NewJSONEncoder(encoderConfig)
				fileCore := zapcore.NewCore(
					fileEncoder,
					zapcore.AddSync(file),
					level,
				)
				cores = append(cores, fileCore)
			}
		}
	}

	// コアの結合
	core := zapcore.NewTee(cores...)

	// ロガーの作成
	zapLogger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return &Logger{
		zap:        zapLogger,
		sugar:      zapLogger.Sugar(),
		Verbose:    verbose,
		NoProgress: !showProgress,
		file:       file,
	}
}

// Close はロガーを閉じる
func (l *Logger) Close() {
	_ = l.zap.Sync()

	// ファイルハンドルを明示的に閉じる
	if l.file != nil {
		l.file.Close()
		l.file = nil
	}
}

// Debug はデバッグレベルのログを出力する
func (l *Logger) Debug(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 進捗表示を消去
	l.clearProgress()

	// ログ出力
	l.sugar.Debugf(format, args...)
}

// Info は情報レベルのログを出力する
func (l *Logger) Info(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 進捗表示を消去
	l.clearProgress()

	// ログ出力
	l.sugar.Infof(format, args...)
}

// Warn は警告レベルのログを出力する
func (l *Logger) Warn(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 進捗表示を消去
	l.clearProgress()

	// ログ出力
	l.sugar.Warnf(format, args...)
}

// Error はエラーレベルのログを出力する
func (l *Logger) Error(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 進捗表示を消去
	l.clearProgress()

	// ログ出力
	l.sugar.Errorf(format, args...)
}

// Fatal は致命的エラーレベルのログを出力する
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 進捗表示を消去
	l.clearProgress()

	// ログ出力
	l.sugar.Fatalf(format, args...)
}

// Progress は進捗情報を出力する（上書き可能）
func (l *Logger) Progress(format string, args ...interface{}) {
	if l.NoProgress {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// 現在の時刻を追加
	now := time.Now().Format("15:04:05")
	message := fmt.Sprintf("[%s] %s", now, fmt.Sprintf(format, args...))

	// 前の行を消去して新しい進捗を表示
	fmt.Print("\r\033[K") // カーソルを行頭に移動して行をクリア
	fmt.Print(message)

	l.lastLine = message
}

// clearProgress は進捗表示を消去する
func (l *Logger) clearProgress() {
	if l.NoProgress || l.lastLine == "" {
		return
	}

	fmt.Print("\r\033[K") // カーソルを行頭に移動して行をクリア
	l.lastLine = ""
}

// WithFields は構造化ログ用のフィールドを追加する
func (l *Logger) WithFields(fields map[string]interface{}) *zap.SugaredLogger {
	zapFields := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		zapFields = append(zapFields, k, v)
	}
	return l.sugar.With(zapFields...)
}

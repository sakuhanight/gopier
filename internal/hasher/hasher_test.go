package hasher

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestNewHasher(t *testing.T) {
	tests := []struct {
		name        string
		algorithm   Algorithm
		bufferSize  int
		expectError bool
	}{
		{
			name:        "MD5 with default buffer",
			algorithm:   MD5,
			bufferSize:  0,
			expectError: false,
		},
		{
			name:        "SHA1 with custom buffer",
			algorithm:   SHA1,
			bufferSize:  1024,
			expectError: false,
		},
		{
			name:        "SHA256 with large buffer",
			algorithm:   SHA256,
			bufferSize:  1024 * 1024,
			expectError: false,
		},
		{
			name:        "Negative buffer size",
			algorithm:   SHA256,
			bufferSize:  -1,
			expectError: false, // デフォルト値が使用される
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasher := NewHasher(tt.algorithm, tt.bufferSize)

			if hasher == nil {
				t.Error("NewHasher() が nil を返しました")
				return
			}

			if hasher.algorithm != tt.algorithm {
				t.Errorf("algorithm = %v, 期待値 %v", hasher.algorithm, tt.algorithm)
			}

			// バッファサイズが0以下の場合はデフォルト値が使用される
			expectedBufferSize := tt.bufferSize
			if tt.bufferSize <= 0 {
				expectedBufferSize = 32 * 1024 * 1024 // 32MB
			}

			if hasher.bufferSize != expectedBufferSize {
				t.Errorf("bufferSize = %d, 期待値 %d", hasher.bufferSize, expectedBufferSize)
			}
		})
	}
}

func TestGetHasher(t *testing.T) {
	tests := []struct {
		name        string
		algorithm   Algorithm
		expectError bool
	}{
		{
			name:        "MD5",
			algorithm:   MD5,
			expectError: false,
		},
		{
			name:        "SHA1",
			algorithm:   SHA1,
			expectError: false,
		},
		{
			name:        "SHA256",
			algorithm:   SHA256,
			expectError: false,
		},
		{
			name:        "Unknown algorithm",
			algorithm:   "unknown",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasher := NewHasher(tt.algorithm, 1024)
			result, err := hasher.getHasher()

			if tt.expectError {
				if err == nil {
					t.Error("期待されるエラーが発生しませんでした")
				}
				return
			}

			if err != nil {
				t.Errorf("予期しないエラー: %v", err)
				return
			}

			if result == nil {
				t.Error("getHasher() が nil を返しました")
			}
		})
	}
}

func TestHashFile(t *testing.T) {
	// テスト用の一時ファイルを作成
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello, World! This is a test file for hashing."

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("テストファイルの作成に失敗: %v", err)
	}

	tests := []struct {
		name        string
		algorithm   Algorithm
		filePath    string
		expectError bool
	}{
		{
			name:        "MD5 hash",
			algorithm:   MD5,
			filePath:    testFile,
			expectError: false,
		},
		{
			name:        "SHA1 hash",
			algorithm:   SHA1,
			filePath:    testFile,
			expectError: false,
		},
		{
			name:        "SHA256 hash",
			algorithm:   SHA256,
			filePath:    testFile,
			expectError: false,
		},
		{
			name:        "Non-existent file",
			algorithm:   SHA256,
			filePath:    "non-existent-file.txt",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasher := NewHasher(tt.algorithm, 1024)
			result, err := hasher.HashFile(tt.filePath)

			if tt.expectError {
				if err == nil {
					t.Error("期待されるエラーが発生しませんでした")
				}
				return
			}

			if err != nil {
				t.Errorf("予期しないエラー: %v", err)
				return
			}

			if result == "" {
				t.Error("HashFile() が空文字列を返しました")
			}

			// ハッシュ値の長さを確認
			expectedLength := getExpectedHashLength(tt.algorithm)
			if len(result) != expectedLength {
				t.Errorf("ハッシュ値の長さ = %d, 期待値 %d", len(result), expectedLength)
			}

			// 16進数文字列として有効かチェック
			_, err = hex.DecodeString(result)
			if err != nil {
				t.Errorf("ハッシュ値が有効な16進数文字列ではありません: %v", err)
			}
		})
	}
}

func TestVerifyFileHash(t *testing.T) {
	// テスト用の一時ファイルを作成
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello, World! This is a test file for verification."

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("テストファイルの作成に失敗: %v", err)
	}

	// 正しいハッシュ値を計算
	hasher := NewHasher(SHA256, 1024)
	correctHash, err := hasher.HashFile(testFile)
	if err != nil {
		t.Fatalf("正しいハッシュ値の計算に失敗: %v", err)
	}

	tests := []struct {
		name         string
		filePath     string
		expectedHash string
		expectMatch  bool
		expectError  bool
	}{
		{
			name:         "Correct hash",
			filePath:     testFile,
			expectedHash: correctHash,
			expectMatch:  true,
			expectError:  false,
		},
		{
			name:         "Incorrect hash",
			filePath:     testFile,
			expectedHash: "incorrecthash",
			expectMatch:  false,
			expectError:  false,
		},
		{
			name:         "Non-existent file",
			filePath:     "non-existent-file.txt",
			expectedHash: correctHash,
			expectMatch:  false,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := hasher.VerifyFileHash(tt.filePath, tt.expectedHash)

			if tt.expectError {
				if err == nil {
					t.Error("期待されるエラーが発生しませんでした")
				}
				return
			}

			if err != nil {
				t.Errorf("予期しないエラー: %v", err)
				return
			}

			if result != tt.expectMatch {
				t.Errorf("VerifyFileHash() = %v, 期待値 %v", result, tt.expectMatch)
			}
		})
	}
}

func TestHashDirectory(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir := t.TempDir()

	// テストファイルを作成
	files := map[string]string{
		"file1.txt": "Content of file 1",
		"file2.txt": "Content of file 2",
		"file3.doc": "Content of file 3",
	}

	for filename, content := range files {
		filePath := filepath.Join(tempDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("テストファイル %s の作成に失敗: %v", filename, err)
		}
	}

	// サブディレクトリを作成
	subDir := filepath.Join(tempDir, "subdir")
	err := os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("サブディレクトリの作成に失敗: %v", err)
	}

	// サブディレクトリにファイルを作成
	subFile := filepath.Join(subDir, "subfile.txt")
	err = os.WriteFile(subFile, []byte("Subdirectory file content"), 0644)
	if err != nil {
		t.Fatalf("サブディレクトリファイルの作成に失敗: %v", err)
	}

	tests := []struct {
		name        string
		recursive   bool
		expectCount int
	}{
		{
			name:        "Non-recursive",
			recursive:   false,
			expectCount: 3, // トップレベルのファイルのみ
		},
		{
			name:        "Recursive",
			recursive:   true,
			expectCount: 4, // トップレベル + サブディレクトリ
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasher := NewHasher(SHA256, 1024)
			result, err := hasher.HashDirectory(tempDir, tt.recursive)

			if err != nil {
				t.Errorf("HashDirectory() エラー: %v", err)
				return
			}

			if len(result) != tt.expectCount {
				t.Errorf("ハッシュ結果の数 = %d, 期待値 %d", len(result), tt.expectCount)
			}

			// 各ファイルのハッシュ値が空でないことを確認
			for filePath, hash := range result {
				if hash == "" {
					t.Errorf("ファイル %s のハッシュ値が空です", filePath)
				}
			}
		})
	}
}

func TestCompareDirectories(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	sourceDir := t.TempDir()
	destDir := t.TempDir()

	// ソースディレクトリにファイルを作成
	sourceFiles := map[string]string{
		"file1.txt": "Content of file 1",
		"file2.txt": "Content of file 2",
		"file3.txt": "Content of file 3",
	}

	for filename, content := range sourceFiles {
		filePath := filepath.Join(sourceDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("ソースファイル %s の作成に失敗: %v", filename, err)
		}
	}

	// 宛先ディレクトリにファイルを作成（一部異なる内容）
	destFiles := map[string]string{
		"file1.txt": "Content of file 1", // 同じ内容
		"file2.txt": "Different content", // 異なる内容
		// file3.txt は存在しない
		"file4.txt": "Extra file", // 追加のファイル
	}

	for filename, content := range destFiles {
		filePath := filepath.Join(destDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("宛先ファイル %s の作成に失敗: %v", filename, err)
		}
	}

	hasher := NewHasher(SHA256, 1024)
	mismatchedFiles, err := hasher.CompareDirectories(sourceDir, destDir, false)

	if err != nil {
		t.Errorf("CompareDirectories() エラー: %v", err)
		return
	}

	// 期待される不一致ファイルの数
	expectedMismatches := 3 // file2.txt (ハッシュ不一致), file3.txt (宛先に存在しない), file4.txt (ソースに存在しない)

	if len(mismatchedFiles) != expectedMismatches {
		t.Errorf("不一致ファイルの数 = %d, 期待値 %d", len(mismatchedFiles), expectedMismatches)
	}
}

func TestGetAlgorithmName(t *testing.T) {
	tests := []struct {
		name      string
		algorithm Algorithm
		expected  string
	}{
		{
			name:      "MD5",
			algorithm: MD5,
			expected:  "md5",
		},
		{
			name:      "SHA1",
			algorithm: SHA1,
			expected:  "sha1",
		},
		{
			name:      "SHA256",
			algorithm: SHA256,
			expected:  "sha256",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasher := NewHasher(tt.algorithm, 1024)
			result := hasher.GetAlgorithmName()

			if result != tt.expected {
				t.Errorf("GetAlgorithmName() = %s, 期待値 %s", result, tt.expected)
			}
		})
	}
}

// ヘルパー関数
func getExpectedHashLength(algorithm Algorithm) int {
	switch algorithm {
	case MD5:
		return 32 // MD5は128ビット = 16バイト = 32文字の16進数
	case SHA1:
		return 40 // SHA1は160ビット = 20バイト = 40文字の16進数
	case SHA256:
		return 64 // SHA256は256ビット = 32バイト = 64文字の16進数
	default:
		return 0
	}
}

// ベンチマークテスト
func BenchmarkHashFile(b *testing.B) {
	// テスト用の一時ファイルを作成
	tempDir := b.TempDir()
	testFile := filepath.Join(tempDir, "benchmark.txt")
	testContent := "This is a benchmark test file with some content to hash."

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		b.Fatalf("ベンチマークファイルの作成に失敗: %v", err)
	}

	hasher := NewHasher(SHA256, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := hasher.HashFile(testFile)
		if err != nil {
			b.Fatalf("HashFile() エラー: %v", err)
		}
	}
}

func BenchmarkVerifyFileHash(b *testing.B) {
	// テスト用の一時ファイルを作成
	tempDir := b.TempDir()
	testFile := filepath.Join(tempDir, "benchmark.txt")
	testContent := "This is a benchmark test file for verification."

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		b.Fatalf("ベンチマークファイルの作成に失敗: %v", err)
	}

	hasher := NewHasher(SHA256, 1024)
	correctHash, err := hasher.HashFile(testFile)
	if err != nil {
		b.Fatalf("正しいハッシュ値の計算に失敗: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := hasher.VerifyFileHash(testFile, correctHash)
		if err != nil {
			b.Fatalf("VerifyFileHash() エラー: %v", err)
		}
	}
}

package hasher

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
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

// TestHashDirectory_EdgeCases はHashDirectory関数のエッジケースをテスト
func TestHashDirectory_EdgeCases(t *testing.T) {
	tempDir := t.TempDir()

	// 空のディレクトリ
	emptyDir := filepath.Join(tempDir, "empty")
	os.MkdirAll(emptyDir, 0755)

	// 複数のファイルを含むディレクトリ
	multiFileDir := filepath.Join(tempDir, "multifile")
	os.MkdirAll(multiFileDir, 0755)

	// 異なるサイズのファイルを作成
	files := []struct {
		name    string
		content string
	}{
		{"small.txt", "small content"},
		{"medium.txt", "medium content " + string(make([]byte, 1000))},
		{"large.txt", "large content " + string(make([]byte, 10000))},
	}

	for _, file := range files {
		filePath := filepath.Join(multiFileDir, file.name)
		err := os.WriteFile(filePath, []byte(file.content), 0644)
		if err != nil {
			t.Fatalf("テストファイルの作成に失敗: %v", err)
		}
	}

	// サブディレクトリを含むディレクトリ
	subDir := filepath.Join(multiFileDir, "subdir")
	os.MkdirAll(subDir, 0755)
	subFile := filepath.Join(subDir, "subfile.txt")
	os.WriteFile(subFile, []byte("subdir content"), 0644)

	hasher := NewHasher(SHA256, 1024)

	// 空のディレクトリのハッシュ
	emptyHash, err := hasher.HashDirectory(emptyDir, true)
	if err != nil {
		t.Errorf("空のディレクトリのハッシュ計算が失敗: %v", err)
	}
	// 空のディレクトリは空のマップを返す
	if len(emptyHash) != 0 {
		t.Error("空のディレクトリのハッシュが空ではありません")
	}

	// 複数ファイルを含むディレクトリのハッシュ
	multiHash, err := hasher.HashDirectory(multiFileDir, true)
	if err != nil {
		t.Errorf("複数ファイルディレクトリのハッシュ計算が失敗: %v", err)
	}
	if len(multiHash) == 0 {
		t.Error("複数ファイルディレクトリのハッシュが空です")
	}

	// 存在しないディレクトリ
	_, err = hasher.HashDirectory(filepath.Join(tempDir, "nonexistent"), true)
	if err == nil {
		t.Error("存在しないディレクトリでエラーが発生しませんでした")
	}

	// ファイルをディレクトリとして指定（エラーが発生する）
	_, err = hasher.HashDirectory(filepath.Join(multiFileDir, "small.txt"), true)
	if err == nil {
		t.Error("ファイルをディレクトリとして指定した場合にエラーが発生しませんでした")
	}
}

// TestCompareDirectories_EdgeCases はCompareDirectories関数のエッジケースをテスト
func TestCompareDirectories_EdgeCases(t *testing.T) {
	tempDir := t.TempDir()

	// 同じ内容のディレクトリ
	dir1 := filepath.Join(tempDir, "dir1")
	dir2 := filepath.Join(tempDir, "dir2")
	os.MkdirAll(dir1, 0755)
	os.MkdirAll(dir2, 0755)

	// 同じファイルを作成
	testContent := "test content"
	file1 := filepath.Join(dir1, "test.txt")
	file2 := filepath.Join(dir2, "test.txt")
	os.WriteFile(file1, []byte(testContent), 0644)
	os.WriteFile(file2, []byte(testContent), 0644)

	// 異なる内容のディレクトリ
	dir3 := filepath.Join(tempDir, "dir3")
	os.MkdirAll(dir3, 0755)
	file3 := filepath.Join(dir3, "test.txt")
	os.WriteFile(file3, []byte("different content"), 0644)

	// 空のディレクトリ
	emptyDir1 := filepath.Join(tempDir, "empty1")
	emptyDir2 := filepath.Join(tempDir, "empty2")
	os.MkdirAll(emptyDir1, 0755)
	os.MkdirAll(emptyDir2, 0755)

	hasher := NewHasher(SHA256, 1024)

	// 同じ内容のディレクトリの比較
	mismatches, err := hasher.CompareDirectories(dir1, dir2, false)
	if err != nil {
		t.Errorf("同じ内容ディレクトリの比較が失敗: %v", err)
	}
	if len(mismatches) > 0 {
		t.Errorf("同じ内容のディレクトリに差分があります: %v", mismatches)
	}

	// 異なる内容のディレクトリの比較
	mismatches, err = hasher.CompareDirectories(dir1, dir3, false)
	if err != nil {
		t.Errorf("異なる内容ディレクトリの比較が失敗: %v", err)
	}
	if len(mismatches) == 0 {
		t.Error("異なる内容のディレクトリに差分がありません")
	}

	// 空のディレクトリの比較
	mismatches, err = hasher.CompareDirectories(emptyDir1, emptyDir2, false)
	if err != nil {
		t.Errorf("空ディレクトリの比較が失敗: %v", err)
	}
	if len(mismatches) > 0 {
		t.Error("空のディレクトリに差分があります")
	}

	// 存在しないディレクトリとの比較
	_, err = hasher.CompareDirectories(dir1, filepath.Join(tempDir, "nonexistent"), false)
	if err == nil {
		t.Error("存在しないディレクトリとの比較でエラーが発生しませんでした")
	}

	// ファイルとディレクトリの比較（エラーが発生する）
	_, err = hasher.CompareDirectories(file1, dir1, false)
	if err == nil {
		t.Error("ファイルとディレクトリの比較でエラーが発生しませんでした")
	}
}

// TestHashFile_EdgeCases はHashFile関数のエッジケースをテスト
func TestHashFile_EdgeCases(t *testing.T) {
	tempDir := t.TempDir()

	// 空のファイル
	emptyFile := filepath.Join(tempDir, "empty.txt")
	os.WriteFile(emptyFile, []byte{}, 0644)

	// 大きなファイル
	largeFile := filepath.Join(tempDir, "large.txt")
	largeContent := make([]byte, 100*1024) // 100KB
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}
	os.WriteFile(largeFile, largeContent, 0644)

	// シンボリックリンク（Windows環境ではスキップ）
	var symlinkFile string
	if runtime.GOOS != "windows" {
		symlinkFile = filepath.Join(tempDir, "symlink.txt")
		err := os.Symlink(emptyFile, symlinkFile)
		if err != nil {
			t.Logf("シンボリックリンクの作成に失敗（権限不足の可能性）: %v", err)
			symlinkFile = ""
		}
	}

	hasher := NewHasher(SHA256, 1024)

	// 空のファイルのハッシュ
	emptyHash, err := hasher.HashFile(emptyFile)
	if err != nil {
		t.Errorf("空のファイルのハッシュ計算が失敗: %v", err)
	}
	if emptyHash == "" {
		t.Error("空のファイルのハッシュが空です")
	}

	// 大きなファイルのハッシュ
	largeHash, err := hasher.HashFile(largeFile)
	if err != nil {
		t.Errorf("大きなファイルのハッシュ計算が失敗: %v", err)
	}
	if largeHash == "" {
		t.Error("大きなファイルのハッシュが空です")
	}

	// シンボリックリンクのハッシュ（Windows環境ではスキップ）
	if symlinkFile != "" {
		symlinkHash, err := hasher.HashFile(symlinkFile)
		if err != nil {
			t.Errorf("シンボリックリンクのハッシュ計算が失敗: %v", err)
		}
		if symlinkHash == "" {
			t.Error("シンボリックリンクのハッシュが空です")
		}
	}

	// 存在しないファイル
	_, err = hasher.HashFile(filepath.Join(tempDir, "nonexistent.txt"))
	if err == nil {
		t.Error("存在しないファイルでエラーが発生しませんでした")
	}

	// ディレクトリをファイルとして指定
	_, err = hasher.HashFile(tempDir)
	if err == nil {
		t.Error("ディレクトリをファイルとして指定した場合にエラーが発生しませんでした")
	}
}

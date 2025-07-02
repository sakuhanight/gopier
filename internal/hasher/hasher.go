package hasher

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
)

// Algorithm はハッシュアルゴリズムの種類を表す型
type Algorithm string

const (
	// MD5 はMD5ハッシュアルゴリズム
	MD5 Algorithm = "md5"
	// SHA1 はSHA-1ハッシュアルゴリズム
	SHA1 Algorithm = "sha1"
	// SHA256 はSHA-256ハッシュアルゴリズム
	SHA256 Algorithm = "sha256"
)

// Hasher はファイルハッシュ計算を行う構造体
type Hasher struct {
	algorithm  Algorithm
	bufferSize int
}

// NewHasher は新しいハッシャーを作成する
func NewHasher(algorithm Algorithm, bufferSize int) *Hasher {
	// バッファサイズが0以下の場合はデフォルト値を使用
	if bufferSize <= 0 {
		bufferSize = 32 * 1024 * 1024 // 32MB
	}

	return &Hasher{
		algorithm:  algorithm,
		bufferSize: bufferSize,
	}
}

// getHasher は指定されたアルゴリズムのハッシャーを返す
func (h *Hasher) getHasher() (hash.Hash, error) {
	switch h.algorithm {
	case MD5:
		return md5.New(), nil
	case SHA1:
		return sha1.New(), nil
	case SHA256:
		return sha256.New(), nil
	default:
		return nil, fmt.Errorf("未サポートのハッシュアルゴリズム: %s", h.algorithm)
	}
}

// HashFile はファイルのハッシュ値を計算する
func (h *Hasher) HashFile(filePath string) (string, error) {
	// ファイルを開く
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("ファイルを開けません: %w", err)
	}
	defer file.Close()

	// ハッシャーを取得
	hasher, err := h.getHasher()
	if err != nil {
		return "", err
	}

	// バッファを作成
	buffer := make([]byte, h.bufferSize)

	// ファイルを読み込んでハッシュを計算
	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return "", fmt.Errorf("ファイル読み込みエラー: %w", err)
		}

		if n == 0 {
			break
		}

		hasher.Write(buffer[:n])
	}

	// ハッシュ値を16進数文字列に変換
	hashSum := hasher.Sum(nil)
	hashString := hex.EncodeToString(hashSum)

	return hashString, nil
}

// VerifyFileHash はファイルのハッシュ値が期待値と一致するかを検証する
func (h *Hasher) VerifyFileHash(filePath, expectedHash string) (bool, error) {
	actualHash, err := h.HashFile(filePath)
	if err != nil {
		return false, err
	}

	return actualHash == expectedHash, nil
}

// HashDirectory はディレクトリ内の全ファイルのハッシュを計算する
// 戻り値はファイルパス（ベースディレクトリからの相対パス）をキー、ハッシュ値を値とするマップ
func (h *Hasher) HashDirectory(dirPath string, recursive bool) (map[string]string, error) {
	// 追加: ディレクトリかどうかチェック
	info, err := os.Stat(dirPath)
	if err != nil {
		return nil, fmt.Errorf("パスの確認に失敗: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("指定されたパスはディレクトリではありません: %s", dirPath)
	}

	results := make(map[string]string)

	err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// ディレクトリはスキップ
		if info.IsDir() {
			// 再帰的でない場合は、トップレベルディレクトリのみ処理
			if !recursive && path != dirPath {
				return filepath.SkipDir
			}
			return nil
		}

		// ファイルのハッシュを計算
		hash, err := h.HashFile(path)
		if err != nil {
			return fmt.Errorf("ハッシュ計算エラー (%s): %w", path, err)
		}

		// 相対パスを計算
		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return fmt.Errorf("相対パス計算エラー: %w", err)
		}

		// 結果に追加
		results[relPath] = hash

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("ディレクトリ走査エラー: %w", err)
	}

	return results, nil
}

// CompareDirectories は2つのディレクトリのファイルハッシュを比較する
// 戻り値は不一致ファイルのリスト、エラー
func (h *Hasher) CompareDirectories(sourceDir, destDir string, recursive bool) ([]string, error) {
	// 追加: ディレクトリかどうかチェック
	info1, err := os.Stat(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("ソースパスの確認に失敗: %w", err)
	}
	if !info1.IsDir() {
		return nil, fmt.Errorf("ソースパスはディレクトリではありません: %s", sourceDir)
	}
	info2, err := os.Stat(destDir)
	if err != nil {
		return nil, fmt.Errorf("宛先パスの確認に失敗: %w", err)
	}
	if !info2.IsDir() {
		return nil, fmt.Errorf("宛先パスはディレクトリではありません: %s", destDir)
	}

	// ソースディレクトリのハッシュを計算
	sourceHashes, err := h.HashDirectory(sourceDir, recursive)
	if err != nil {
		return nil, fmt.Errorf("ソースディレクトリのハッシュ計算エラー: %w", err)
	}

	// 宛先ディレクトリのハッシュを計算
	destHashes, err := h.HashDirectory(destDir, recursive)
	if err != nil {
		return nil, fmt.Errorf("宛先ディレクトリのハッシュ計算エラー: %w", err)
	}

	// 不一致ファイルのリスト
	var mismatchedFiles []string

	// ソースファイルを確認
	for relPath, sourceHash := range sourceHashes {
		destHash, exists := destHashes[relPath]

		// 宛先に存在しない場合
		if !exists {
			mismatchedFiles = append(mismatchedFiles, relPath+" (宛先に存在しません)")
			continue
		}

		// ハッシュが一致しない場合
		if sourceHash != destHash {
			mismatchedFiles = append(mismatchedFiles, relPath+" (ハッシュ不一致)")
		}
	}

	// 宛先にのみ存在するファイルを確認
	for relPath := range destHashes {
		_, exists := sourceHashes[relPath]
		if !exists {
			mismatchedFiles = append(mismatchedFiles, relPath+" (ソースに存在しません)")
		}
	}

	return mismatchedFiles, nil
}

// GetAlgorithmName はハッシュアルゴリズムの名前を返す
func (h *Hasher) GetAlgorithmName() string {
	return string(h.algorithm)
}

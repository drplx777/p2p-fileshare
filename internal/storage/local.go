package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
)

type SaveResult struct {
	SizeBytes int64
	SHA256Hex string
	CID       string
	LocalPath string
}

// SaveStream writes data from r into dstDir using tmp file, computes sha256 and returns CIDv1 (raw) based on sha256.
func SaveStream(dstDir string, filename string, r io.Reader) (SaveResult, error) {
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return SaveResult{}, fmt.Errorf("mkdir data dir: %w", err)
	}

	tmp, err := os.CreateTemp(dstDir, "upload-*.tmp")
	if err != nil {
		return SaveResult{}, fmt.Errorf("create temp file: %w", err)
	}

	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(tmp, h), r)
	if err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return SaveResult{}, fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return SaveResult{}, fmt.Errorf("close temp file: %w", err)
	}

	sum := h.Sum(nil) // 32 bytes
	shaHex := hex.EncodeToString(sum)

	mh, err := multihash.Encode(sum, multihash.SHA2_256)
	if err != nil {
		_ = os.Remove(tmp.Name())
		return SaveResult{}, fmt.Errorf("multihash: %w", err)
	}
	c := cid.NewCidV1(cid.Raw, mh).String()

	finalName := shaHex
	if filename != "" {
		ext := filepath.Ext(filename)
		if ext != "" && len(ext) <= 16 {
			finalName = finalName + ext
		}
	}
	finalPath := filepath.Join(dstDir, finalName)
	if _, err := os.Stat(finalPath); err == nil {
		// content already exists; keep existing file
		_ = os.Remove(tmp.Name())
		return SaveResult{
			SizeBytes: n,
			SHA256Hex: shaHex,
			CID:       c,
			LocalPath: finalPath,
		}, nil
	}
	if err := os.Rename(tmp.Name(), finalPath); err != nil {
		_ = os.Remove(tmp.Name())
		return SaveResult{}, fmt.Errorf("move into place: %w", err)
	}

	return SaveResult{
		SizeBytes: n,
		SHA256Hex: shaHex,
		CID:       c,
		LocalPath: finalPath,
	}, nil
}


package storage

import (
	"strings"
	"testing"
)

func TestSaveStream_DeterministicCID(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	res1, err := SaveStream(dir, "a.txt", strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("SaveStream #1: %v", err)
	}
	res2, err := SaveStream(dir, "b.txt", strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("SaveStream #2: %v", err)
	}

	if res1.SHA256Hex != res2.SHA256Hex {
		t.Fatalf("sha mismatch: %s vs %s", res1.SHA256Hex, res2.SHA256Hex)
	}
	if res1.CID != res2.CID {
		t.Fatalf("cid mismatch: %s vs %s", res1.CID, res2.CID)
	}
	if res1.SizeBytes != 5 || res2.SizeBytes != 5 {
		t.Fatalf("unexpected sizes: %d %d", res1.SizeBytes, res2.SizeBytes)
	}
}

func TestSaveStream_WhenFileAlreadyExists(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	res1, err := SaveStream(dir, "a.txt", strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("SaveStream #1: %v", err)
	}
	res2, err := SaveStream(dir, "a.txt", strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("SaveStream #2: %v", err)
	}
	if res1.LocalPath != res2.LocalPath {
		t.Fatalf("expected same local path: %s vs %s", res1.LocalPath, res2.LocalPath)
	}
}


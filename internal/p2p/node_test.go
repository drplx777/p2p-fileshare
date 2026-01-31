package p2p

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

func TestProtocol_FetchRoundtrip(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dir := t.TempDir()
	path := filepath.Join(dir, "file.bin")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	const proto = "/p2p-fileshare/1.0.0-test"

	// node A serves file by CID string "cid123" (we don't validate in test beyond parse, so use a real CID)
	realCID := "bafkreibm6jg3ux5qumhcn2b3flc3tyu6dmlb4xa7u5bf44yegnrjhcwq4e" // CIDv1 raw sha256("hello")
	a, err := NewNode(ctx, []string{"/ip4/127.0.0.1/tcp/0"}, nil, false, proto, func(ctx context.Context, cid string) (string, error) {
		if cid != realCID {
			return "", os.ErrNotExist
		}
		return path, nil
	})
	if err != nil {
		t.Fatalf("new node A: %v", err)
	}
	defer func() { _ = a.Close() }()

	b, err := NewNode(ctx, []string{"/ip4/127.0.0.1/tcp/0"}, nil, false, proto, nil)
	if err != nil {
		t.Fatalf("new node B: %v", err)
	}
	defer func() { _ = b.Close() }()

	// Connect B -> A using addrinfo.
	ai := peer.AddrInfo{ID: a.Host.ID(), Addrs: a.Host.Addrs()}
	r, size, err := b.Fetch(ctx, ai, realCID)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	defer func() { _ = r.Close() }()

	if size != int64(len("hello")) {
		t.Fatalf("unexpected size: %d", size)
	}
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("unexpected body: %q", strings.TrimSpace(string(got)))
	}
}


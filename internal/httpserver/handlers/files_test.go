package handlers

import (
	"context"
	"strings"
	"testing"

	"github.com/drplx/p2p-fileshare/internal/repo"
)

type memRepo struct {
	created []repo.File
}

func (m *memRepo) CreateFile(ctx context.Context, f repo.File) (repo.File, error) {
	m.created = append(m.created, f)
	return f, nil
}
func (m *memRepo) GetFileByID(ctx context.Context, id string) (repo.File, error) {
	for _, f := range m.created {
		if f.ID == id {
			return f, nil
		}
	}
	return repo.File{}, repo.ErrNotFound
}
func (m *memRepo) GetFileByCID(ctx context.Context, cid string) (repo.File, error) {
	for _, f := range m.created {
		if f.CID == cid {
			return f, nil
		}
	}
	return repo.File{}, repo.ErrNotFound
}
func (m *memRepo) ListFiles(ctx context.Context, userID string, limit int) ([]repo.File, error) {
	var out []repo.File
	for _, f := range m.created {
		if userID != "" && f.UserID != userID {
			continue
		}
		out = append(out, f)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func TestFilesHandler_CreateFromStream(t *testing.T) {
	t.Parallel()

	repoMem := &memRepo{}
	h := &FilesHandler{Repo: repoMem, DataDir: t.TempDir()}

	f1, err := h.CreateFromStream(context.Background(), "user1", "x.txt", strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("CreateFromStream: %v", err)
	}
	f2, err := h.CreateFromStream(context.Background(), "user1", "y.txt", strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("CreateFromStream: %v", err)
	}

	if f1.CID != f2.CID {
		t.Fatalf("expected same CID for same content: %s vs %s", f1.CID, f2.CID)
	}
	if f1.SHA256Hex != f2.SHA256Hex {
		t.Fatalf("expected same sha for same content: %s vs %s", f1.SHA256Hex, f2.SHA256Hex)
	}
}

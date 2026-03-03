package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/drplx/p2p-fileshare/internal/httpserver/middleware"
	"github.com/drplx/p2p-fileshare/internal/repo"
	"github.com/gofiber/fiber/v3"
)

// memFilesRepoForShares: returns files by ID for share tests (we seed one file per test).
type memFilesRepoForShares struct {
	files map[string]repo.File
	mu    sync.RWMutex
}

func newMemFilesRepoForShares() *memFilesRepoForShares {
	return &memFilesRepoForShares{files: make(map[string]repo.File)}
}

func (m *memFilesRepoForShares) add(f repo.File) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.files[f.ID] = f
}

func (m *memFilesRepoForShares) CreateFile(ctx context.Context, f repo.File) (repo.File, error) {
	return repo.File{}, nil
}
func (m *memFilesRepoForShares) GetFileByID(ctx context.Context, id string) (repo.File, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if f, ok := m.files[id]; ok {
		return f, nil
	}
	return repo.File{}, repo.ErrNotFound
}
func (m *memFilesRepoForShares) GetFileByCID(ctx context.Context, cid string) (repo.File, error) {
	return repo.File{}, repo.ErrNotFound
}
func (m *memFilesRepoForShares) ListFiles(ctx context.Context, userID string, limit int) ([]repo.File, error) {
	return nil, nil
}

type memSharesRepo struct {
	byFileID map[string]repo.FileShare
	byToken  map[string]repo.FileShare
	mu       sync.RWMutex
}

func newMemSharesRepo() *memSharesRepo {
	return &memSharesRepo{
		byFileID: make(map[string]repo.FileShare),
		byToken:  make(map[string]repo.FileShare),
	}
}

func (m *memSharesRepo) CreateOrUpdateShare(ctx context.Context, fileID, token string, expiresAt *time.Time) (repo.FileShare, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := repo.FileShare{FileID: fileID, Token: token, CreatedAt: time.Now(), ExpiresAt: expiresAt}
	m.byFileID[fileID] = s
	m.byToken[token] = s
	return s, nil
}

func (m *memSharesRepo) GetShareByToken(ctx context.Context, token string) (repo.FileShare, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s, ok := m.byToken[token]; ok {
		return s, nil
	}
	return repo.FileShare{}, repo.ErrNotFound
}

func (m *memSharesRepo) GetShareByFileID(ctx context.Context, fileID string) (repo.FileShare, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s, ok := m.byFileID[fileID]; ok {
		return s, nil
	}
	return repo.FileShare{}, repo.ErrNotFound
}

func (m *memSharesRepo) DeleteShare(ctx context.Context, fileID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.byFileID[fileID]; ok {
		delete(m.byToken, s.Token)
		delete(m.byFileID, fileID)
	}
	return nil
}

func TestSharesHandler_CreateOrGetShare_Success(t *testing.T) {
	t.Parallel()
	users := newMemUsersRepo()
	files := newMemFilesRepoForShares()
	shares := newMemSharesRepo()

	jwtSecret := []byte("test-secret-key")
	authHandler := &AuthHandler{Users: users, JWT: struct{ Secret []byte }{Secret: jwtSecret}}
	sharesHandler := &SharesHandler{Files: files, Shares: shares}

	app := fiber.New()
	app.Post("/auth/register", authHandler.Register)
	app.Post("/auth/login", authHandler.Login)
	api := app.Group("/api", middleware.RequireAuth(jwtSecret))
	api.Post("/files/:id/share", sharesHandler.CreateOrGetShare)

	regBody := bytes.NewBufferString(`{"email":"share@test.com","password":"password123"}`)
	regReq := httptest.NewRequest(http.MethodPost, "/auth/register", regBody)
	regReq.Header.Set("Content-Type", "application/json")
	regResp, err := app.Test(regReq)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	defer regResp.Body.Close()
	if regResp.StatusCode != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d", regResp.StatusCode)
	}
	var regOut struct {
		Token string `json:"token"`
		User  struct { ID string `json:"id"` } `json:"user"`
	}
	if err := json.NewDecoder(regResp.Body).Decode(&regOut); err != nil {
		t.Fatalf("decode register: %v", err)
	}
	userID := regOut.User.ID
	token := regOut.Token

	files.add(repo.File{
		ID: "file-1", UserID: userID, Name: "doc.pdf", LocalPath: "/tmp/doc.pdf",
		SizeBytes: 100, SHA256Hex: "abc", CID: "cid1", CreatedAt: time.Now(),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/files/file-1/share", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	var out struct {
		Token string `json:"token"`
		URL   string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Token == "" || out.URL == "" {
		t.Errorf("expected token and url, got token=%q url=%q", out.Token, out.URL)
	}
}

func TestSharesHandler_CreateOrGetShare_ReturnsExisting(t *testing.T) {
	t.Parallel()
	users := newMemUsersRepo()
	files := newMemFilesRepoForShares()
	shares := newMemSharesRepo()

	jwtSecret := []byte("test-secret-key")
	authHandler := &AuthHandler{Users: users, JWT: struct{ Secret []byte }{Secret: jwtSecret}}
	sharesHandler := &SharesHandler{Files: files, Shares: shares}

	app := fiber.New()
	app.Post("/auth/register", authHandler.Register)
	api := app.Group("/api", middleware.RequireAuth(jwtSecret))
	api.Post("/files/:id/share", sharesHandler.CreateOrGetShare)

	regBody := bytes.NewBufferString(`{"email":"share2@test.com","password":"password123"}`)
	regReq := httptest.NewRequest(http.MethodPost, "/auth/register", regBody)
	regReq.Header.Set("Content-Type", "application/json")
	regResp, _ := app.Test(regReq)
	defer regResp.Body.Close()
	var regOut struct {
		Token string `json:"token"`
		User  struct { ID string `json:"id"` } `json:"user"`
	}
	_ = json.NewDecoder(regResp.Body).Decode(&regOut)
	userID := regOut.User.ID
	token := regOut.Token

	files.add(repo.File{
		ID: "file-2", UserID: userID, Name: "x.txt", LocalPath: "/tmp/x.txt",
		SizeBytes: 5, SHA256Hex: "x", CID: "c2", CreatedAt: time.Now(),
	})
	_, _ = shares.CreateOrUpdateShare(context.Background(), "file-2", "existing-token-123", nil)

	req := httptest.NewRequest(http.MethodPost, "/api/files/file-2/share", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 (existing share), got %d", resp.StatusCode)
	}
	var out struct {
		Token string `json:"token"`
		URL   string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Token != "existing-token-123" {
		t.Errorf("expected existing-token-123, got %q", out.Token)
	}
}

func TestSharesHandler_CreateOrGetShare_Unauthorized(t *testing.T) {
	t.Parallel()
	files := newMemFilesRepoForShares()
	shares := newMemSharesRepo()
	files.add(repo.File{
		ID: "f1", UserID: "user1", Name: "a", LocalPath: "/a",
		SizeBytes: 1, SHA256Hex: "a", CID: "c", CreatedAt: time.Now(),
	})

	app := fiber.New()
	app.Post("/files/:id/share", (&SharesHandler{Files: files, Shares: shares}).CreateOrGetShare)

	req := httptest.NewRequest(http.MethodPost, "/files/f1/share", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestSharesHandler_GetShareInfo_Success(t *testing.T) {
	t.Parallel()
	files := newMemFilesRepoForShares()
	shares := newMemSharesRepo()

	files.add(repo.File{
		ID: "f-info", UserID: "u1", Name: "report.pdf", LocalPath: "/data/report.pdf",
		SizeBytes: 1024, SHA256Hex: "sha", CID: "cid", CreatedAt: time.Now(),
	})
	_, _ = shares.CreateOrUpdateShare(context.Background(), "f-info", "token-abc", nil)

	app := fiber.New()
	app.Get("/shares/:token", (&SharesHandler{Files: files, Shares: shares}).GetShareInfo)

	req := httptest.NewRequest(http.MethodGet, "/shares/token-abc", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var out struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		SizeBytes int64  `json:"sizeBytes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Name != "report.pdf" || out.SizeBytes != 1024 {
		t.Errorf("expected report.pdf 1024, got %q %d", out.Name, out.SizeBytes)
	}
}

func TestSharesHandler_GetShareInfo_NotFound(t *testing.T) {
	t.Parallel()
	files := newMemFilesRepoForShares()
	shares := newMemSharesRepo()

	app := fiber.New()
	app.Get("/shares/:token", (&SharesHandler{Files: files, Shares: shares}).GetShareInfo)

	req := httptest.NewRequest(http.MethodGet, "/shares/invalid-token", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestSharesHandler_RevokeShare_Success(t *testing.T) {
	t.Parallel()
	users := newMemUsersRepo()
	files := newMemFilesRepoForShares()
	shares := newMemSharesRepo()

	jwtSecret := []byte("test-secret-key")
	authHandler := &AuthHandler{Users: users, JWT: struct{ Secret []byte }{Secret: jwtSecret}}
	sharesHandler := &SharesHandler{Files: files, Shares: shares}

	app := fiber.New()
	app.Post("/auth/register", authHandler.Register)
	api := app.Group("/api", middleware.RequireAuth(jwtSecret))
	api.Delete("/files/:id/share", sharesHandler.RevokeShare)

	regBody := bytes.NewBufferString(`{"email":"revoke@test.com","password":"password123"}`)
	regReq := httptest.NewRequest(http.MethodPost, "/auth/register", regBody)
	regReq.Header.Set("Content-Type", "application/json")
	regResp, _ := app.Test(regReq)
	defer regResp.Body.Close()
	var regOut struct {
		Token string `json:"token"`
		User  struct { ID string `json:"id"` } `json:"user"`
	}
	_ = json.NewDecoder(regResp.Body).Decode(&regOut)
	userID := regOut.User.ID
	token := regOut.Token

	files.add(repo.File{
		ID: "file-revoke", UserID: userID, Name: "z", LocalPath: "/z",
		SizeBytes: 1, SHA256Hex: "z", CID: "cz", CreatedAt: time.Now(),
	})
	_, _ = shares.CreateOrUpdateShare(context.Background(), "file-revoke", "revoke-token", nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/files/file-revoke/share", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	_, err = shares.GetShareByToken(context.Background(), "revoke-token")
	if err != repo.ErrNotFound {
		t.Errorf("share should be deleted, got err=%v", err)
	}
}

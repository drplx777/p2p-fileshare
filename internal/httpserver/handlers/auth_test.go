package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/drplx/p2p-fileshare/internal/repo"
	"github.com/gofiber/fiber/v3"
)

type memUsersRepo struct {
	byID    map[string]repo.User
	byEmail map[string]repo.User
}

func newMemUsersRepo() *memUsersRepo {
	return &memUsersRepo{
		byID:    make(map[string]repo.User),
		byEmail: make(map[string]repo.User),
	}
}

func (m *memUsersRepo) CreateUser(ctx context.Context, u repo.User) (repo.User, error) {
	if _, ok := m.byEmail[u.Email]; ok {
		return repo.User{}, repo.ErrDuplicate
	}
	u.CreatedAt = time.Now()
	m.byID[u.ID] = u
	m.byEmail[u.Email] = u
	return u, nil
}

func (m *memUsersRepo) GetUserByID(ctx context.Context, id string) (repo.User, error) {
	if u, ok := m.byID[id]; ok {
		return u, nil
	}
	return repo.User{}, repo.ErrNotFound
}

func (m *memUsersRepo) GetUserByEmail(ctx context.Context, email string) (repo.User, error) {
	if u, ok := m.byEmail[email]; ok {
		return u, nil
	}
	return repo.User{}, repo.ErrNotFound
}

func TestAuthHandler_Register_Success(t *testing.T) {
	t.Parallel()
	users := newMemUsersRepo()
	h := &AuthHandler{
		Users: users,
		JWT:   struct{ Secret []byte }{Secret: []byte("test-secret-key")},
	}
	app := fiber.New()
	app.Post("/register", h.Register)

	body := bytes.NewBufferString(`{"email":"user@test.com","password":"password123"}`)
	req := httptest.NewRequest(http.MethodPost, "/register", body)
	req.Header.Set("Content-Type", "application/json")

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
		User  struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Token == "" {
		t.Error("expected non-empty token")
	}
	if out.User.Email != "user@test.com" {
		t.Errorf("expected user@test.com, got %s", out.User.Email)
	}
	if out.User.ID == "" {
		t.Error("expected non-empty user id")
	}
}

func TestAuthHandler_Register_DuplicateEmail(t *testing.T) {
	t.Parallel()
	users := newMemUsersRepo()
	h := &AuthHandler{
		Users: users,
		JWT:   struct{ Secret []byte }{Secret: []byte("test-secret-key")},
	}
	app := fiber.New()
	app.Post("/register", h.Register)

	bodyStr := `{"email":"dup@test.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(bodyStr))
	req.Header.Set("Content-Type", "application/json")
	_, _ = app.Test(req)

	req2 := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(bodyStr))
	req2.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req2)
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
}

func TestAuthHandler_Register_Validation(t *testing.T) {
	t.Parallel()
	users := newMemUsersRepo()
	h := &AuthHandler{
		Users: users,
		JWT:   struct{ Secret []byte }{Secret: []byte("test-secret-key")},
	}
	app := fiber.New()
	app.Post("/register", h.Register)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"empty email", `{"email":"","password":"password123"}`, http.StatusBadRequest},
		{"short password", `{"email":"a@b.com","password":"short"}`, http.StatusBadRequest},
		{"invalid JSON", `{email}`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Test: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tt.wantStatus {
				t.Errorf("expected %d, got %d", tt.wantStatus, resp.StatusCode)
			}
		})
	}
}

func TestAuthHandler_Login_Success(t *testing.T) {
	t.Parallel()
	users := newMemUsersRepo()
	h := &AuthHandler{
		Users: users,
		JWT:   struct{ Secret []byte }{Secret: []byte("test-secret-key")},
	}
	app := fiber.New()
	app.Post("/register", h.Register)
	app.Post("/login", h.Login)

	regBody := bytes.NewBufferString(`{"email":"login@test.com","password":"password123"}`)
	regReq := httptest.NewRequest(http.MethodPost, "/register", regBody)
	regReq.Header.Set("Content-Type", "application/json")
	regResp, err := app.Test(regReq)
	if err != nil || regResp.StatusCode != http.StatusCreated {
		t.Fatalf("register failed: %v", err)
	}
	regResp.Body.Close()

	loginBody := bytes.NewBufferString(`{"email":"login@test.com","password":"password123"}`)
	loginReq := httptest.NewRequest(http.MethodPost, "/login", loginBody)
	loginReq.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(loginReq)
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var out struct {
		Token string `json:"token"`
		User  struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Token == "" || out.User.Email != "login@test.com" {
		t.Errorf("expected token and email, got token=%q email=%s", out.Token, out.User.Email)
	}
}

func TestAuthHandler_Login_WrongPassword(t *testing.T) {
	t.Parallel()
	users := newMemUsersRepo()
	h := &AuthHandler{
		Users: users,
		JWT:   struct{ Secret []byte }{Secret: []byte("test-secret-key")},
	}
	app := fiber.New()
	app.Post("/register", h.Register)
	app.Post("/login", h.Login)

	regBody := bytes.NewBufferString(`{"email":"wrong@test.com","password":"password123"}`)
	regReq := httptest.NewRequest(http.MethodPost, "/register", regBody)
	regReq.Header.Set("Content-Type", "application/json")
	_, _ = app.Test(regReq)

	loginBody := bytes.NewBufferString(`{"email":"wrong@test.com","password":"wrongpassword"}`)
	loginReq := httptest.NewRequest(http.MethodPost, "/login", loginBody)
	loginReq.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(loginReq)
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestAuthHandler_Login_UnknownUser(t *testing.T) {
	t.Parallel()
	users := newMemUsersRepo()
	h := &AuthHandler{
		Users: users,
		JWT:   struct{ Secret []byte }{Secret: []byte("test-secret-key")},
	}
	app := fiber.New()
	app.Post("/login", h.Login)

	body := bytes.NewBufferString(`{"email":"nobody@test.com","password":"anypassword"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

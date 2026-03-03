package handlers

import (
	"net/http"
	"strings"

	"github.com/drplx/p2p-fileshare/internal/auth"
	"github.com/drplx/p2p-fileshare/internal/repo"
	"github.com/gofiber/fiber/v3"
	"github.com/oklog/ulid/v2"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	Users repo.UserRepository
	JWT   struct {
		Secret []byte
	}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string `json:"token"`
	User  struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
}

func (h *AuthHandler) Register(c fiber.Ctx) error {
	var req registerRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON"})
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "email required"})
	}
	if len(req.Password) < 8 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "password must be at least 8 characters"})
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to hash password"})
	}
	u := repo.User{
		ID:           ulid.Make().String(),
		Email:        email,
		PasswordHash: string(hash),
	}
	created, err := h.Users.CreateUser(c.Context(), u)
	if err != nil {
		if err == repo.ErrDuplicate {
			return c.Status(http.StatusConflict).JSON(fiber.Map{"error": "email already registered"})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	token, err := auth.NewToken(h.JWT.Secret, created.ID, auth.DefaultExpire)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create token"})
	}
	return c.Status(http.StatusCreated).JSON(authResponse{
		Token: token,
		User: struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		}{ID: created.ID, Email: created.Email},
	})
}

func (h *AuthHandler) Login(c fiber.Ctx) error {
	var req loginRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON"})
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "email required"})
	}
	u, err := h.Users.GetUserByEmail(c.Context(), email)
	if err != nil {
		if err == repo.ErrNotFound {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "invalid email or password"})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "invalid email or password"})
	}
	token, err := auth.NewToken(h.JWT.Secret, u.ID, auth.DefaultExpire)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create token"})
	}
	return c.JSON(authResponse{
		Token: token,
		User: struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		}{ID: u.ID, Email: u.Email},
	})
}

package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"

	"github.com/drplx/p2p-fileshare/internal/httpserver/middleware"
	"github.com/drplx/p2p-fileshare/internal/repo"
	"github.com/gofiber/fiber/v3"
)

const shareTokenBytes = 32

type SharesHandler struct {
	Files repo.FileRepository
	Shares repo.FileShareRepository
}

// CreateOrGetShare creates a share link for the file (owner only). Returns existing share if already shared.
func (h *SharesHandler) CreateOrGetShare(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "id required"})
	}
	f, err := h.Files.GetFileByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "not found"})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if f.UserID != userID {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{"error": "not your file"})
	}
	existing, err := h.Shares.GetShareByFileID(c.Context(), id)
	if err == nil {
		return c.JSON(fiber.Map{
			"token": existing.Token,
			"url":   "/api/v1/shares/" + existing.Token + "/download",
		})
	}
	token := mustGenerateToken()
	share, err := h.Shares.CreateOrUpdateShare(c.Context(), id, token, nil)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"token": share.Token,
		"url":   "/api/v1/shares/" + share.Token + "/download",
	})
}

// RevokeShare removes the share link (owner only).
func (h *SharesHandler) RevokeShare(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "id required"})
	}
	f, err := h.Files.GetFileByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "not found"})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if f.UserID != userID {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{"error": "not your file"})
	}
	if err := h.Shares.DeleteShare(c.Context(), id); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(http.StatusNoContent).Send(nil)
}

// GetShareInfo returns file info by share token (no auth). For UI to show filename/size before download.
func (h *SharesHandler) GetShareInfo(c fiber.Ctx) error {
	token := c.Params("token")
	if token == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "token required"})
	}
	share, err := h.Shares.GetShareByToken(c.Context(), token)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "link not found or expired"})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	f, err := h.Files.GetFileByID(c.Context(), share.FileID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "file not found"})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"id":         f.ID,
		"name":       f.Name,
		"sizeBytes":  f.SizeBytes,
		"createdAt":  f.CreatedAt,
	})
}

// DownloadByToken allows downloading file by share token (no auth).
func (h *SharesHandler) DownloadByToken(c fiber.Ctx) error {
	token := c.Params("token")
	if token == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "token required"})
	}
	share, err := h.Shares.GetShareByToken(c.Context(), token)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "link not found or expired"})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	f, err := h.Files.GetFileByID(c.Context(), share.FileID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "file not found"})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	c.Set("Content-Disposition", "attachment; filename=\""+f.Name+"\"")
	return c.SendFile(f.LocalPath)
}

func mustGenerateToken() string {
	b := make([]byte, shareTokenBytes)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

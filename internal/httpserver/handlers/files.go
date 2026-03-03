package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/drplx/p2p-fileshare/internal/httpserver/middleware"
	"github.com/drplx/p2p-fileshare/internal/repo"
	"github.com/drplx/p2p-fileshare/internal/storage"
	"github.com/gofiber/fiber/v3"
	"github.com/oklog/ulid/v2"
)

type FilesHandler struct {
	Repo    repo.FileRepository
	DataDir string
	P2P     interface {
		Provide(ctx context.Context, cid string) error
	}
}

type fileDTO struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	SizeBytes int64     `json:"sizeBytes"`
	SHA256Hex string    `json:"sha256Hex"`
	CID       string    `json:"cid"`
	CreatedAt time.Time `json:"createdAt"`
}

func toDTO(f repo.File) fileDTO {
	return fileDTO{
		ID:        f.ID,
		Name:      f.Name,
		SizeBytes: f.SizeBytes,
		SHA256Hex: f.SHA256Hex,
		CID:       f.CID,
		CreatedAt: f.CreatedAt,
	}
}

func (h *FilesHandler) Upload(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	fh, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(http.StatusBadRequest, "missing multipart field 'file'")
	}
	f, err := fh.Open()
	if err != nil {
		return fiber.NewError(http.StatusBadRequest, "failed to open upload")
	}
	defer func() { _ = f.Close() }()

	saved, err := storage.SaveStream(h.DataDir, fh.Filename, f)
	if err != nil {
		return fiber.NewError(http.StatusInternalServerError, err.Error())
	}

	newFile := repo.File{
		ID:        ulid.Make().String(),
		UserID:    userID,
		Name:      fh.Filename,
		SizeBytes: saved.SizeBytes,
		SHA256Hex: saved.SHA256Hex,
		CID:       saved.CID,
		LocalPath: saved.LocalPath,
	}

	created, err := h.Repo.CreateFile(c.Context(), newFile)
	if err != nil {
		return fiber.NewError(http.StatusInternalServerError, err.Error())
	}
	if h.P2P != nil {
		_ = h.P2P.Provide(c.Context(), created.CID)
	}

	return c.Status(http.StatusCreated).JSON(toDTO(created))
}

func (h *FilesHandler) List(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	files, err := h.Repo.ListFiles(c.Context(), userID, 100)
	if err != nil {
		return fiber.NewError(http.StatusInternalServerError, err.Error())
	}
	out := make([]fileDTO, 0, len(files))
	for _, f := range files {
		out = append(out, toDTO(f))
	}
	return c.JSON(out)
}

func (h *FilesHandler) Get(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	id := c.Params("id")
	if id == "" {
		return fiber.NewError(http.StatusBadRequest, "id required")
	}
	f, err := h.Repo.GetFileByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return fiber.NewError(http.StatusNotFound, "not found")
		}
		return fiber.NewError(http.StatusInternalServerError, err.Error())
	}
	if f.UserID != userID {
		return fiber.NewError(http.StatusNotFound, "not found")
	}
	return c.JSON(toDTO(f))
}

func (h *FilesHandler) Download(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	id := c.Params("id")
	if id == "" {
		return fiber.NewError(http.StatusBadRequest, "id required")
	}
	f, err := h.Repo.GetFileByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return fiber.NewError(http.StatusNotFound, "not found")
		}
		return fiber.NewError(http.StatusInternalServerError, err.Error())
	}
	if f.UserID != userID {
		return fiber.NewError(http.StatusNotFound, "not found")
	}
	return c.SendFile(f.LocalPath)
}

func (h *FilesHandler) CreateFromStream(ctx context.Context, userID, filename string, r interface{ Read([]byte) (int, error) }) (repo.File, error) {
	saved, err := storage.SaveStream(h.DataDir, filename, r)
	if err != nil {
		return repo.File{}, fmt.Errorf("save: %w", err)
	}
	f := repo.File{
		ID:        ulid.Make().String(),
		UserID:    userID,
		Name:      filename,
		SizeBytes: saved.SizeBytes,
		SHA256Hex: saved.SHA256Hex,
		CID:       saved.CID,
		LocalPath: saved.LocalPath,
	}
	created, err := h.Repo.CreateFile(ctx, f)
	if err != nil {
		return repo.File{}, err
	}
	return created, nil
}

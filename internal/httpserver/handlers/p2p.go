package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/drplx/p2p-fileshare/internal/httpserver/middleware"
	"github.com/drplx/p2p-fileshare/internal/repo"
	"github.com/drplx/p2p-fileshare/internal/storage"
	"github.com/gofiber/fiber/v3"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/oklog/ulid/v2"
)

type P2PNode interface {
	PeerID() string
	Addrs() []string
	Provide(ctx context.Context, cid string) error
	FindProviders(ctx context.Context, cid string, limit int) ([]peer.AddrInfo, error)
	Fetch(ctx context.Context, from peer.AddrInfo, cid string) (r io.ReadCloser, size int64, err error)
}

type P2PHandler struct {
	Node    P2PNode
	Repo    repo.FileRepository
	DataDir string
}

func (h *P2PHandler) Me(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"peerId": h.Node.PeerID(),
		"addrs":  h.Node.Addrs(),
	})
}

func (h *P2PHandler) Search(c fiber.Ctx) error {
	cidStr := c.Query("cid")
	if cidStr == "" {
		return fiber.NewError(http.StatusBadRequest, "cid required")
	}
	limit := 10
	if s := c.Query("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			limit = n
		}
	}
	providers, err := h.Node.FindProviders(c.Context(), cidStr, limit)
	if err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}

	type providerDTO struct {
		PeerID string   `json:"peerId"`
		Addrs  []string `json:"addrs"`
	}
	out := make([]providerDTO, 0, len(providers))
	for _, p := range providers {
		addrs := make([]string, 0, len(p.Addrs))
		for _, a := range p.Addrs {
			addrs = append(addrs, a.String())
		}
		out = append(out, providerDTO{PeerID: p.ID.String(), Addrs: addrs})
	}
	return c.JSON(out)
}

func (h *P2PHandler) Fetch(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	cidStr := c.Query("cid")
	if cidStr == "" {
		return fiber.NewError(http.StatusBadRequest, "cid required")
	}

	if existing, err := h.Repo.GetFileByCID(c.Context(), cidStr); err == nil {
		return c.JSON(toDTO(existing))
	}

	providers, err := h.Node.FindProviders(c.Context(), cidStr, 20)
	if err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}
	if len(providers) == 0 {
		return fiber.NewError(http.StatusNotFound, "no providers found")
	}

	var lastErr error
	for _, p := range providers {
		if p.ID.String() == h.Node.PeerID() {
			continue
		}
		r, _, err := h.Node.Fetch(c.Context(), p, cidStr)
		if err != nil {
			lastErr = err
			continue
		}
		saved, err := storage.SaveStream(h.DataDir, "", r)
		_ = r.Close()
		if err != nil {
			lastErr = err
			continue
		}
		if saved.CID != cidStr {
			lastErr = errors.New("cid mismatch after download")
			continue
		}

		f := repo.File{
			ID:        ulid.Make().String(),
			UserID:    userID,
			Name:      cidStr,
			SizeBytes: saved.SizeBytes,
			SHA256Hex: saved.SHA256Hex,
			CID:       saved.CID,
			LocalPath: saved.LocalPath,
		}
		created, err := h.Repo.CreateFile(c.Context(), f)
		if err != nil {
			lastErr = err
			continue
		}

		_ = h.Node.Provide(c.Context(), created.CID)
		return c.JSON(toDTO(created))
	}

	if lastErr != nil {
		return fiber.NewError(http.StatusBadGateway, lastErr.Error())
	}
	return fiber.NewError(http.StatusBadGateway, "failed to fetch from providers")
}

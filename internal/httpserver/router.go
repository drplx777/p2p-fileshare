package httpserver

import (
	"github.com/drplx/p2p-fileshare/internal/httpserver/handlers"
	"github.com/gofiber/fiber/v3"
)

type Deps struct {
	Files *handlers.FilesHandler
	P2P   *handlers.P2PHandler
}

func NewApp(deps Deps) *fiber.App {
	app := fiber.New()

	v1 := app.Group("/api/v1")
	v1.Get("/health", handlers.Health)

	v1.Post("/files", deps.Files.Upload)
	v1.Get("/files", deps.Files.List)
	v1.Get("/files/:id", deps.Files.Get)
	v1.Get("/files/:id/download", deps.Files.Download)

	v1.Get("/p2p/me", deps.P2P.Me)
	v1.Get("/p2p/search", deps.P2P.Search)
	v1.Get("/p2p/fetch", deps.P2P.Fetch)

	return app
}


package httpserver

import (
	"github.com/drplx/p2p-fileshare/internal/httpserver/handlers"
	"github.com/drplx/p2p-fileshare/internal/httpserver/middleware"
	"github.com/gofiber/fiber/v3"
)

type Deps struct {
	Auth   *handlers.AuthHandler
	Files  *handlers.FilesHandler
	P2P    *handlers.P2PHandler
	Shares *handlers.SharesHandler
	JWTSecret []byte
}

func NewApp(deps Deps) *fiber.App {
	app := fiber.New()

	v1 := app.Group("/api/v1")
	v1.Get("/health", handlers.Health)

	// Public auth
	v1.Post("/auth/register", deps.Auth.Register)
	v1.Post("/auth/login", deps.Auth.Login)

	// Public share links (no auth)
	v1.Get("/shares/:token", deps.Shares.GetShareInfo)
	v1.Get("/shares/:token/download", deps.Shares.DownloadByToken)

	// Protected routes (require Bearer token)
	protected := v1.Group("", middleware.RequireAuth(deps.JWTSecret))
	protected.Post("/files", deps.Files.Upload)
	protected.Get("/files", deps.Files.List)
	protected.Get("/files/:id", deps.Files.Get)
	protected.Get("/files/:id/download", deps.Files.Download)
	protected.Post("/files/:id/share", deps.Shares.CreateOrGetShare)
	protected.Delete("/files/:id/share", deps.Shares.RevokeShare)

	protected.Get("/p2p/me", deps.P2P.Me)
	protected.Get("/p2p/search", deps.P2P.Search)
	protected.Get("/p2p/fetch", deps.P2P.Fetch)

	return app
}


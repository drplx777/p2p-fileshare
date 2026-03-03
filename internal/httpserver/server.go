package httpserver

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
)

type Server struct {
	app *fiber.App
}

func New(app *fiber.App) *Server {
	return &Server{app: app}
}

func (s *Server) Listen(addr string) error {
	return s.app.Listen(addr)
}

func (s *Server) Shutdown(ctx context.Context) error {
	done := make(chan error, 1)
	go func() { done <- s.app.Shutdown() }()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		select {
		case err := <-done:
			return err
		case <-time.After(250 * time.Millisecond):
			return ctx.Err()
		}
	}
}

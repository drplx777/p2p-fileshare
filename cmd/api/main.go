package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/drplx/p2p-fileshare/internal/config"
	"github.com/drplx/p2p-fileshare/internal/db"
	"github.com/drplx/p2p-fileshare/internal/httpserver"
	"github.com/drplx/p2p-fileshare/internal/httpserver/handlers"
	"github.com/drplx/p2p-fileshare/internal/p2p"
	"github.com/drplx/p2p-fileshare/internal/repo/postgres"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	usersRepo := postgres.NewUsersRepo(pool)
	filesRepo := postgres.NewFilesRepo(pool)
	sharesRepo := postgres.NewFileSharesRepo(pool)

	node, err := p2p.NewNode(ctx, cfg.P2PListenAddrs, cfg.P2PBootstrapPeers, cfg.P2PEnableMDNS, cfg.P2PProtocolID, func(ctx context.Context, cid string) (string, error) {
		f, err := filesRepo.GetFileByCID(ctx, cid)
		if err != nil {
			return "", err
		}
		return f.LocalPath, nil
	})
	if err != nil {
		log.Fatalf("p2p: %v", err)
	}
	defer func() { _ = node.Close() }()

	authHandler := &handlers.AuthHandler{
		Users: usersRepo,
		JWT:   struct{ Secret []byte }{Secret: cfg.JWTSecret},
	}
	filesHandler := &handlers.FilesHandler{
		Repo:    filesRepo,
		DataDir: cfg.DataDir,
		P2P:     node,
	}
	p2pHandler := &handlers.P2PHandler{
		Node:    node,
		Repo:    filesRepo,
		DataDir: cfg.DataDir,
	}
	sharesHandler := &handlers.SharesHandler{
		Files:  filesRepo,
		Shares: sharesRepo,
	}

	app := httpserver.NewApp(httpserver.Deps{
		Auth:       authHandler,
		Files:      filesHandler,
		P2P:        p2pHandler,
		Shares:     sharesHandler,
		JWTSecret:  cfg.JWTSecret,
	})
	srv := httpserver.New(app)

	go func() {
		log.Printf("http listening on %s", cfg.HTTPAddr)
		log.Printf("p2p peer id: %s", node.PeerID())
		if err := srv.Listen(cfg.HTTPAddr); err != nil {
			log.Printf("http stopped: %v", err)
			stop()
		}
	}()

	<-ctx.Done()
	log.Printf("shutting down...")
	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctxShutdown)
}


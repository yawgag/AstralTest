package app

import (
	"AstralTest/config"
	"AstralTest/internal/service"
	"AstralTest/internal/storage"
	"AstralTest/internal/storage/cache"
	"AstralTest/internal/storage/postgres"
	"AstralTest/internal/transport"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type App struct {
	Config *config.Config
	Router *http.ServeMux
}

func InitApp() (*App, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("load config error: %w", err)
	}

	dbConn, err := postgres.InitDb(cfg.DbURL)
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	userStorage := storage.NewUserStorage(dbConn)
	sessionStorage := storage.NewSessionStorage(dbConn)
	authService := service.NewAuthService(userStorage, sessionStorage, cfg.AdminToken)

	fileStorage, err := storage.NewFileStorage(dbConn, cfg.LocalFileStoragePath)
	if err != nil {
		return nil, fmt.Errorf("can't init local storage")
	}
	cacheStorage := cache.NewStructuredCache()
	wcsService := service.NewWcsService(sessionStorage, fileStorage, cfg.LocalFileStoragePath, cacheStorage)

	handler := transport.NewHandler(authService, wcsService)

	return &App{
		Config: cfg,
		Router: handler.InitRouter(),
	}, nil
}

func (a *App) Run() {
	fmt.Println("Run server")

	srv := &http.Server{
		Addr:    a.Config.ServerAddres,
		Handler: a.Router,
	}

	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("server shutdown: %v", err)
		}
		close(idleConnsClosed)
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("server ListenAndServe error: %v", err)
	}

	<-idleConnsClosed
}

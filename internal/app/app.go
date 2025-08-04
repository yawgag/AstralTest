package app

import (
	"AstralTest/config"
	"AstralTest/internal/service"
	"AstralTest/internal/storage"
	"AstralTest/internal/storage/cache"
	"AstralTest/internal/storage/postgres"
	"AstralTest/internal/transport"
	"fmt"
	"log"
	"net/http"
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
	if err := http.ListenAndServe(a.Config.ServerAddres, a.Router); err != nil {
		log.Fatal(err)
	}
}

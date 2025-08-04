package config

import (
	"fmt"
	"os"

	"github.com/google/uuid"
)

type Config struct {
	ServerAddres         string
	DbURL                string
	AdminToken           uuid.UUID
	LocalFileStoragePath string
}

func LoadConfig() (*Config, error) {

	uuidAdminToken, err := uuid.Parse(os.Getenv("ADMIN_TOKEN"))
	if err != nil {
		return nil, fmt.Errorf("bad admin token")
	}
	config := &Config{
		ServerAddres:         os.Getenv("SERVER_ADDR"),
		DbURL:                os.Getenv("DB_URL"),
		LocalFileStoragePath: os.Getenv("LOCAL_STORAGE_PATH"),
		AdminToken:           uuidAdminToken,
	}

	if config.ServerAddres == "" || config.DbURL == "" || config.LocalFileStoragePath == "" {
		return nil, fmt.Errorf("not enough data in config")
	}
	return config, nil
}

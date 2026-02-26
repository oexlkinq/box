package main

import (
	"fmt"
	"log/slog"

	"github.com/caarlos0/env/v11"
	"github.com/gin-gonic/gin"
	"github.com/oexlkinq/box/routes"
	"github.com/oexlkinq/box/storage"
)

func setup(cfg *Config) error {
	r := gin.Default()
	r.SetTrustedProxies(nil)

	s, err := storage.New(cfg.StoragePath, cfg.DBFilePath, cfg.MaxStorageSize)
	if err != nil {
		return fmt.Errorf("create storage: %w", err)
	}

	routes.SetupRoutes(r, s, cfg.BasePath)

	return r.Run(cfg.ListenAddr)
}

type Config struct {
	StoragePath    string `env:"STORAGE_PATH,required"`
	DBFilePath     string `env:"DB_FILE_PATH,required"`
	MaxStorageSize int64  `env:"MAX_STORAGE_SIZE,required"`
	BasePath       string `env:"BASE_PATH,required"`
	ListenAddr     string `env:"LISTEN_ADDR,required"`
}

func main() {
	cfg := Config{}
	err := env.Parse(&cfg)
	if err != nil {
		slog.Error(err.Error())
		return
	}

	slog.Error(setup(&cfg).Error())
}

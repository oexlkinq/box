package main

import (
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/oexlkinq/box/routes"
	"github.com/oexlkinq/box/storage"
)

func setup(storagePath string, dbFilePath string, maxStorageSize int64) error {
	r := gin.Default()

	s, err := storage.New(storagePath, dbFilePath, maxStorageSize)
	if err != nil {
		return fmt.Errorf("create storage: %w", err)
	}

	routes.SetupRoutes(r, s)

	return r.Run("[::]:2080")
}

func main() {
	slog.Error(setup("./", "./files.db", 1<<30).Error())
}

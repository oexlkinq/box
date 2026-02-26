package routes

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/oexlkinq/box/storage"
)

type FileUrl struct {
	FolderUrl
	File string `uri:"file" binding:"required,max=256"`
}

func MakeGetFile(s *storage.Storage) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		fileUrl := FileUrl{}

		err := ctx.ShouldBindUri(&fileUrl)
		if err != nil {
			ctx.AbortWithError(http.StatusBadRequest, err)
			return
		}

		file, err := s.Get(filepath.Join(fileUrl.Folder, fileUrl.File))
		if err != nil {
			if errors.Is(err, storage.ErrFileNotExists) {
				ctx.AbortWithError(http.StatusNotFound, err)
				return
			}

			ctx.AbortWithError(http.StatusInternalServerError, fmt.Errorf("get file from storage: %w", err))
			return
		}
		defer file.Close()

		_, err = io.Copy(ctx.Writer, file)
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, fmt.Errorf("write file to response: %w", err))
			return
		}
	}
}

func MakeDeleteFile(s *storage.Storage) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		fileUrl := FileUrl{}

		err := ctx.ShouldBindUri(&fileUrl)
		if err != nil {
			ctx.AbortWithError(http.StatusBadRequest, err)
			return
		}

		err = s.Delete(filepath.Join(fileUrl.Folder, fileUrl.File))
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, fmt.Errorf("delete file from storage: %w", err))
			return
		}

		ctx.String(http.StatusOK, "done")
	}
}

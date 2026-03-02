package routes

import (
	"errors"
	"fmt"
	"mime"
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

		file, size, err := s.Get(ctx, filepath.Join(fileUrl.Folder, fileUrl.File))
		if err != nil {
			if errors.Is(err, storage.ErrFileNotExists) {
				ctx.AbortWithError(http.StatusNotFound, err)
				return
			}

			ctx.AbortWithError(http.StatusInternalServerError, fmt.Errorf("get file from storage: %w", err))
			return
		}

		ctx.DataFromReader(
			http.StatusOK,
			size,
			mime.TypeByExtension(filepath.Ext(fileUrl.File)),
			file,
			map[string]string{"Content-Disposition": fmt.Sprintf(`attachment; filename="%s"`, fileUrl.File)},
		)
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

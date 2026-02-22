package routes

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/oexlkinq/box/storage"
)

type FileUrl struct {
	Folder string `uri:"folder" binding:"required,max=256"`
	File   string `uri:"file" binding:"required,max=256"`
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

func MakePostFile(s *storage.Storage) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		fileUrl := FileUrl{}

		err := ctx.ShouldBindUri(&fileUrl)
		if err != nil {
			ctx.AbortWithError(http.StatusBadRequest, err)
			return
		}

		// Read multipart form
		reader, err := ctx.Request.MultipartReader()
		if err != nil {
			ctx.AbortWithError(http.StatusBadRequest, fmt.Errorf("start multipart reader: %w", err))
			return
		}

		var part *multipart.Part
		var filename string
		for {
			part, err = reader.NextPart()
			if err != nil {
				// if form ended and there were no files
				if errors.Is(err, io.EOF) {
					ctx.String(http.StatusBadRequest, "no files in the form")
					return
				}

				ctx.AbortWithError(http.StatusInternalServerError, fmt.Errorf("read next form part: %w", err))
				return
			}
			defer part.Close()

			filename = part.FileName()
			if filename == "" {
				continue
			}

			break
		}

		err = s.Create(part, filepath.Join(fileUrl.Folder, fileUrl.File), ctx.Request.ContentLength)
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, fmt.Errorf("push file to storage: %w", err))
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
	}
}

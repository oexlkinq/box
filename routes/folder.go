package routes

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"

	"github.com/dustin/go-humanize"
	"github.com/gin-gonic/gin"
	"github.com/oexlkinq/box/storage"
)

type FolderUrl struct {
	Folder string `uri:"folder" binding:"required,max=256"`
}

type folderPageParams struct {
	Files           []fileInfo
	AbsFolderUrl    string
	AbsFolderDelUrl string
	ID              string
	FreeSpace       string
	AvailableSpace  string
}

type fileInfo struct {
	AbsFileUrl    string
	AbsFileDelUrl string
	Size          string
	Url           string
}

func MakeGetFolder(s *storage.Storage, basePath string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		folderUrl := FolderUrl{}

		err := ctx.ShouldBindUri(&folderUrl)
		if err != nil {
			ctx.AbortWithError(http.StatusBadRequest, err)
			return
		}

		// достать список файлов в папке из бд
		files, err := s.List(folderUrl.Folder)
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		afps := []fileInfo{}
		for _, file := range files {
			afps = append(afps, fileInfo{
				AbsFileUrl:    filepath.Join(basePath, file.Url),
				AbsFileDelUrl: filepath.Join(basePath, "del", file.Url),
				Size:          humanize.IBytes(uint64(file.Size)),
				Url:           file.Url,
			})
		}

		freeSpace, err := s.GetFreeSpace()
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		ctx.HTML(http.StatusOK, "folder.tmpl", folderPageParams{
			Files:           afps,
			AbsFolderUrl:    filepath.Join(basePath, folderUrl.Folder),
			AbsFolderDelUrl: filepath.Join(basePath, "del", folderUrl.Folder),
			ID:              folderUrl.Folder,
			FreeSpace:       humanize.IBytes(uint64(freeSpace)),
			AvailableSpace:  humanize.IBytes(uint64(s.MaxStorageSize)),
		})
	}
}

func MakePostFolder(s *storage.Storage, basePath string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		folderUrl := FolderUrl{}

		err := ctx.ShouldBindUri(&folderUrl)
		if err != nil {
			ctx.AbortWithError(http.StatusBadRequest, err)
			return
		}

		// выделить место для папки
		// BUG: папка может уже существовать и тогда реальное увеличение
		// занимаемого места будет не таким большим как ContentLength,
		// т.к. некоторые файлы будут усечены и записаны заново
		fldr, err := s.Allocate(ctx.Request.ContentLength)
		if err != nil {
			if errors.Is(err, storage.ErrFileIsTooLarge) {
				ctx.AbortWithError(http.StatusRequestEntityTooLarge, err)
				return
			}

			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		// Read multipart form
		reader, err := ctx.Request.MultipartReader()
		if err != nil {
			ctx.AbortWithError(http.StatusBadRequest, fmt.Errorf("start multipart reader: %w", err))
			return
		}

		for {
			part, err := reader.NextPart()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}

				ctx.AbortWithError(http.StatusInternalServerError, fmt.Errorf("read next form part: %w", err))
				return
			}
			defer part.Close()

			filename := part.FileName()
			if filename == "" {
				continue
			}

			_, err = fldr.Put(part, filepath.Join(folderUrl.Folder, filename))
			if err != nil {
				ctx.AbortWithError(http.StatusInternalServerError, fmt.Errorf("push file to storage: %w", err))
				return
			}
		}

		ctx.Redirect(http.StatusSeeOther, ctx.Request.URL.Path)
	}
}

func MakeDeleteFolder(s *storage.Storage, basePath string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		folderUrl := FolderUrl{}

		err := ctx.ShouldBindUri(&folderUrl)
		if err != nil {
			ctx.AbortWithError(http.StatusBadRequest, err)
			return
		}

		// достать список файлов в папке из бд
		files, err := s.List(folderUrl.Folder)
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		for _, file := range files {
			err = s.Delete(file.Url)
			if err != nil {
				ctx.AbortWithError(http.StatusInternalServerError, err)
				return
			}
		}

		ctx.Redirect(http.StatusSeeOther, basePath)
	}
}

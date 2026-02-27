package routes

import (
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/oexlkinq/box/storage"
)

type FormFolderUrl struct {
	Folder string `form:"folder" binding:"required,max=256"`
}

func MakeFolderPage(s *storage.Storage, basePath string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		form := FormFolderUrl{}

		err := ctx.ShouldBind(&form)
		if err != nil {
			ctx.AbortWithError(http.StatusBadRequest, err)
			return
		}

		ctx.Redirect(http.StatusSeeOther, filepath.Join(basePath, form.Folder))
	}
}

func StartPage(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "startpage.tmpl", struct{}{})
}

package routes

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/oexlkinq/box/storage"
)

//go:embed templates
var tmplFS embed.FS

func SetupRoutes(r *gin.Engine, s *storage.Storage, basePath string) *gin.Engine {
	sub, _ := fs.Sub(tmplFS, "templates")
	r.LoadHTMLFS(http.FS(sub), "*.tmpl")

	// root
	r.GET("/", StartPage)
	r.POST("/", MakeFolderPage(s, basePath))

	// file
	r.GET("/:folder/:file", MakeGetFile(s))

	deleteFile := MakeDeleteFile(s, basePath)
	r.DELETE("/:folder/:file", deleteFile)
	r.GET("/del/:folder/:file", deleteFile)

	//folder
	r.GET("/:folder", MakeGetFolder(s, basePath))
	r.POST("/:folder", MakePostFolder(s, basePath))

	deleteFolder := MakeDeleteFolder(s, basePath)
	r.DELETE("/:folder", deleteFolder)
	r.GET("/del/:folder", deleteFolder)

	return r
}

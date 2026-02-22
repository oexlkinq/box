package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/oexlkinq/box/storage"
)

func SetupRoutes(r *gin.Engine, s *storage.Storage) *gin.Engine {
	r.GET("/:folder/:file", MakeGetFile(s))
	r.POST("/:folder/:file", MakePostFile(s))
	r.DELETE("/:folder/:file", MakeDeleteFile(s))

	return r
}

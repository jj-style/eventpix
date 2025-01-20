package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jj-style/eventpix/internal/service"
)

func handleStorage(r *gin.RouterGroup, svc service.StorageService) {
	r.GET("/thumbnail/:id", func(c *gin.Context) {
		id := c.Param("id")
		fname, got, err := svc.GetThumbnail(c, id)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fname))
		c.Header("Cache-Control", "max-age=3600") // proxies cache for 1 hour
		c.Data(http.StatusOK, "application/octet-stream", got)
	})
	r.GET("/picture/:id", func(c *gin.Context) {
		id := c.Param("id")
		fname, got, err := svc.GetPicture(c, id)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fname))
		c.Header("Cache-Control", "max-age=3600") // proxies cache for 1 hour
		c.Data(http.StatusOK, "application/octet-stream", got)
	})
}

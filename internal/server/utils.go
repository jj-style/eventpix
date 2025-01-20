package server

import (
	"github.com/gin-gonic/gin"
)

func AbortWithStatus(c *gin.Context, code int) {
	c.Status(code)
	c.Abort()
}

func AbortWithError(c *gin.Context, code int, err error) {
	c.Status(code)
	c.Error(err)
	c.Abort()
}

package handler

import "github.com/gin-gonic/gin"

func HealthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Write([]byte("OK!"))
	}
}

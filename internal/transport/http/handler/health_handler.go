package handler

import "github.com/gin-gonic/gin"

func HealthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{
			"name":   "Stasis Server API",
			"status": "healthy",
		})
	}
}

package middleware

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
	// Apply CORS middleware
	return cors.New(cors.Config{
		AllowOrigins: allowedOrigins,
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"},
		AllowHeaders: []string{
			"Origin",
			"Content-Length",
			"Content-Type",
			"Authorization",
			"Accept",
			"Accept-Encoding",
			"Accept-Language",
			"Cache-Control",
			"Cookie",
			"X-Requested-With",
			"X-Auth-Token",
		},
		ExposeHeaders: []string{
			"Content-Length",
			"Content-Type",
			"Set-Cookie",
			"Authorization",
		},
		AllowCredentials: true,
		MaxAge:           12 * 60 * 60, // 12 hours preflight cache
	})
}

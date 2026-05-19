package middleware

import (
	"qris-latency-optimizer/config"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func allowedOrigins() []string {
	origins := config.App.CORSAllowedOrigins
	if origins == "" {
		origins = "http://localhost:5173,http://127.0.0.1:5173"
	}

	allowed := []string{}
	for _, origin := range strings.Split(origins, ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			allowed = append(allowed, origin)
		}
	}

	return allowed
}

func isAllowedOrigin(origin string, allowedList []string) bool {
	for _, allowed := range allowedList {
		if origin == allowed {
			return true
		}
	}
	if strings.HasSuffix(origin, ":5173") || strings.HasSuffix(origin, ":5174") {
		return true
	}
	if strings.HasSuffix(origin, ":8080") {
		return true
	}
	return false
}

func CorsHandler(r *gin.Engine) {
	allowedList := allowedOrigins()

	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			return isAllowedOrigin(origin, allowedList)
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))
}

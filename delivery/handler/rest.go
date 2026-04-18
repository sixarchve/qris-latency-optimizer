package handler

import (
	"qris-latency-optimizer/usecase/service"

	"github.com/gin-gonic/gin"
)

func Rest(r *gin.Engine) {
	r.GET("/api/qris", service.GenerateDynamic)

	r.GET("/ping", service.Ping)
}
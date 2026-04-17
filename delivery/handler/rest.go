package handler

import (
	"qris-latency-optimizer/usecase/service"

	"github.com/gin-gonic/gin"
)

func Rest(r *gin.Engine) {
	r.GET("/ping", service.Ping)
	
	r.GET("/api/qris", GenerateQRIS)
}
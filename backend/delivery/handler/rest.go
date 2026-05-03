package handler

import (
	"qris-latency-optimizer/usecase/customer"
	"qris-latency-optimizer/usecase/service"

	"github.com/gin-gonic/gin"
)

func Rest(r *gin.Engine) {
	r.GET("/api/qris", service.GenerateDynamic)

	r.GET("/ping", service.Ping)

	// New transaction endpoints
	r.POST("/api/transactions/scan", customer.ScanQR)
	r.GET("/api/transactions/:id", service.GetTransactionStatus)
	r.POST("/api/transactions/:id/confirm", customer.ConfirmPayment)
}

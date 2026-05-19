package handler

import (
	"qris-latency-optimizer/delivery/middleware"

	"github.com/gin-gonic/gin"
)

type Handlers struct {
	Merchant    *MerchantHandler
	QRIS        *QRISHandler
	Transaction *TransactionHandler
	Monitor     *MonitorHandler
	Ping        *PingHandler
}

func SetupRouter(h *Handlers) *gin.Engine {
	r := gin.Default()
	middleware.CorsHandler(r)
	r.Use(middleware.LatencyTracker())

	r.GET("/api/qris", h.QRIS.GenerateDynamic)
	r.GET("/api/merchants", h.Merchant.GetMerchants)
	r.GET("/api/transactions/:id", h.Transaction.GetTransactionStatus)
	r.POST("/api/transactions/scan", h.Transaction.ScanQR)
	r.POST("/api/transactions/:id/confirm", h.Transaction.ConfirmPaymentAsync)
	r.POST("/api/transactions/:id/confirm-sync", h.Transaction.ConfirmPaymentSync)
	r.GET("/api/ping", h.Ping.Ping)

	h.Monitor.RegisterRoutes(r)

	return r
}

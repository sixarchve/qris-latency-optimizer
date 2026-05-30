package handler

import (
	"qris-latency-optimizer/delivery/middleware"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Handlers struct {
	Merchant    *MerchantHandler
	QRIS        *QRISHandler
	Transaction *TransactionHandler
	Ping        *PingHandler
	Telemetry   *TelemetryHandler
}

func SetupRouter(h *Handlers) *gin.Engine {
	r := gin.Default()
	middleware.CorsHandler(r)
	r.Use(middleware.PrometheusMiddleware())

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	r.GET("/api/qris", h.QRIS.GenerateDynamic)
	r.GET("/api/merchants", h.Merchant.GetMerchants)
	r.GET("/api/transactions/:id", h.Transaction.GetTransactionStatus)
	r.POST("/api/transactions/scan", h.Transaction.ScanQR)
	r.POST("/api/transactions/:id/confirm", h.Transaction.ConfirmPaymentAsync)
	r.POST("/api/transactions/:id/confirm-sync", h.Transaction.ConfirmPaymentSync)
	r.POST("/api/telemetry", h.Telemetry.ReceiveTelemetry)
	r.GET("/api/ping", h.Ping.Ping)

	return r
}

package handler

import (
	"qris-latency-optimizer/internal/websocket"
	"qris-latency-optimizer/usecase/customer"
	"qris-latency-optimizer/usecase/service"

	"github.com/gin-gonic/gin"
)

// Rest - register all API routes
func Rest(r *gin.Engine, wsHub *websocket.Hub) {
	// Create QR code endpoint
	r.GET("/api/qris", service.GenerateDynamic)

	// Backend endpoints
	r.GET("/api/merchants", service.GetMerchants)
	r.GET("/api/transactions/:id", service.GetTransactionStatus)

	// Customer endpoints
	r.POST("/api/transactions/scan", customer.ScanQR)
	r.POST("/api/transactions/:id/confirm", customer.ConfirmPayment)

	// Health check
	r.GET("/api/ping", service.Ping)
	r.GET("/api/ws/status", func(c *gin.Context) {
		merchantID := c.Query("merchant_id")
		response := gin.H{
			"connected_count": wsHub.GetConnectedCount(),
		}
		if merchantID != "" {
			response["merchant_id"] = merchantID
			response["merchant_connected"] = wsHub.IsMerchantConnected(merchantID)
			response["merchant_connection_count"] = wsHub.GetMerchantConnectionCount(merchantID)
			response["pending_notifications"] = wsHub.GetPendingCount(merchantID)
		}
		c.JSON(200, response)
	})

	// WebSocket endpoint
	r.GET("/ws", func(c *gin.Context) {
		wsHub.HandleWebSocket(c)
	})
}

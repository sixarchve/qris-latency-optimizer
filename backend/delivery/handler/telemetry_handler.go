package handler

import (
	"net/http"
	"qris-latency-optimizer/delivery/middleware"

	"github.com/gin-gonic/gin"
)

type TelemetryPayload struct {
	Path            string  `json:"path" binding:"required"`
	Method          string  `json:"method" binding:"required"`
	DurationMs      float64 `json:"client_duration_ms" binding:"required"`
}

type TelemetryHandler struct{}

func NewTelemetryHandler() *TelemetryHandler {
	return &TelemetryHandler{}
}

func (h *TelemetryHandler) ReceiveTelemetry(c *gin.Context) {
	var payload TelemetryPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
		return
	}

	// Convert ms to seconds
	durationSec := payload.DurationMs / 1000.0

	// Record the metric using our Prometheus middleware function
	middleware.RecordClientLatency(payload.Method, payload.Path, durationSec)

	c.JSON(http.StatusOK, gin.H{"status": "recorded"})
}

package handler

import (
	"qris-latency-optimizer/delivery/middleware"
	"qris-latency-optimizer/internal/monitor"

	"github.com/gin-gonic/gin"
)

type MonitorHandler struct{}

func NewMonitorHandler() *MonitorHandler {
	return &MonitorHandler{}
}

func (h *MonitorHandler) RegisterRoutes(r *gin.Engine) {
	r.GET("/api/monitor/system", monitor.GetSystemMonitor)
	r.POST("/api/monitor/k6/data", monitor.PostK6Data)
	r.POST("/api/monitor/k6/summary", monitor.PostK6Summary)
	r.GET("/api/monitor/k6", monitor.GetK6Dashboard)
	r.DELETE("/api/monitor/k6", monitor.ClearK6Data)
	r.GET("/api/monitor/live", middleware.GetLiveLatency)

	r.StaticFile("/monitor", "./monitoring/index.html")
	r.StaticFile("/latency", "./monitoring/latency.html")
}

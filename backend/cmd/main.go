package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"qris-latency-optimizer/delivery/handler"
	"qris-latency-optimizer/internal/websocket"
	"qris-latency-optimizer/repository/database"
	"qris-latency-optimizer/repository/rabbitmq"
	"qris-latency-optimizer/repository/redis"
	"qris-latency-optimizer/usecase/service"
	"qris-latency-optimizer/worker"

	"github.com/gin-gonic/gin"
)

func main() {
	fmt.Println("=== QRIS Latency Optimizer Starting ===")

	// 1. Load environment
	database.LoadEnv()

	// 2. Initialize WebSocket config (OPTIMIZED)
	websocket.InitWSConfig()

	// 3. Connect to PostgreSQL
	database.ConnectDB()
	fmt.Println("✓ PostgreSQL connected & migrated")

	// 4. Connect to Redis
	redis.ConnectRedis()
	redis.WarmUpCache()
	fmt.Println("✓ Redis connected & cache warmed")

	// 5. Connect to RabbitMQ
	rabbitmq.ConnectRabbitMQ()
	defer rabbitmq.Close()

	// 6. Initialize WebSocket Hub
	wsHub := websocket.NewHub()
	go wsHub.Run()
	fmt.Println("✓ WebSocket Hub initialized")

	// 7. Set WebSocket hub reference
	worker.SetWSHub(wsHub)

	// 8. Start consumer worker
	worker.StartPaymentConsumer()
	fmt.Println("✓ Consumer worker started")

	// --- HTTP Server ---
	r := gin.New()
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/ws", "/api/ping"},
	}))
	r.Use(gin.Recovery())
	handler.CorsHandler(r)
	r.Use(service.LatencyTracker())
	handler.Rest(r, wsHub)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		fmt.Println("=== Server running on :8080 ===")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\n=== Shutting down gracefully ===")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	rabbitmq.Close()
	fmt.Println("=== Shutdown complete ===")
}

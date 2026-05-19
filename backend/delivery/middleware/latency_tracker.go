package middleware

import (
	"sync"
	"time"

	"qris-latency-optimizer/internal/monitor"

	"github.com/gin-gonic/gin"
)

// LiveEvent represents a single tracked API request
type LiveEvent struct {
	Timestamp  time.Time `json:"timestamp"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	StatusCode int       `json:"status_code"`
	LatencyMs  float64   `json:"latency_ms"`
	ClientIP   string    `json:"client_ip"`
	Endpoint   string    `json:"endpoint"` // friendly name: "scan", "confirm", "confirm-sync", "qris", etc.
}

const maxLiveEvents = 500

var (
	liveEvents    []LiveEvent
	liveLock      sync.RWMutex
	liveListeners []chan LiveEvent
	listenerLock  sync.Mutex
)

// trackedEndpoints maps route patterns to friendly names
var trackedEndpoints = map[string]string{
	"/api/transactions/scan": "scan",
	"/api/qris":              "qris",
	"/api/merchants":         "merchants",
	"/api/ping":              "ping",
}

func classifyEndpoint(path, method string) string {
	// Exact matches first
	if name, ok := trackedEndpoints[path]; ok {
		return name
	}
	// Pattern matches for parameterized routes
	if len(path) > 20 && method == "POST" {
		if len(path) > 50 && path[len(path)-8:] == "-sync" {
			return "confirm-sync"
		}
		if len(path) > 45 && path[len(path)-7:] == "confirm" {
			return "confirm"
		}
	}
	// Check by suffix for /api/transactions/:id/confirm patterns
	if method == "POST" {
		for i := len(path) - 1; i >= 0; i-- {
			if path[i] == '/' {
				suffix := path[i+1:]
				if suffix == "confirm" {
					return "confirm"
				}
				if suffix == "confirm-sync" {
					return "confirm-sync"
				}
				break
			}
		}
	}
	if method == "GET" && len(path) > 18 {
		// /api/transactions/:id (GET status check)
		prefix := "/api/transactions/"
		if len(path) > len(prefix) && path[:len(prefix)] == prefix {
			return "status-check"
		}
	}
	return ""
}

// LatencyTracker is a Gin middleware that records latency for API endpoints
func LatencyTracker() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		endpoint := classifyEndpoint(c.Request.URL.Path, c.Request.Method)
		if endpoint == "" {
			return // skip non-API or monitoring endpoints
		}

		latency := time.Since(start)
		event := LiveEvent{
			Timestamp:  time.Now(),
			Method:     c.Request.Method,
			Path:       c.Request.URL.Path,
			StatusCode: c.Writer.Status(),
			LatencyMs:  float64(latency.Microseconds()) / 1000.0,
			ClientIP:   c.ClientIP(),
			Endpoint:   endpoint,
		}

		liveLock.Lock()
		liveEvents = append(liveEvents, event)
		if len(liveEvents) > maxLiveEvents {
			liveEvents = liveEvents[len(liveEvents)-maxLiveEvents:]
		}
		liveLock.Unlock()

		// Also push to k6 store for the dashboard charts
		if endpoint == "confirm" || endpoint == "confirm-sync" {
			scenario := "Event_Driven_Async"
			if endpoint == "confirm-sync" {
				scenario = "Synchronous_DB"
			}
			monitor.RecordLatencyEvent(scenario, event.Timestamp, event.LatencyMs, event.StatusCode)
		}
	}
}

// GetLiveLatency returns the recent live events for the dashboard
func GetLiveLatency(c *gin.Context) {
	liveLock.RLock()
	defer liveLock.RUnlock()

	// Build per-endpoint stats
	type endpointStats struct {
		Count  int     `json:"count"`
		AvgMs  float64 `json:"avg_ms"`
		MinMs  float64 `json:"min_ms"`
		MaxMs  float64 `json:"max_ms"`
		P95Ms  float64 `json:"p95_ms"`
		LastMs float64 `json:"last_ms"`
		Errors int     `json:"errors"`
	}

	statsMap := make(map[string]*struct {
		vals   []float64
		errors int
	})

	// Last N events for the feed
	feedSize := 50
	start := 0
	if len(liveEvents) > feedSize {
		start = len(liveEvents) - feedSize
	}
	recentEvents := liveEvents[start:]

	for _, ev := range liveEvents {
		if _, ok := statsMap[ev.Endpoint]; !ok {
			statsMap[ev.Endpoint] = &struct {
				vals   []float64
				errors int
			}{}
		}
		s := statsMap[ev.Endpoint]
		s.vals = append(s.vals, ev.LatencyMs)
		if ev.StatusCode >= 400 {
			s.errors++
		}
	}

	result := make(map[string]endpointStats)
	for ep, s := range statsMap {
		if len(s.vals) == 0 {
			continue
		}
		sum := 0.0
		minV := s.vals[0]
		maxV := s.vals[0]
		for _, v := range s.vals {
			sum += v
			if v < minV {
				minV = v
			}
			if v > maxV {
				maxV = v
			}
		}
		// Simple p95 without sorting (approximate)
		sorted := make([]float64, len(s.vals))
		copy(sorted, s.vals)
		// Insertion sort for small arrays
		for i := 1; i < len(sorted); i++ {
			key := sorted[i]
			j := i - 1
			for j >= 0 && sorted[j] > key {
				sorted[j+1] = sorted[j]
				j--
			}
			sorted[j+1] = key
		}
		p95Idx := int(float64(len(sorted)) * 0.95)
		if p95Idx >= len(sorted) {
			p95Idx = len(sorted) - 1
		}

		result[ep] = endpointStats{
			Count:  len(s.vals),
			AvgMs:  sum / float64(len(s.vals)),
			MinMs:  minV,
			MaxMs:  maxV,
			P95Ms:  sorted[p95Idx],
			LastMs: s.vals[len(s.vals)-1],
			Errors: s.errors,
		}
	}

	c.JSON(200, gin.H{
		"stats":     result,
		"recent":    recentEvents,
		"total":     len(liveEvents),
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

package monitor

import (
	"math"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// K6DataPoint represents a single metric sample from k6
type K6DataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Scenario  string    `json:"scenario"` // "Event_Driven_Async" or "Synchronous_DB"
	Metric    string    `json:"metric"`   // "http_req_duration", "http_reqs", "vus", etc.
	Value     float64   `json:"value"`
	Status    int       `json:"status,omitempty"` // HTTP status code
	Error     bool      `json:"error,omitempty"`
}

// K6BatchPayload is a batch of data points sent from k6
type K6BatchPayload struct {
	Points []K6DataPoint `json:"points"`
}

// K6TestRun tracks the state of a running test
type K6TestRun struct {
	Scenario  string    `json:"scenario"`
	StartedAt time.Time `json:"started_at"`
	IsActive  bool      `json:"is_active"`
}

var (
	k6Store     = make(map[string][]K6DataPoint) // scenario -> points
	k6Runs      = make(map[string]*K6TestRun)    // scenario -> run info
	k6StoreLock sync.RWMutex
)

// PostK6Data receives metric data from k6 scripts via handleSummary
func PostK6Data(c *gin.Context) {
	var payload K6BatchPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	k6StoreLock.Lock()
	defer k6StoreLock.Unlock()

	for _, point := range payload.Points {
		if point.Timestamp.IsZero() {
			point.Timestamp = time.Now()
		}
		scenario := point.Scenario
		if scenario == "" {
			scenario = "unknown"
		}

		k6Store[scenario] = append(k6Store[scenario], point)

		// Track active runs
		if _, exists := k6Runs[scenario]; !exists {
			k6Runs[scenario] = &K6TestRun{
				Scenario:  scenario,
				StartedAt: point.Timestamp,
				IsActive:  true,
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"received": len(payload.Points),
		"status":   "ok",
	})
}

// RecordLatencyEvent allows internal components like the latency tracker middleware to record data
func RecordLatencyEvent(scenario string, timestamp time.Time, durationMs float64, statusCode int) {
	k6StoreLock.Lock()
	defer k6StoreLock.Unlock()

	point := K6DataPoint{
		Timestamp: timestamp,
		Scenario:  scenario,
		Metric:    "http_req_duration",
		Value:     durationMs,
		Status:    statusCode,
		Error:     statusCode >= 400,
	}

	k6Store[scenario] = append(k6Store[scenario], point)

	if _, exists := k6Runs[scenario]; !exists {
		k6Runs[scenario] = &K6TestRun{
			Scenario:  scenario,
			StartedAt: timestamp,
			IsActive:  true,
		}
	}
}

// K6Summary sends the final summary from k6's handleSummary
type K6Summary struct {
	Scenario     string  `json:"scenario"`
	TotalReqs    int     `json:"total_reqs"`
	AvgDuration  float64 `json:"avg_duration"`
	P95Duration  float64 `json:"p95_duration"`
	P99Duration  float64 `json:"p99_duration"`
	MinDuration  float64 `json:"min_duration"`
	MaxDuration  float64 `json:"max_duration"`
	ErrorRate    float64 `json:"error_rate"`
	Throughput   float64 `json:"throughput"`
	ChecksPass   int     `json:"checks_pass"`
	ChecksFail   int     `json:"checks_fail"`
	DataSent     float64 `json:"data_sent"`
	DataReceived float64 `json:"data_received"`
	MaxVUs       int     `json:"max_vus"`
	Duration     float64 `json:"duration"`
}

var (
	k6Summaries     = make(map[string]*K6Summary)
	k6SummariesLock sync.RWMutex
)

// PostK6Summary receives the test summary
func PostK6Summary(c *gin.Context) {
	var summary K6Summary
	if err := c.ShouldBindJSON(&summary); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	k6SummariesLock.Lock()
	k6Summaries[summary.Scenario] = &summary
	k6SummariesLock.Unlock()

	// Mark run as completed
	k6StoreLock.Lock()
	if run, exists := k6Runs[summary.Scenario]; exists {
		run.IsActive = false
	}
	k6StoreLock.Unlock()

	c.JSON(http.StatusOK, gin.H{"status": "ok", "scenario": summary.Scenario})
}

// GetK6Dashboard returns aggregated data for the dashboard
func GetK6Dashboard(c *gin.Context) {
	k6StoreLock.RLock()
	defer k6StoreLock.RUnlock()
	k6SummariesLock.RLock()
	defer k6SummariesLock.RUnlock()

	scenarios := make(map[string]gin.H)

	for scenario, points := range k6Store {
		// Collect http_req_duration values for time-series
		var durations []float64
		var timestamps []string
		var errorCount, totalCount int
		var statusCodes = make(map[int]int)
		var vusPoints []gin.H
		var rpsPoints []gin.H

		// Bucket durations by second for time-series
		type bucket struct {
			durations []float64
			errors    int
			total     int
			vus       float64
		}
		buckets := make(map[int64]*bucket)

		for _, p := range points {
			ts := p.Timestamp.Unix()
			if _, ok := buckets[ts]; !ok {
				buckets[ts] = &bucket{}
			}
			b := buckets[ts]

			switch p.Metric {
			case "http_req_duration":
				b.durations = append(b.durations, p.Value)
				durations = append(durations, p.Value)
				b.total++
				if p.Error {
					b.errors++
					errorCount++
				}
				totalCount++
				if p.Status > 0 {
					statusCodes[p.Status]++
				}
			case "vus":
				b.vus = p.Value
			}
		}

		// Sort bucket keys
		var sortedKeys []int64
		for k := range buckets {
			sortedKeys = append(sortedKeys, k)
		}
		sort.Slice(sortedKeys, func(i, j int) bool { return sortedKeys[i] < sortedKeys[j] })

		// Build time series
		var p95Series []gin.H
		var avgSeries []gin.H
		var errorSeries []gin.H

		for _, ts := range sortedKeys {
			b := buckets[ts]
			tsStr := time.Unix(ts, 0).Format("15:04:05")
			timestamps = append(timestamps, tsStr)

			if len(b.durations) > 0 {
				sort.Float64s(b.durations)
				p95Idx := int(math.Ceil(float64(len(b.durations))*0.95)) - 1
				if p95Idx < 0 {
					p95Idx = 0
				}
				if p95Idx >= len(b.durations) {
					p95Idx = len(b.durations) - 1
				}

				avg := 0.0
				for _, d := range b.durations {
					avg += d
				}
				avg /= float64(len(b.durations))

				p95Series = append(p95Series, gin.H{"t": tsStr, "v": math.Round(b.durations[p95Idx]*100) / 100})
				avgSeries = append(avgSeries, gin.H{"t": tsStr, "v": math.Round(avg*100) / 100})
			}

			errRate := 0.0
			if b.total > 0 {
				errRate = float64(b.errors) / float64(b.total) * 100
			}
			errorSeries = append(errorSeries, gin.H{"t": tsStr, "v": math.Round(errRate*100) / 100})

			rpsPoints = append(rpsPoints, gin.H{"t": tsStr, "v": b.total})
			vusPoints = append(vusPoints, gin.H{"t": tsStr, "v": b.vus})
		}

		// Overall aggregation
		sort.Float64s(durations)
		var overallP95, overallP99, overallAvg, overallMin, overallMax float64
		if len(durations) > 0 {
			overallMin = durations[0]
			overallMax = durations[len(durations)-1]
			sum := 0.0
			for _, d := range durations {
				sum += d
			}
			overallAvg = sum / float64(len(durations))

			p95Idx := int(math.Ceil(float64(len(durations))*0.95)) - 1
			if p95Idx >= len(durations) {
				p95Idx = len(durations) - 1
			}
			overallP95 = durations[p95Idx]

			p99Idx := int(math.Ceil(float64(len(durations))*0.99)) - 1
			if p99Idx >= len(durations) {
				p99Idx = len(durations) - 1
			}
			overallP99 = durations[p99Idx]
		}

		scenarioData := gin.H{
			"timestamps":   timestamps,
			"p95_series":   p95Series,
			"avg_series":   avgSeries,
			"error_series": errorSeries,
			"rps_series":   rpsPoints,
			"vus_series":   vusPoints,
			"stats": gin.H{
				"total_requests": totalCount,
				"error_count":    errorCount,
				"error_rate":     math.Round(float64(errorCount)/math.Max(float64(totalCount), 1)*10000) / 100,
				"p95":            math.Round(overallP95*100) / 100,
				"p99":            math.Round(overallP99*100) / 100,
				"avg":            math.Round(overallAvg*100) / 100,
				"min":            math.Round(overallMin*100) / 100,
				"max":            math.Round(overallMax*100) / 100,
			},
			"status_codes": statusCodes,
		}

		// Attach summary if available
		if summary, exists := k6Summaries[scenario]; exists {
			scenarioData["summary"] = summary
		}

		scenarios[scenario] = scenarioData
	}

	// Build run status
	runs := []gin.H{}
	for _, run := range k6Runs {
		runs = append(runs, gin.H{
			"scenario":   run.Scenario,
			"started_at": run.StartedAt.Format(time.RFC3339),
			"is_active":  run.IsActive,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"scenarios": scenarios,
		"runs":      runs,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// ClearK6Data resets the stored data
func ClearK6Data(c *gin.Context) {
	k6StoreLock.Lock()
	k6Store = make(map[string][]K6DataPoint)
	k6Runs = make(map[string]*K6TestRun)
	k6StoreLock.Unlock()

	k6SummariesLock.Lock()
	k6Summaries = make(map[string]*K6Summary)
	k6SummariesLock.Unlock()

	c.JSON(http.StatusOK, gin.H{"status": "cleared"})
}

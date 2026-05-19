package monitor

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"qris-latency-optimizer/repository/rabbitmq"
	"qris-latency-optimizer/repository/redis"

	"github.com/gin-gonic/gin"
)

var startTime = time.Now()

// cpuTimes holds raw CPU tick counts from /proc/stat
type cpuTimes struct {
	User    uint64
	Nice    uint64
	System  uint64
	Idle    uint64
	IOWait  uint64
	IRQ     uint64
	SoftIRQ uint64
	Steal   uint64
}

func (c cpuTimes) total() uint64 {
	return c.User + c.Nice + c.System + c.Idle + c.IOWait + c.IRQ + c.SoftIRQ + c.Steal
}

func (c cpuTimes) idle() uint64 {
	return c.Idle + c.IOWait
}

// readCPUTimes reads CPU times from /proc/stat (Linux only)
func readCPUTimes() (*cpuTimes, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return nil, fmt.Errorf("cannot read /proc/stat: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) < 9 {
				return nil, fmt.Errorf("unexpected /proc/stat format")
			}

			values := make([]uint64, 8)
			for i := 0; i < 8; i++ {
				v, err := strconv.ParseUint(fields[i+1], 10, 64)
				if err != nil {
					return nil, fmt.Errorf("parse error in /proc/stat: %w", err)
				}
				values[i] = v
			}

			return &cpuTimes{
				User:    values[0],
				Nice:    values[1],
				System:  values[2],
				Idle:    values[3],
				IOWait:  values[4],
				IRQ:     values[5],
				SoftIRQ: values[6],
				Steal:   values[7],
			}, nil
		}
	}

	return nil, fmt.Errorf("/proc/stat: cpu line not found")
}

// measureCPUUsage samples CPU usage over a short interval
func measureCPUUsage() (float64, error) {
	t1, err := readCPUTimes()
	if err != nil {
		return 0, err
	}

	time.Sleep(200 * time.Millisecond)

	t2, err := readCPUTimes()
	if err != nil {
		return 0, err
	}

	totalDelta := float64(t2.total() - t1.total())
	idleDelta := float64(t2.idle() - t1.idle())

	if totalDelta == 0 {
		return 0, nil
	}

	usage := ((totalDelta - idleDelta) / totalDelta) * 100.0
	return usage, nil
}

// readMemInfo reads memory statistics from /proc/meminfo (Linux only)
func readMemInfo() (totalMB, usedMB, freeMB uint64, usedPercent float64, err error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("cannot read /proc/meminfo: %w", err)
	}
	defer file.Close()

	var memTotal, memAvailable uint64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		val, parseErr := strconv.ParseUint(fields[1], 10, 64)
		if parseErr != nil {
			continue
		}

		switch fields[0] {
		case "MemTotal:":
			memTotal = val
		case "MemAvailable:":
			memAvailable = val
		}
	}

	if memTotal == 0 {
		return 0, 0, 0, 0, fmt.Errorf("could not parse MemTotal from /proc/meminfo")
	}

	totalMB = memTotal / 1024
	freeMB = memAvailable / 1024
	usedMB = totalMB - freeMB
	usedPercent = (float64(usedMB) / float64(totalMB)) * 100.0

	return totalMB, usedMB, freeMB, usedPercent, nil
}

// GetSystemMonitor returns real-time system metrics
func GetSystemMonitor(c *gin.Context) {
	// CPU usage
	cpuUsage, cpuErr := measureCPUUsage()
	cpuInfo := gin.H{
		"usage_percent": fmt.Sprintf("%.2f", cpuUsage),
		"num_cores":     runtime.NumCPU(),
	}
	if cpuErr != nil {
		cpuInfo["error"] = cpuErr.Error()
	}

	// Memory info
	memTotal, memUsed, memFree, memPercent, memErr := readMemInfo()
	memInfo := gin.H{
		"total_mb":     memTotal,
		"used_mb":      memUsed,
		"free_mb":      memFree,
		"used_percent": fmt.Sprintf("%.2f", memPercent),
	}
	if memErr != nil {
		memInfo["error"] = memErr.Error()
	}

	// Go runtime stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	goRuntime := gin.H{
		"goroutines":     runtime.NumGoroutine(),
		"heap_alloc_mb":  fmt.Sprintf("%.2f", float64(memStats.HeapAlloc)/1024/1024),
		"heap_sys_mb":    fmt.Sprintf("%.2f", float64(memStats.HeapSys)/1024/1024),
		"gc_cycles":      memStats.NumGC,
		"gc_pause_total": fmt.Sprintf("%.2fms", float64(memStats.PauseTotalNs)/1e6),
	}

	// Service health
	services := gin.H{
		"rabbitmq_connected": rabbitmq.IsConnected(),
		"redis_connected":    redis.RedisAvailable,
	}

	// Uptime
	uptime := time.Since(startTime)

	c.JSON(http.StatusOK, gin.H{
		"status":     "ok",
		"timestamp":  time.Now().Format(time.RFC3339),
		"uptime":     uptime.String(),
		"cpu":        cpuInfo,
		"memory":     memInfo,
		"go_runtime": goRuntime,
		"services":   services,
	})
}

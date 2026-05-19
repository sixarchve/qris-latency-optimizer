package monitor

import (
	"testing"
)

func TestReadCPUTimes(t *testing.T) {
	times, err := readCPUTimes()
	if err != nil {
		t.Skipf("Skipping CPU test (not Linux or no /proc/stat): %v", err)
	}

	if times.total() == 0 {
		t.Error("total CPU time should be > 0")
	}

	// Idle should be less than total
	if times.idle() > times.total() {
		t.Error("idle time should not exceed total time")
	}
}

func TestMeasureCPUUsage(t *testing.T) {
	usage, err := measureCPUUsage()
	if err != nil {
		t.Skipf("Skipping CPU usage test (not Linux): %v", err)
	}

	// Usage should be between 0 and 100
	if usage < 0 || usage > 100 {
		t.Errorf("CPU usage should be 0-100%%, got: %.2f%%", usage)
	}
}

func TestReadMemInfo(t *testing.T) {
	totalMB, usedMB, freeMB, usedPercent, err := readMemInfo()
	if err != nil {
		t.Skipf("Skipping memory test (not Linux): %v", err)
	}

	if totalMB == 0 {
		t.Error("total memory should be > 0")
	}

	if usedMB+freeMB != totalMB {
		t.Errorf("used(%d) + free(%d) should equal total(%d)", usedMB, freeMB, totalMB)
	}

	if usedPercent < 0 || usedPercent > 100 {
		t.Errorf("memory usage should be 0-100%%, got: %.2f%%", usedPercent)
	}
}

func TestCPUTimesCalculations(t *testing.T) {
	times := cpuTimes{
		User:    100,
		Nice:    10,
		System:  50,
		Idle:    800,
		IOWait:  20,
		IRQ:     5,
		SoftIRQ: 3,
		Steal:   2,
	}

	expectedTotal := uint64(100 + 10 + 50 + 800 + 20 + 5 + 3 + 2)
	if times.total() != expectedTotal {
		t.Errorf("expected total %d, got %d", expectedTotal, times.total())
	}

	expectedIdle := uint64(800 + 20)
	if times.idle() != expectedIdle {
		t.Errorf("expected idle %d, got %d", expectedIdle, times.idle())
	}
}

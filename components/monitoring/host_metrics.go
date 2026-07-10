package monitoring

import (
	"time"

	"github.com/danielvm/bigbase/kernel"

	gocpu "github.com/shirou/gopsutil/v3/cpu"
	godisk "github.com/shirou/gopsutil/v3/disk"
	gomem "github.com/shirou/gopsutil/v3/mem"
	gonet "github.com/shirou/gopsutil/v3/net"
)

// HostMetrics holds a point-in-time snapshot of OS-level resource usage.
type HostMetrics struct {
	CPUPercent     float64 `json:"cpu_percent"`
	MemUsedBytes   int64   `json:"mem_used_bytes"`
	MemTotalBytes  int64   `json:"mem_total_bytes"`
	DiskUsedBytes  int64   `json:"disk_used_bytes"`
	DiskTotalBytes int64   `json:"disk_total_bytes"`
	NetBytesIn     int64   `json:"net_bytes_in"`
	NetBytesOut    int64   `json:"net_bytes_out"`
	CollectedAt    string  `json:"collected_at"`
}

// collectHostMetrics gathers a single snapshot of host metrics.
// cpuInterval controls how long to block for CPU sampling; pass 0 for a
// non-blocking estimate (compares to the previous call stored internally by gopsutil).
func collectHostMetrics(logger kernel.Logger) HostMetrics {
	return collectHostMetricsWithInterval(logger, time.Second)
}

func collectHostMetricsWithInterval(logger kernel.Logger, cpuInterval time.Duration) HostMetrics {
	h := HostMetrics{CollectedAt: time.Now().UTC().Format(time.RFC3339)}

	// CPU — blocking sample for cpuInterval, aggregate across all cores.
	if pcts, err := gocpu.Percent(cpuInterval, false); err != nil {
		logger.Warn("host cpu percent", "error", err)
	} else if len(pcts) > 0 {
		h.CPUPercent = pcts[0]
	}

	// Memory.
	if vm, err := gomem.VirtualMemory(); err != nil {
		logger.Warn("host memory", "error", err)
	} else {
		h.MemUsedBytes = int64(vm.Used)
		h.MemTotalBytes = int64(vm.Total)
	}

	// Disk — root mount point.
	if du, err := godisk.Usage("/"); err != nil {
		logger.Warn("host disk usage", "error", err)
	} else {
		h.DiskUsedBytes = int64(du.Used)
		h.DiskTotalBytes = int64(du.Total)
	}

	// Network — aggregate across all interfaces.
	if counters, err := gonet.IOCounters(false); err != nil {
		logger.Warn("host net io", "error", err)
	} else if len(counters) > 0 {
		h.NetBytesIn = int64(counters[0].BytesRecv)
		h.NetBytesOut = int64(counters[0].BytesSent)
	}

	return h
}

// collectHostMetricsNonBlocking gathers host metrics without a CPU blocking interval.
// It returns quickly but CPU% may be 0 on the first call.
func collectHostMetricsNonBlocking(logger kernel.Logger) HostMetrics {
	return collectHostMetricsWithInterval(logger, 0)
}

package monitoring

import (
	"net/http"
	"os"
	"runtime"
	"syscall"

	"github.com/danielvm/bigbase/kernel"
)

// ProcessInfo describes a single tracked process.
type ProcessInfo struct {
	PID    int     `json:"pid"`
	Name   string  `json:"name"`
	CPUPCT float64 `json:"cpu_pct"`
	MemMB  float64 `json:"mem_mb"`
	Status string  `json:"status"`
}

// handleProcesses returns the current BigBase process resource usage.
// Child processes (functions runtime) would be tracked here in the future;
// for now we report the current process itself.
func (m *Monitoring) handleProcesses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	pid := os.Getpid()
	cpuPct := m.sampleCPUPercent()
	memMB := selfMemMB()

	info := ProcessInfo{
		PID:    pid,
		Name:   "bigbase",
		CPUPCT: cpuPct,
		MemMB:  memMB,
		Status: "running",
	}

	kernel.WriteJSON(w, http.StatusOK, map[string]any{"processes": []ProcessInfo{info}})
}

// selfMemMB returns the current process's resident set size in MB using getrusage.
func selfMemMB() float64 {
	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
		return 0
	}
	// Maxrss is in bytes on Linux; on macOS it is in bytes too (since Go 1.22 syscall reflects that).
	// Fallback: use runtime.MemStats.
	if ru.Maxrss > 0 {
		divisor := int64(1024 * 1024)
		if runtime.GOOS == "darwin" {
			// macOS Maxrss is already in bytes.
			return float64(ru.Maxrss) / float64(divisor)
		}
		// Linux Maxrss is in kilobytes.
		return float64(ru.Maxrss) / 1024.0
	}
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return float64(ms.Alloc) / 1024 / 1024
}

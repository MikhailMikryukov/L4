package handlers

import (
	"fmt"
	"net/http"
	"runtime"
	"time"

	"L4.4/internal/metrics"
)

// MetricsHandler обрабатывает запросы к /metrics endpoint
type MetricsHandler struct {
	collector *metrics.Collector
	startTime time.Time
}

// NewMetricsHandler создает новый экземпляр MetricsHandler
func NewMetricsHandler() *MetricsHandler {
	return &MetricsHandler{
		collector: metrics.NewCollector(),
		startTime: time.Now(),
	}
}

// ServeHTTP обрабатывает HTTP запрос для метрик
func (h *MetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	memStats := h.collector.Collect()

	// Запись всех метрик
	h.writeMemoryMetrics(w, memStats)
	h.writeGCMetrics(w, memStats)
	h.writeSystemMetrics(w)
}

// writeMemoryMetrics записывает метрики памяти
func (h *MetricsHandler) writeMemoryMetrics(w http.ResponseWriter, memStats *metrics.MemStats) {
	// Alloc metrics
	writeMetric(w, "go_memstats_alloc_bytes", "gauge",
		"Number of bytes allocated and still in use.",
		float64(memStats.Alloc))

	writeMetric(w, "go_memstats_total_alloc_bytes", "counter",
		"Total number of bytes allocated, even if freed.",
		float64(memStats.TotalAlloc))

	writeMetric(w, "go_memstats_sys_bytes", "gauge",
		"Number of bytes obtained from system.",
		float64(memStats.Sys))

	writeMetric(w, "go_memstats_lookups_total", "counter",
		"Total number of pointer lookups.",
		float64(memStats.Lookups))

	writeMetric(w, "go_memstats_mallocs_total", "counter",
		"Total number of mallocs.",
		float64(memStats.Mallocs))

	writeMetric(w, "go_memstats_frees_total", "counter",
		"Total number of frees.",
		float64(memStats.Frees))

	// Heap metrics
	writeMetric(w, "go_memstats_heap_alloc_bytes", "gauge",
		"Number of heap bytes allocated and still in use.",
		float64(memStats.HeapAlloc))

	writeMetric(w, "go_memstats_heap_sys_bytes", "gauge",
		"Number of heap bytes obtained from system.",
		float64(memStats.HeapSys))

	writeMetric(w, "go_memstats_heap_idle_bytes", "gauge",
		"Number of heap bytes waiting to be used.",
		float64(memStats.HeapIdle))

	writeMetric(w, "go_memstats_heap_inuse_bytes", "gauge",
		"Number of heap bytes that are in use.",
		float64(memStats.HeapInuse))

	writeMetric(w, "go_memstats_heap_released_bytes", "gauge",
		"Number of heap bytes released to OS.",
		float64(memStats.HeapReleased))

	writeMetric(w, "go_memstats_heap_objects", "gauge",
		"Number of allocated objects.",
		float64(memStats.HeapObjects))

	// Stack metrics
	writeMetric(w, "go_memstats_stack_inuse_bytes", "gauge",
		"Number of bytes in use by the stack allocator.",
		float64(memStats.StackInuse))

	writeMetric(w, "go_memstats_stack_sys_bytes", "gauge",
		"Number of bytes obtained from system for stack allocator.",
		float64(memStats.StackSys))

	// MSpan metrics
	writeMetric(w, "go_memstats_mspan_inuse_bytes", "gauge",
		"Number of mspan structures in use.",
		float64(memStats.MSpanInuse))

	writeMetric(w, "go_memstats_mspan_sys_bytes", "gauge",
		"Number of bytes used for mspan structures.",
		float64(memStats.MSpanSys))

	// MCache metrics
	writeMetric(w, "go_memstats_mcache_inuse_bytes", "gauge",
		"Number of mcache structures in use.",
		float64(memStats.MCacheInuse))

	writeMetric(w, "go_memstats_mcache_sys_bytes", "gauge",
		"Number of bytes used for mcache structures.",
		float64(memStats.MCacheSys))

	// Other system metrics
	writeMetric(w, "go_memstats_buck_hash_sys_bytes", "gauge",
		"Number of bytes used by the profiling bucket hash table.",
		float64(memStats.BuckHashSys))

	writeMetric(w, "go_memstats_gc_sys_bytes", "gauge",
		"Number of bytes used for garbage collection system metadata.",
		float64(memStats.GCSys))

	writeMetric(w, "go_memstats_other_sys_bytes", "gauge",
		"Number of bytes used for other system allocations.",
		float64(memStats.OtherSys))

	writeMetric(w, "go_memstats_next_gc_bytes", "gauge",
		"Target heap size for next GC cycle.",
		float64(memStats.NextGC))
}

// writeGCMetrics записывает метрики GC
func (h *MetricsHandler) writeGCMetrics(w http.ResponseWriter, memStats *metrics.MemStats) {
	writeMetric(w, "go_memstats_last_gc_time_seconds", "gauge",
		"Time of last garbage collection.",
		float64(memStats.LastGC))

	writeMetric(w, "go_memstats_num_gc_total", "counter",
		"Total number of completed GC cycles.",
		float64(memStats.NumGC))

	writeMetric(w, "go_memstats_num_forced_gc_total", "counter",
		"Number of GC cycles that were forced by the application.",
		float64(memStats.NumForcedGC))

	writeMetric(w, "go_memstats_gc_cpu_fraction", "gauge",
		"Fraction of CPU time used by GC.",
		memStats.GCCPUFraction)

	// GC pause durations
	fmt.Fprintf(w, "# HELP go_gc_duration_seconds A summary of the pause duration of garbage collection cycles.\n")
	fmt.Fprintf(w, "# TYPE go_gc_duration_seconds summary\n")
	for i, pauseNs := range memStats.PauseNs {
		if i >= int(memStats.NumGC) {
			break
		}
		fmt.Fprintf(w, "go_gc_duration_seconds{quantile=\"%d\"} %f\n", i, float64(pauseNs)/1e9)
	}
	fmt.Fprintf(w, "go_gc_duration_seconds_sum %f\n", float64(memStats.PauseTotalNs)/1e9)
	fmt.Fprintf(w, "go_gc_duration_seconds_count %d\n", memStats.NumGC)

	// Current GC percentage
	writeMetric(w, "go_gc_percent", "gauge",
		"Current GC target percentage.",
		float64(memStats.GCPercent))
}

// writeSystemMetrics записывает системные метрики
func (h *MetricsHandler) writeSystemMetrics(w http.ResponseWriter) {
	writeMetric(w, "go_goroutines", "gauge",
		"Number of goroutines that currently exist.",
		float64(runtime.NumGoroutine()))

	writeMetric(w, "go_threads", "gauge",
		"Number of OS threads created.",
		float64(runtime.NumCPU()))

	writeMetric(w, "process_uptime_seconds", "gauge",
		"Uptime of the process in seconds.",
		time.Since(h.startTime).Seconds())
}

// writeMetric вспомогательная функция для записи метрики в формате Prometheus
func writeMetric(w http.ResponseWriter, name, metricType, help string, value float64) {
	fmt.Fprintf(w, "# HELP %s %s\n", name, help)
	fmt.Fprintf(w, "# TYPE %s %s\n", name, metricType)
	fmt.Fprintf(w, "%s %f\n", name, value)
}

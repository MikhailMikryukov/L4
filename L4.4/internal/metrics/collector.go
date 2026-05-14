package metrics

import (
	"runtime"
	"runtime/debug"
	"time"
)

// MemStats представляет структуру с метриками памяти и GC
type MemStats struct {
	// Основные метрики памяти
	Alloc      uint64
	TotalAlloc uint64
	Sys        uint64
	Lookups    uint64
	Mallocs    uint64
	Frees      uint64

	// Метрики кучи
	HeapAlloc    uint64
	HeapSys      uint64
	HeapIdle     uint64
	HeapInuse    uint64
	HeapReleased uint64
	HeapObjects  uint64

	// Метрики стека
	StackInuse uint64
	StackSys   uint64

	// Метрики MSpan
	MSpanInuse uint64
	MSpanSys   uint64

	// Метрики MCache
	MCacheInuse uint64
	MCacheSys   uint64

	// Другие системные метрики
	BuckHashSys uint64
	GCSys       uint64
	OtherSys    uint64
	NextGC      uint64

	// Метрики GC
	LastGC        uint64
	NumGC         uint32
	NumForcedGC   uint32
	GCCPUFraction float64
	PauseTotalNs  uint64
	PauseNs       [256]uint64
	GCPercent     int
}

// Collector собирает метрики из runtime
type Collector struct{}

// NewCollector создает новый экземпляр Collector
func NewCollector() *Collector {
	return &Collector{}
}

// Collect собирает все метрики памяти и GC
func (c *Collector) Collect() *MemStats {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	return &MemStats{
		// Основные метрики памяти
		Alloc:      ms.Alloc,
		TotalAlloc: ms.TotalAlloc,
		Sys:        ms.Sys,
		Lookups:    ms.Lookups,
		Mallocs:    ms.Mallocs,
		Frees:      ms.Frees,

		// Метрики кучи
		HeapAlloc:    ms.HeapAlloc,
		HeapSys:      ms.HeapSys,
		HeapIdle:     ms.HeapIdle,
		HeapInuse:    ms.HeapInuse,
		HeapReleased: ms.HeapReleased,
		HeapObjects:  ms.HeapObjects,

		// Метрики стека
		StackInuse: ms.StackInuse,
		StackSys:   ms.StackSys,

		// Метрики MSpan
		MSpanInuse: ms.MSpanInuse,
		MSpanSys:   ms.MSpanSys,

		// Метрики MCache
		MCacheInuse: ms.MCacheInuse,
		MCacheSys:   ms.MCacheSys,

		// Другие системные метрики
		BuckHashSys: ms.BuckHashSys,
		GCSys:       ms.GCSys,
		OtherSys:    ms.OtherSys,
		NextGC:      ms.NextGC,

		// Метрики GC
		LastGC:        ms.LastGC / 1e9, // Конвертация в секунды
		NumGC:         ms.NumGC,
		NumForcedGC:   ms.NumForcedGC,
		GCCPUFraction: ms.GCCPUFraction,
		PauseTotalNs:  ms.PauseTotalNs,
		PauseNs:       ms.PauseNs,
		GCPercent:     debug.SetGCPercent(-1),
	}
}

// GetCPUCount возвращает количество CPU
func (c *Collector) GetCPUCount() int {
	return runtime.NumCPU()
}

// GetGoroutineCount возвращает количество горутин
func (c *Collector) GetGoroutineCount() int {
	return runtime.NumGoroutine()
}

// GetUptime возвращает время работы приложения
func (c *Collector) GetUptime(startTime time.Time) time.Duration {
	return time.Since(startTime)
}

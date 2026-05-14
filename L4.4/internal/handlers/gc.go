package handlers

import (
	"encoding/json"
	"net/http"
	"runtime/debug"
	"strconv"
)

// GCHandler обрабатывает запросы к /gc/percent endpoint
type GCHandler struct{}

// NewGCHandler создает новый экземпляр GCHandler
func NewGCHandler() *GCHandler {
	return &GCHandler{}
}

// ServeHTTP обрабатывает HTTP запросы для управления GC процентом
func (h *GCHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r)
	case http.MethodPost:
		h.handlePost(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGet обрабатывает GET запрос - возвращает текущее значение GC процента
func (h *GCHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	percent := debug.SetGCPercent(-1)

	response := map[string]interface{}{
		"gc_percent": percent,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handlePost обрабатывает POST запрос - устанавливает новое значение GC процента
func (h *GCHandler) handlePost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	percentStr := r.FormValue("percent")
	percent, err := strconv.Atoi(percentStr)
	if err != nil {
		http.Error(w, "Invalid percent value", http.StatusBadRequest)
		return
	}

	if percent < 0 {
		http.Error(w, "GC percent must be non-negative", http.StatusBadRequest)
		return
	}

	oldPercent := debug.SetGCPercent(percent)

	response := map[string]interface{}{
		"old_gc_percent": oldPercent,
		"new_gc_percent": percent,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

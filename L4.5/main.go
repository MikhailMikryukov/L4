package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"strconv"
	"strings"
)

type Response struct {
	Result float64   `json:"result"`
	Nums   []float64 `json:"nums"`
}

// Неоптимизированная версия
func sumHandler(w http.ResponseWriter, r *http.Request) {
	numsStr := r.URL.Query().Get("nums")
	if numsStr == "" {
		numsStr = "1,2,3,4,5"
	}

	numStrs := strings.Split(numsStr, ",")

	// Не предаллоцирован
	var nums []float64
	var sum float64

	// fmt.Sprintf создаёт лишнюю строку
	for _, s := range numStrs {
		formatted := fmt.Sprintf("%s", strings.TrimSpace(s))
		num, err := strconv.ParseFloat(formatted, 64)
		if err != nil {
			continue
		}
		nums = append(nums, num)
		sum += num
	}

	resp := Response{
		Result: sum,
		Nums:   nums,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Оптимизированная версия
func sumOptimizedHandler(w http.ResponseWriter, r *http.Request) {
	numsStr := r.URL.Query().Get("nums")
	if numsStr == "" {
		numsStr = "1,2,3,4,5"
	}

	numStrs := strings.Split(numsStr, ",")

	// Предаллоцируем
	nums := make([]float64, 0, len(numStrs))
	var sum float64

	// Без лишних конвертаций
	for _, s := range numStrs {
		num, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
		if err != nil {
			continue
		}
		nums = append(nums, num)
		sum += num
	}

	resp := Response{
		Result: sum,
		Nums:   nums,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	http.HandleFunc("/sum", sumHandler)
	http.HandleFunc("/sum/optimized", sumOptimizedHandler)

	log.Println("   Server starting on :8080")
	log.Println("   GET /sum?nums=1,2,3           - неоптимизированная версия")
	log.Println("   GET /sum/optimized?nums=1,2,3 - оптимизированная версия")
	log.Println("   pprof: http://localhost:8080/debug/pprof/")

	http.ListenAndServe(":8080", nil)
}

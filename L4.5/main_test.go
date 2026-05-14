package main

import (
	"net/http/httptest"
	"testing"
)

func BenchmarkSum(t *testing.B) {
	req := httptest.NewRequest("GET", "/sum?nums=1,2,3,4,5,6,7,8,9,10", nil)

	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		w := httptest.NewRecorder()
		sumHandler(w, req)
	}
}

func BenchmarkSumOptimized(t *testing.B) {
	req := httptest.NewRequest("GET", "/sum/optimized?nums=1,2,3,4,5,6,7,8,9,10", nil)

	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		w := httptest.NewRecorder()
		sumOptimizedHandler(w, req)
	}
}

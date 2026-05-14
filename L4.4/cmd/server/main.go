package main

import (
	"context"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"L4.4/internal/handlers"
)

func main() {
	// Канал для graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Инициализация обработчиков
	metricsHandler := handlers.NewMetricsHandler()
	gcHandler := handlers.NewGCHandler()
	
	http.HandleFunc("/metrics", metricsHandler.ServeHTTP)
	http.HandleFunc("/gc/percent", gcHandler.ServeHTTP)
	http.HandleFunc("/health", handlers.HealthCheck)

	// Настройка сервера
	port := getEnv("PORT", "8080")
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      loggingMiddleware(http.DefaultServeMux), // Используем DefaultServeMux
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Запуск сервера в горутине
	go func() {
		log.Printf("Starting server on port %s", port)
		log.Printf("Metrics endpoint: http://localhost:%s/metrics", port)
		log.Printf("GC percent endpoint: http://localhost:%s/gc/percent", port)
		log.Printf("Pprof endpoint: http://localhost:%s/debug/pprof/", port)
		log.Printf("Health check: http://localhost:%s/health", port)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// Ожидание сигнала завершения
	<-stop
	log.Println("Shutting down server...")

	// Graceful shutdown с таймаутом
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}

// getEnv возвращает значение переменной окружения или значение по умолчанию
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// loggingMiddleware добавляет логирование всех запросов
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Создаем ResponseWriter wrapper для захвата статус кода
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		log.Printf("%s %s %d %v", r.Method, r.URL.Path, wrapped.statusCode, time.Since(start))
	})
}

// responseWriter wrapper для захвата HTTP статус кода
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

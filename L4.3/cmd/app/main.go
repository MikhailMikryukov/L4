package main

import (
	"L4.3/internal/config"
	"L4.3/internal/handlers"
	"L4.3/internal/repository/postgres"
	"L4.3/internal/worker"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	cfg := config.LoadConfig()

	// Подключение к бд
	db, err := postgres.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	// Фоновый воркер для обработки напоминаний
	reminderWorker := worker.NewReminderWorker(cfg.ReminderWorkerBuffSize)
	reminderWorker.Start()
	defer reminderWorker.Stop()

	// Фоновый воркер для архивации старых событий
	archiveWorker := worker.NewArchiveWorker(time.Duration(cfg.ArchiveEventsInterval), db)
	archiveWorker.Start()
	defer archiveWorker.Stop()

	// Настройка и создание обработчика HTTP запросов
	router := handlers.NewRouter(db, reminderWorker, cfg)

	server := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: router,
	}

	// Запускаем сервер на порту
	fmt.Println("Starting server at port", cfg.ServerPort)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("Could not start server: %v\n", err)
	}
}

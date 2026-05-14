package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
	"strconv"
)

// Config конфиг
type Config struct {
	ServerPort             string
	DatabaseURL            string
	ReminderWorkerBuffSize int
	ArchiveEventsInterval  int
	LoggerBuffSize         int
	LoggerBatchSize        int
}

// LoadConfig загружает конфиг
func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	r := getEnv("REMINDER_WORKER_BUFF_SIZE", "10")
	a := getEnv("ARCHIVE_EVENTS_INTERVAL_MINUTES", "10")
	l := getEnv("LOGGER_BUFF_SIZE", "10")
	lB := getEnv("LOGGER_BATCH_SIZE", "10")

	rBuff, _ := strconv.Atoi(r)
	aInterval, _ := strconv.Atoi(a)
	lBuff, _ := strconv.Atoi(l)
	lBatch, _ := strconv.Atoi(lB)

	return &Config{
		ServerPort:             getEnv("SERVER_PORT", "3000"),
		DatabaseURL:            getEnv("DATABASE_URL", "postgresql://user:pass@localhost:5432/sales_tracker"),
		ReminderWorkerBuffSize: rBuff,
		ArchiveEventsInterval:  aInterval,
		LoggerBuffSize:         lBuff,
		LoggerBatchSize:        lBatch,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

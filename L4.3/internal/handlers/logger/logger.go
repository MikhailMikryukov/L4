package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

// LogEntry представляет одну запись лога
type LogEntry struct {
	Timestamp time.Time   `json:"timestamp"`
	Level     string      `json:"level"`
	Message   string      `json:"message"`
	Path      string      `json:"path,omitempty"`
	Method    string      `json:"method,omitempty"`
	Duration  int64       `json:"duration_ms,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

// AsyncLogger асинхронный логгер
type AsyncLogger struct {
	logChan   chan LogEntry
	done      chan struct{}
	writer    *log.Logger
	batchSize int
}

// NewAsyncLogger создает новый асинхронный логгер
func NewAsyncLogger(bufferSize int, batchSize int) *AsyncLogger {
	logger := &AsyncLogger{
		logChan:   make(chan LogEntry, bufferSize),
		done:      make(chan struct{}),
		writer:    log.New(os.Stdout, "", 0),
		batchSize: batchSize,
	}

	// Запускаем горутину-обработчик
	go logger.processLogs()

	return logger
}

// processLogs обрабатывает логи из канала
func (al *AsyncLogger) processLogs() {
	batch := make([]LogEntry, 0, al.batchSize)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case entry, ok := <-al.logChan:
			if !ok {
				// Канал закрыт, записываем остатки и выходим
				al.flushBatch(batch)
				close(al.done)
				return
			}

			batch = append(batch, entry)

			// Если набралась полная пачка - записываем
			if len(batch) >= al.batchSize {
				al.flushBatch(batch)
				batch = batch[:0] // очищаем слайс
			}

		case <-ticker.C:
			// По таймеру записываем накопленные логи
			if len(batch) > 0 {
				al.flushBatch(batch)
				batch = batch[:0]
			}
		}
	}
}

// flushBatch записывает пачку логов
func (al *AsyncLogger) flushBatch(batch []LogEntry) {
	for _, entry := range batch {
		// Форматируем вывод
		jsonData, _ := json.Marshal(entry)
		al.writer.Println(string(jsonData))
	}
}

// Log отправляет запись в канал (неблокирующий)
func (al *AsyncLogger) Log(level, message string, fields ...interface{}) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
	}

	// Добавляем дополнительные поля если есть
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			key := fields[i].(string)
			value := fields[i+1]
			switch key {
			case "path":
				entry.Path = value.(string)
			case "method":
				entry.Method = value.(string)
			case "duration":
				entry.Duration = value.(int64)
			default:
				if entry.Data == nil {
					entry.Data = make(map[string]interface{})
				}
				entry.Data.(map[string]interface{})[key] = value
			}
		}
	}

	// Неблокирующая отправка
	select {
	case al.logChan <- entry:
	default:
		// Канал переполнен - логируем ошибку (но не блокируем хендлер)
		fmt.Fprintf(os.Stderr, "Логгер переполнен, запись потеряна: %s\n", message)
	}
}

// Close закрывает логгер
func (al *AsyncLogger) Close() {
	close(al.logChan)
	<-al.done
}

// Info записывает с уровнем info
func (al *AsyncLogger) Info(message string, fields ...interface{}) {
	al.Log("INFO", message, fields...)
}

// Error записывает с уровнем error
func (al *AsyncLogger) Error(message string, fields ...interface{}) {
	al.Log("ERROR", message, fields...)
}

// Debug записывает с уровнем debug
func (al *AsyncLogger) Debug(message string, fields ...interface{}) {
	al.Log("DEBUG", message, fields...)
}

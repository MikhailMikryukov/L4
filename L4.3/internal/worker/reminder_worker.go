package worker

import (
	"fmt"
	"sync"
	"time"
)

// ReminderTask - задача напоминания
type ReminderTask struct {
	ID       int
	Message  string
	NotifyAt time.Time
	UserID   string
}

// ReminderWorker - фоновый воркер для обработки напоминаний
type ReminderWorker struct {
	taskChan chan ReminderTask
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewReminderWorker создаёт новый воркер
func NewReminderWorker(bufferSize int) *ReminderWorker {
	return &ReminderWorker{
		taskChan: make(chan ReminderTask, bufferSize),
		stopChan: make(chan struct{}),
	}
}

// AddTask добавляет задачу в очередь
func (w *ReminderWorker) AddTask(task ReminderTask) bool {
	select {
	case w.taskChan <- task:
		fmt.Printf("Task %d added to queue\n", task.ID)
		return true
	default:
		fmt.Printf("Task %d rejected: queue is full\n", task.ID)
		return false
	}
}

// Start запускает воркера
func (w *ReminderWorker) Start() {
	w.wg.Add(1)
	go w.workerLoop()
}

// Stop останавливает воркера
func (w *ReminderWorker) Stop() {
	close(w.stopChan)
	w.wg.Wait()
}

// workerLoop - основной цикл воркера
func (w *ReminderWorker) workerLoop() {
	defer w.wg.Done()

	// Очередь ожидающих задач
	pendingTasks := make([]ReminderTask, 0)

	// Таймер для проверки ближайшей задачи
	var nextTimer *time.Timer
	var timerChan <-chan time.Time

	for {
		// Обновляем таймер на основе ближайшей задачи
		if len(pendingTasks) > 0 && nextTimer == nil {
			nextTime := w.getNextNotifyTime(pendingTasks)
			if !nextTime.IsZero() {
				waitDuration := time.Until(nextTime)
				fmt.Println(waitDuration)
				if waitDuration > 0 {
					nextTimer = time.NewTimer(waitDuration)
					timerChan = nextTimer.C
				}
			}
		}

		select {
		case <-w.stopChan:
			// Останавливаем воркера
			if nextTimer != nil {
				nextTimer.Stop()
			}
			fmt.Println("Worker stopped")
			return

		case newTask := <-w.taskChan:
			// Получаем новую задачу
			fmt.Printf("Worker received task: %d (notify at %s)\n",
				newTask.ID, newTask.NotifyAt.Format("15:04:05"))
			pendingTasks = append(pendingTasks, newTask)

			// Если это первая задача или она ближайшая - сбрасываем таймер
			if len(pendingTasks) == 1 || newTask.NotifyAt.Equal(w.getNextNotifyTime(pendingTasks)) {

				if nextTimer != nil {
					nextTimer.Stop()
					nextTimer = nil
				}
				// Таймер будет пересоздан на следующей итерации
			}

		case <-timerChan:
			// Время пришло - отправляем все напоминания, которые должны быть отправлены
			now := time.Now()
			toSend := make([]ReminderTask, 0)
			remaining := make([]ReminderTask, 0)
			for _, task := range pendingTasks {
				if task.NotifyAt.Before(now) || task.NotifyAt.Equal(now) {
					toSend = append(toSend, task)
				} else {
					remaining = append(remaining, task)
				}
			}

			// Отправляем напоминания
			for _, task := range toSend {
				w.sendReminder(task)
			}

			// Обновляем очередь
			pendingTasks = remaining

			// Сбрасываем таймер
			if nextTimer != nil {
				nextTimer.Stop()
				nextTimer = nil
			}
			timerChan = nil
		}
	}
}

// getNextNotifyTime возвращает ближайшее время уведомления
func (w *ReminderWorker) getNextNotifyTime(tasks []ReminderTask) time.Time {
	if len(tasks) == 0 {
		return time.Time{}
	}

	next := tasks[0].NotifyAt
	for _, task := range tasks {
		if task.NotifyAt.Before(next) {
			next = task.NotifyAt
		}
	}
	fmt.Println(next)
	return next
}

// sendReminder отправляет напоминание
func (w *ReminderWorker) sendReminder(task ReminderTask) {
	fmt.Printf("REMINDER [%d] to user %s: %s (scheduled at %s)\n",
		task.ID,
		task.UserID,
		task.Message,
		task.NotifyAt.Format("15:04:05"),
	)
}

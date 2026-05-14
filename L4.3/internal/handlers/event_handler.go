package handlers

import (
	"L4.3/internal/config"
	handlers "L4.3/internal/handlers/logger"
	"L4.3/internal/models"
	"L4.3/internal/repository/postgres"
	"L4.3/internal/worker"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

// EventHandler обработчик событий
type EventHandler struct {
	db     *postgres.DB
	worker *worker.ReminderWorker
	logger *handlers.AsyncLogger
}

type DeleteRequest struct {
	ID int `json:"id"`
}

// NewEventHandler создание нового обработчика
func NewEventHandler(db *postgres.DB, w *worker.ReminderWorker, cfg config.Config) *EventHandler {
	logger := handlers.NewAsyncLogger(cfg.LoggerBuffSize, cfg.LoggerBatchSize)

	return &EventHandler{
		db:     db,
		worker: w,
		logger: logger,
	}
}

// NewRouter настройка обработчика
func NewRouter(db *postgres.DB, w *worker.ReminderWorker, cfg *config.Config) *http.ServeMux {
	mux := http.NewServeMux()
	eventHandler := NewEventHandler(db, w, *cfg)

	mux.HandleFunc("/create_event", eventHandler.createEvent)
	mux.HandleFunc("/delete_event", eventHandler.deleteEvent)
	mux.HandleFunc("/update_event", eventHandler.updateEvent)
	mux.HandleFunc("/events", eventHandler.getAllEvents)
	mux.HandleFunc("/archived_events", eventHandler.getAllArchivedEvents)

	// Страница для проверки работы приложения
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/templates/index.html")
	})

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static"))))

	return mux
}

func (h *EventHandler) createEvent(w http.ResponseWriter, r *http.Request) {
	var logFields []interface{}
	logFields = append(logFields, "path", r.URL.Path, "method", r.Method)
	var errMsg string

	if r.Method != "POST" {
		h.logger.Error("wrong method", logFields...)

		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var event models.Event

	body, err := io.ReadAll(r.Body)
	if err != nil {
		errMsg = "error reading body " + err.Error()
		h.logger.Error(errMsg, logFields...)

		h.writeJSON(w, http.StatusBadRequest, map[string]string{
			"status":   "error",
			"response": errMsg,
		})
		return
	}

	err = json.Unmarshal(body, &event)
	if err != nil {
		errMsg = "invalid json " + err.Error()
		h.logger.Error(errMsg, logFields...)

		h.writeJSON(w, http.StatusBadRequest, map[string]string{
			"status":   "error",
			"response": errMsg,
		})
		return
	}

	id, err := h.db.SaveEvent(event)
	if err != nil {
		errMsg = "failed save to the db " + err.Error()
		h.logger.Error(errMsg, logFields...)

		h.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"status":   "error",
			"response": errMsg,
		})
		return
	}

	if event.ShouldNotify && !event.NotifyAt.IsZero() {
		task := worker.ReminderTask{
			ID:       id,
			Message:  event.Name,
			NotifyAt: event.NotifyAt,
			UserID:   event.UserID,
		}

		h.worker.AddTask(task)
	}

	logFields = append(logFields, "event_id", event.ID, "event_name", event.Name)
	h.logger.Info("event created", logFields...)

	h.writeJSON(w, http.StatusOK, map[string]string{
		"status": "success",
		"id":     strconv.Itoa(id),
	})
}

func (h *EventHandler) deleteEvent(w http.ResponseWriter, r *http.Request) {
	var logFields []interface{}
	logFields = append(logFields, "path", r.URL.Path, "method", r.Method)
	var errMsg string

	if r.Method != "DELETE" {
		h.logger.Error("wrong method", logFields...)

		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var dr DeleteRequest

	body, err := io.ReadAll(r.Body)
	if err != nil {
		errMsg = "error reading body " + err.Error()
		h.logger.Error(errMsg, logFields...)

		h.writeJSON(w, http.StatusBadRequest, map[string]string{
			"status":   "error",
			"response": errMsg,
		})
		return
	}

	err = json.Unmarshal(body, &dr)
	if err != nil {
		errMsg = "invalid json " + err.Error()
		h.logger.Error(errMsg, logFields...)

		h.writeJSON(w, http.StatusBadRequest, map[string]string{
			"status":   "error",
			"response": errMsg,
		})
		return
	}

	exists, err := h.db.Exists(dr.ID)
	if err != nil {
		errMsg = "failed to check event existence " + err.Error()
		h.logger.Error(errMsg, logFields...)

		h.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"status":   "error",
			"response": errMsg,
		})
		return
	}

	if !exists {
		errMsg = "no such event"
		h.logger.Error(errMsg, logFields...)

		h.writeJSON(w, http.StatusBadRequest, map[string]string{
			"status":   "error",
			"response": errMsg,
		})
		return
	}

	err = h.db.DeleteEvent(dr.ID)
	if err != nil {
		errMsg = "failed to delete event " + err.Error()
		h.logger.Error(errMsg, logFields...)

		h.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"status":   "error",
			"response": errMsg,
		})
		return
	}

	logFields = append(logFields, "event_id", dr.ID)
	h.logger.Info("event deleted", logFields...)

	h.writeJSON(w, http.StatusOK, map[string]string{
		"status":   "success",
		"response": "event deleted",
	})
}

func (h *EventHandler) updateEvent(w http.ResponseWriter, r *http.Request) {
	var logFields []interface{}
	logFields = append(logFields, "path", r.URL.Path, "method", r.Method)
	var errMsg string

	if r.Method != "PUT" {
		h.logger.Error("wrong method", logFields...)

		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var eventUpdated models.Event

	body, err := io.ReadAll(r.Body)
	if err != nil {
		errMsg = "error reading body " + err.Error()
		h.logger.Error(errMsg, logFields...)

		h.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"status":   "error",
			"response": errMsg,
		})
		return
	}

	err = json.Unmarshal(body, &eventUpdated)
	if err != nil {
		errMsg = "invalid json " + err.Error()
		h.logger.Error(errMsg, logFields...)

		h.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"status":   "error",
			"response": errMsg,
		})
		return
	}

	exists, err := h.db.Exists(eventUpdated.ID)
	if err != nil {
		errMsg = "failed to check event existence " + err.Error()
		h.logger.Error(errMsg, logFields...)

		h.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"status":   "error",
			"response": errMsg,
		})
		return
	}

	if !exists {
		errMsg = "no such event"
		h.logger.Error(errMsg, logFields...)

		h.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"status":   "error",
			"response": errMsg,
		})
		return
	}

	eventToUpdate, err := h.db.GetEventByID(eventUpdated.ID)
	if err != nil {
		errMsg = fmt.Sprintf("failed to get event %d %s", eventUpdated.ID, err.Error())
		h.logger.Error(errMsg, logFields...)

		h.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"status":   "error",
			"response": errMsg,
		})
		return
	}

	updates := make(map[string]interface{})

	if eventToUpdate.UserID != eventUpdated.UserID {
		updates["user_id"] = eventUpdated.UserID
	}

	if eventToUpdate.Name != eventUpdated.Name {
		updates["event_name"] = eventUpdated.Name
	}

	if eventToUpdate.Date != eventUpdated.Date {
		updates["event_date"] = eventUpdated.Date
	}

	if eventToUpdate.Description != eventUpdated.Description {
		updates["description"] = eventUpdated.Description
	}

	if eventToUpdate.ShouldNotify != eventUpdated.ShouldNotify {
		updates["should_notify"] = eventUpdated.ShouldNotify
	}

	if eventToUpdate.NotifyAt != eventUpdated.NotifyAt && !eventUpdated.NotifyAt.IsZero() {
		updates["notify_at"] = eventUpdated.NotifyAt
	}

	if eventToUpdate.Description != eventUpdated.Description {
		updates["description"] = eventUpdated.Description
	}

	err = h.db.UpdateEvent(eventToUpdate.ID, updates)
	if err != nil {
		errMsg = "failed to update event " + err.Error()
		h.logger.Error(errMsg, logFields...)

		h.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"status":   "error",
			"response": errMsg,
		})
		return
	}

	for k, v := range updates {
		logFields = append(logFields, "UPDATED "+k, v)
	}

	logFields = append(logFields, "event_id", eventToUpdate.ID)
	h.logger.Info("event updated", logFields...)

	h.writeJSON(w, http.StatusOK, map[string]string{
		"status":   "success",
		"response": "event updated"},
	)
}

func (h *EventHandler) getAllEvents(w http.ResponseWriter, r *http.Request) {
	var logFields []interface{}
	logFields = append(logFields, "path", r.URL.Path, "method", r.Method)
	var errMsg string

	if r.Method != "GET" {
		h.logger.Error("wrong method", logFields...)

		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	events, err := h.db.GetAllEvents()
	if err != nil {
		errMsg = "failed to get events " + err.Error()
		h.logger.Error(errMsg, logFields...)

		h.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"status":   "error",
			"response": errMsg,
		})
		return
	}

	h.writeJSON(w, http.StatusOK, events)
}

func (h *EventHandler) getAllArchivedEvents(w http.ResponseWriter, r *http.Request) {
	var logFields []interface{}
	logFields = append(logFields, "path", r.URL.Path, "method", r.Method)
	var errMsg string

	if r.Method != "GET" {
		h.logger.Error("wrong method", logFields...)

		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	events, err := h.db.GetAllArchivedEvents()
	if err != nil {
		errMsg = "failed to get events " + err.Error()
		h.logger.Error(errMsg, logFields...)

		h.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"status":   "error",
			"response": errMsg,
		})
		return
	}

	h.writeJSON(w, http.StatusOK, events)
}

func (h *EventHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

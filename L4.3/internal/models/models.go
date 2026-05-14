package models

import "time"

type Event struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Date         time.Time `json:"date"`
	UserID       string    `json:"user_id"`
	ShouldNotify bool      `json:"should_notify"`
	NotifyAt     time.Time `json:"notify_at"`
	Description  string    `json:"description"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ArchivedEvent struct {
	ID         int       `json:"id"`
	EventID    int       `json:"event_id"`
	Name       string    `json:"name"`
	Date       time.Time `json:"date"`
	UserID     string    `json:"user_id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	ArchivedAt time.Time `json:"archived_at"`
}

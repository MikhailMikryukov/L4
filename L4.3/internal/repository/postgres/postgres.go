package postgres

import (
	"L4.3/internal/models"
	"context"
	"database/sql"
	"fmt"
	"github.com/wb-go/wbf/dbpg"
	"log"
	"time"
)

// DB обертка над DB wbf
type DB struct {
	db *dbpg.DB
}

// New создание экземпляра
func New(conn string) (*DB, error) {
	opts := &dbpg.Options{MaxOpenConns: 10, MaxIdleConns: 5}
	db, err := dbpg.New(conn, nil, opts)
	if err != nil {
		return nil, err
	}

	// Создаем таблицу если нужно
	err = createTable(db)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &DB{
		db: db,
	}, nil
}

func (d *DB) SaveEvent(e models.Event) (int, error) {
	query := `INSERT INTO events (user_id, event_name, event_date, should_notify, notify_at, description) VALUES ($1,$2,$3, $4, $5,$6) RETURNING id`

	row := d.db.QueryRowContext(context.Background(), query, e.UserID, e.Name, e.Date, e.ShouldNotify, e.NotifyAt, e.Description)
	var id int
	err := row.Scan(&id)
	return id, err
}

func (d *DB) GetEventByID(id int) (models.Event, error) {
	query := `SELECT * FROM events WHERE id = $1`

	var e models.Event
	var shouldNotify sql.NullBool
	var notifyAt sql.NullTime
	row := d.db.QueryRowContext(context.Background(), query, id)
	err := row.Scan(&e.ID, &e.Name, &e.Date, &e.UserID, &e.ShouldNotify, &e.NotifyAt, &e.Description, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return models.Event{}, err
	}
	if shouldNotify.Valid {
		e.ShouldNotify = shouldNotify.Bool
	}
	if notifyAt.Valid {
		e.NotifyAt = notifyAt.Time
	}

	return e, nil
}

func (d *DB) UpdateEvent(id int, updated map[string]interface{}) error {
	if len(updated) == 0 {
		return nil
	}

	updated["updated_at"] = time.Now()

	query := `UPDATE events SET `

	var args []interface{}
	i := 1
	for field, value := range updated {
		query += fmt.Sprintf("%s = $%d, ", field, i)
		args = append(args, value)
		i++
	}

	// Убираем последнюю запятую и пробел
	query = query[:len(query)-2]

	query += fmt.Sprintf(" WHERE id = $%d", i)
	args = append(args, id)

	_, err := d.db.ExecContext(context.Background(), query, args...)
	return err
}

func (d *DB) DeleteEvent(id int) error {
	query := `DELETE FROM events WHERE id = $1`

	_, err := d.db.ExecContext(context.Background(), query, id)
	return err
}

func (d *DB) GetEventsForPeriod(dateFrom time.Time, dateTo time.Time) ([]models.Event, error) {
	query := `SELECT * FROM events WHERE event_date >= $1 AND event_date <= $2`

	rows, err := d.db.QueryContext(context.Background(), query, dateFrom, dateTo)
	if err != nil {
		return []models.Event{}, err
	}

	var events []models.Event
	for rows.Next() {
		var event models.Event
		err = rows.Scan(&event.ID, &event.UserID, &event.Name, &event.Date, &event.CreatedAt, &event.UpdatedAt)
		if err != nil {
			return []models.Event{}, err
		}
		events = append(events, event)
	}

	return events, nil
}

// Exists существует ли запись в бд
func (d *DB) Exists(id int) (bool, error) {
	var exists bool
	err := d.db.QueryRowContext(context.Background(), `SELECT EXISTS (SELECT 1 FROM events WHERE id = $1)`, id).Scan(&exists)
	return exists, err
}

func (d *DB) GetAllEvents() ([]models.Event, error) {
	query := `SELECT * FROM events`

	rows, err := d.db.QueryContext(context.Background(), query)
	if err != nil {
		return []models.Event{}, err
	}

	var events []models.Event
	for rows.Next() {
		var e models.Event
		var shouldNotify sql.NullBool
		var notifyAt sql.NullTime
		err = rows.Scan(&e.ID, &e.Name, &e.Date, &e.UserID, &shouldNotify, &notifyAt, &e.Description, &e.CreatedAt, &e.UpdatedAt)
		if err != nil {
			return []models.Event{}, err
		}
		if shouldNotify.Valid {
			e.ShouldNotify = shouldNotify.Bool
		}
		if notifyAt.Valid {
			e.NotifyAt = notifyAt.Time
		}
		events = append(events, e)
	}
	return events, nil
}

func (d *DB) GetAllArchivedEvents() ([]models.ArchivedEvent, error) {
	query := `SELECT * FROM archived_events`

	rows, err := d.db.QueryContext(context.Background(), query)
	if err != nil {
		return []models.ArchivedEvent{}, err
	}

	var events []models.ArchivedEvent
	for rows.Next() {
		var e models.ArchivedEvent
		err = rows.Scan(&e.ID, &e.EventID, &e.Name, &e.Date, &e.UserID, &e.CreatedAt, &e.UpdatedAt, &e.ArchivedAt)
		if err != nil {
			return []models.ArchivedEvent{}, err
		}
		events = append(events, e)
	}
	return events, nil
}

// BeginTx начинает транзакцию
func (d *DB) BeginTx() (*sql.Tx, error) {
	return d.db.BeginTx(context.Background(), &sql.TxOptions{})
}

func (d *DB) ArchiveOldEvents(tx *sql.Tx) error {
	insertQuery := `INSERT INTO archived_events (event_id, user_id, event_name, event_date, created_at, updated_at)
						SELECT id, user_id, event_name, event_date, created_at, updated_at
						FROM events
						WHERE event_date <= $1
						`
	_, err := tx.ExecContext(context.Background(), insertQuery, time.Now())
	if err != nil {
		return err
	}

	deleteQuery := `DELETE FROM events WHERE event_date <= $1`
	_, err = tx.ExecContext(context.Background(), deleteQuery, time.Now())
	if err != nil {
		return err
	}

	return nil
}

func createTable(db *dbpg.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS events (
    		id SERIAL PRIMARY KEY,
    		event_name VARCHAR(255) NOT NULL,
			event_date TIMESTAMPTZ NOT NULL,
    		user_id VARCHAR(36) NOT NULL,
    		should_notify BOOLEAN,
    		notify_at TIMESTAMPTZ,
    		description TEXT,
    		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    		updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
    		)`,
		`CREATE TABLE IF NOT EXISTS archived_events (
    		id SERIAL PRIMARY KEY,
    		event_id INT,
    		event_name VARCHAR(255) NOT NULL,
    		event_date TIMESTAMPTZ NOT NULL,
    		user_id VARCHAR(36) NOT NULL,
    		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    		updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    		archived_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
    		)`,
	}

	for _, q := range queries {
		_, err := db.ExecContext(context.Background(), q)
		if err != nil {
			return err
		}
	}

	return nil
}

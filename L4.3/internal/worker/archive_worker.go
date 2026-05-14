package worker

import (
	"L4.3/internal/repository/postgres"
	"log"
	"sync"
	"time"
)

type ArchiveWorker struct {
	ticker *time.Ticker
	db     *postgres.DB
	wg     sync.WaitGroup
}

func NewArchiveWorker(cleanupInterval time.Duration, db *postgres.DB) *ArchiveWorker {
	ticker := time.NewTicker(cleanupInterval)
	return &ArchiveWorker{
		ticker: ticker,
		db:     db,
	}
}

func (a *ArchiveWorker) Start() {
	a.wg.Add(1)

	go func() {
		for {
			select {
			case <-a.ticker.C:
				tx, err := a.db.BeginTx()
				if err != nil {
					return
				}
				err = a.db.ArchiveOldEvents(tx)
				if err != nil {
					if rollbackErr := tx.Rollback(); rollbackErr != nil {
						log.Fatalf("update drivers: unable to rollback: %v", rollbackErr)
					}
					log.Fatal(err)
				}

				if err := tx.Commit(); err != nil {
					log.Fatal(err)
				}
			}
		}
	}()
}

func (a *ArchiveWorker) Stop() {
	a.wg.Done()
	a.ticker.Stop()
}

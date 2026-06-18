package logger

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

type ClickLog struct {
	IP         string
	CampaignID string
	UserAgent  string
	IsBot      bool
	Reason     string
}

type BatchLogger struct {
	db            *sql.DB
	logChan       chan ClickLog
	batchSize     int
	flushInterval time.Duration
}

func NewBatchLogger(db *sql.DB, batchSize int, flushIntervalMs int) *BatchLogger {
	return &BatchLogger{
		db:            db,
		logChan:       make(chan ClickLog, batchSize*2),
		batchSize:     batchSize,
		flushInterval: time.Duration(flushIntervalMs) * time.Millisecond,
	}
}

func (bl *BatchLogger) LogAsync(entry ClickLog) {
	select {
	case bl.logChan <- entry:
	default:
		log.Println("[BatchLogger] Warning: log channel full, dropping log entry")
	}
}

func (bl *BatchLogger) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(bl.flushInterval)
		defer ticker.Stop()

		buffer := make([]ClickLog, 0, bl.batchSize)

		for {
			select {
			case entry := <-bl.logChan:
				buffer = append(buffer, entry)
				if len(buffer) >= bl.batchSize {
					bl.flush(buffer)
					buffer = buffer[:0]
					ticker.Reset(bl.flushInterval)
				}
			case <-ticker.C:
				if len(buffer) > 0 {
					bl.flush(buffer)
					buffer = buffer[:0]
				}
			case <-ctx.Done():
				if len(buffer) > 0 {
					bl.flush(buffer)
				}
				return
			}
		}
	}()
}

func (bl *BatchLogger) flush(batch []ClickLog) {
	if len(batch) == 0 {
		return
	}

	valueStrings := make([]string, 0, len(batch))
	valueArgs := make([]interface{}, 0, len(batch)*5)

	for i, entry := range batch {
		n := i * 5
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d)", n+1, n+2, n+3, n+4, n+5))
		valueArgs = append(valueArgs, entry.IP)
		valueArgs = append(valueArgs, entry.CampaignID)
		valueArgs = append(valueArgs, entry.UserAgent)
		valueArgs = append(valueArgs, entry.IsBot)
		valueArgs = append(valueArgs, entry.Reason)
	}

	stmt := fmt.Sprintf("INSERT INTO click_logs (ip, campaign_id, user_agent, is_bot, reason) VALUES %s", strings.Join(valueStrings, ","))

	_, err := bl.db.Exec(stmt, valueArgs...)
	if err != nil {
		log.Printf("[BatchLogger] Error flushing batch of %d items: %v", len(batch), err)
		return
	}

	log.Printf("[BatchLogger] Successfully flushed batch of %d logs to Postgres", len(batch))
}

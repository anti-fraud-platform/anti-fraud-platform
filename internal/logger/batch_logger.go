package logger

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
	"net"
    "github.com/oschwald/geoip2-golang"
)

type ClickLog struct {
    IP          string
    CampaignID  string
    UserAgent   string
    IsBot       bool
    Reason      string
    Country     string   // Tier 2
    RiskScore   int      // Tier 2
    RiskReasons string   // Tier 2 (comma-separated)
}

type BatchLogger struct {
    db            *sql.DB
    logChan       chan ClickLog
    batchSize     int
    flushInterval time.Duration
    geoReader     *geoip2.Reader   // Tier 2
}

func NewBatchLogger(db *sql.DB, batchSize int, flushIntervalMs int) *BatchLogger {
    bl := &BatchLogger{
        db:            db,
        logChan:       make(chan ClickLog, batchSize*2),
        batchSize:     batchSize,
        flushInterval: time.Duration(flushIntervalMs) * time.Millisecond,
    }

    // Tier 2: load GeoIP database (adjust path as needed)
    if reader, err := geoip2.Open("/usr/share/GeoIP/GeoLite2-Country.mmdb"); err != nil {
        log.Printf("Warning: GeoIP database not loaded: %v", err)
    } else {
        bl.geoReader = reader
    }

    return bl
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

    // Now we have 8 fields: ip, campaign_id, user_agent, is_bot, reason, country, risk_score, risk_reasons
    valueStrings := make([]string, 0, len(batch))
    valueArgs := make([]interface{}, 0, len(batch)*8)

    for i, entry := range batch {
        n := i * 8
        valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
            n+1, n+2, n+3, n+4, n+5, n+6, n+7, n+8))

        valueArgs = append(valueArgs, entry.IP)
        valueArgs = append(valueArgs, entry.CampaignID)
        valueArgs = append(valueArgs, entry.UserAgent)
        valueArgs = append(valueArgs, entry.IsBot)
        valueArgs = append(valueArgs, entry.Reason)

        // Resolve country from IP (if geoReader available)
        country := entry.Country // if already set, use it; otherwise try to resolve
        if country == "" && bl.geoReader != nil {
            ip := net.ParseIP(entry.IP)
            if ip != nil {
                if record, err := bl.geoReader.Country(ip); err == nil && record != nil {
                    country = record.Country.IsoCode
                }
            }
        }
        valueArgs = append(valueArgs, country)

        valueArgs = append(valueArgs, entry.RiskScore)
        valueArgs = append(valueArgs, entry.RiskReasons)
    }

    stmt := fmt.Sprintf(`
        INSERT INTO click_logs 
        (ip, campaign_id, user_agent, is_bot, reason, country, risk_score, risk_reasons) 
        VALUES %s
    `, strings.Join(valueStrings, ","))

    _, err := bl.db.Exec(stmt, valueArgs...)
    if err != nil {
        log.Printf("[BatchLogger] Error flushing batch of %d items: %v", len(batch), err)
        return
    }

    log.Printf("[BatchLogger] Successfully flushed batch of %d logs to Postgres", len(batch))
}

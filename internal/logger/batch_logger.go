package logger

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"anti-fraud/internal/geoiputil"
)

type ClickLog struct {
	IP          string
	CampaignID  string
	UserAgent   string
	IsBot       bool
	Reason      string
	Country     string
	City        string
	ASNNumber   uint
	ASNOrg      string
	RiskScore   int
	RiskReasons string
}

type BatchLogger struct {
	db            *sql.DB
	logChan       chan ClickLog
	batchSize     int
	flushInterval time.Duration
	geoResolver   *geoiputil.Resolver
}

func NewBatchLogger(db *sql.DB, batchSize int, flushIntervalMs int) *BatchLogger {
	return NewBatchLoggerWithResolver(db, batchSize, flushIntervalMs, nil)
}

func NewBatchLoggerWithResolver(db *sql.DB, batchSize int, flushIntervalMs int, resolver *geoiputil.Resolver) *BatchLogger {
	bl := &BatchLogger{
		db:            db,
		logChan:       make(chan ClickLog, batchSize*2),
		batchSize:     batchSize,
		flushInterval: time.Duration(flushIntervalMs) * time.Millisecond,
	}

	if resolver != nil {
		bl.geoResolver = resolver
		return bl
	}

	resolver, errs := geoiputil.OpenBestEffort(geoiputil.PathsFromEnv())
	for _, err := range errs {
		log.Printf("Warning: GeoIP database not loaded: %v", err)
	}
	bl.geoResolver = resolver

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

	// Persist request, GeoIP enrichment, and risk-scoring metadata in one batch.
	valueStrings := make([]string, 0, len(batch))
	valueArgs := make([]interface{}, 0, len(batch)*11)

	for i, entry := range batch {
		n := i * 11
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			n+1, n+2, n+3, n+4, n+5, n+6, n+7, n+8, n+9, n+10, n+11))

		valueArgs = append(valueArgs, entry.IP)
		valueArgs = append(valueArgs, entry.CampaignID)
		valueArgs = append(valueArgs, entry.UserAgent)
		valueArgs = append(valueArgs, entry.IsBot)
		valueArgs = append(valueArgs, entry.Reason)

		country := entry.Country
		city := entry.City
		asnNumber := entry.ASNNumber
		asnOrg := entry.ASNOrg

		if bl.geoResolver != nil {
			ip := net.ParseIP(entry.IP)
			if ip != nil {
				lookup := bl.geoResolver.Lookup(ip)
				if country == "" {
					country = lookup.CountryISO
				}
				if city == "" {
					city = lookup.CityName
				}
				if asnNumber == 0 {
					asnNumber = lookup.ASNNumber
				}
				if asnOrg == "" {
					asnOrg = lookup.ASNOrg
				}
			}
		}

		valueArgs = append(valueArgs, country)
		valueArgs = append(valueArgs, city)
		valueArgs = append(valueArgs, int64(asnNumber))
		valueArgs = append(valueArgs, asnOrg)

		valueArgs = append(valueArgs, entry.RiskScore)
		valueArgs = append(valueArgs, entry.RiskReasons)
	}

	stmt := fmt.Sprintf(`
        INSERT INTO click_logs 
        (ip, campaign_id, user_agent, is_bot, reason, country, city, asn_number, asn_org, risk_score, risk_reasons)
        VALUES %s
    `, strings.Join(valueStrings, ","))

	_, err := bl.db.Exec(stmt, valueArgs...)
	if err != nil {
		log.Printf("[BatchLogger] Error flushing batch of %d items: %v", len(batch), err)
		return
	}

	log.Printf("[BatchLogger] Successfully flushed batch of %d logs to Postgres", len(batch))
}

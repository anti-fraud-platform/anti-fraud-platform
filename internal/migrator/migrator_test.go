package migrator

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

func setupMigratorTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "antifraud"
	}
	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "antifraud123"
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "analytics"
	}

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Skipf("skipping migrator test: database not available: %v", err)
		return nil
	}
	if err := db.Ping(); err != nil {
		t.Skipf("skipping migrator test: database not reachable: %v", err)
		return nil
	}
	return db
}

func TestUpIsIdempotentAndRecordsVersions(t *testing.T) {
	db := setupMigratorTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	baseVersion := 900000000 + int(time.Now().Unix()%100000)
	version1 := baseVersion
	version2 := baseVersion + 1
	tableName := fmt.Sprintf("migrator_test_%d", time.Now().UnixNano()%1000000)

	_, _ = db.Exec("DELETE FROM schema_migrations WHERE version IN ($1, $2)", version1, version2)
	_, _ = db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))
	defer db.Exec("DELETE FROM schema_migrations WHERE version IN ($1, $2)", version1, version2)
	defer db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))

	migrationsDir := t.TempDir()
	migration1 := filepath.Join(migrationsDir, fmt.Sprintf("%d_create_%s.sql", version1, tableName))
	migration2 := filepath.Join(migrationsDir, fmt.Sprintf("%d_seed_%s.sql", version2, tableName))

	if err := os.WriteFile(migration1, []byte(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id INT PRIMARY KEY
		);
	`, tableName)), 0o644); err != nil {
		t.Fatalf("failed to write migration1: %v", err)
	}

	if err := os.WriteFile(migration2, []byte(fmt.Sprintf(`
		INSERT INTO %s (id) VALUES (1)
		ON CONFLICT (id) DO NOTHING;
	`, tableName)), 0o644); err != nil {
		t.Fatalf("failed to write migration2: %v", err)
	}

	mig := New(db, migrationsDir)

	if err := mig.Up(); err != nil {
		t.Fatalf("first Up() failed: %v", err)
	}

	var appliedCount int
	if err := db.QueryRow(
		"SELECT COUNT(*) FROM schema_migrations WHERE version IN ($1, $2)",
		version1, version2,
	).Scan(&appliedCount); err != nil {
		t.Fatalf("failed to count applied versions after first Up(): %v", err)
	}
	if appliedCount != 2 {
		t.Fatalf("expected 2 applied versions after first Up(), got %d", appliedCount)
	}

	var rowCount int
	if err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&rowCount); err != nil {
		t.Fatalf("failed to query seeded rows after first Up(): %v", err)
	}
	if rowCount != 1 {
		t.Fatalf("expected 1 seeded row after first Up(), got %d", rowCount)
	}

	if err := mig.Up(); err != nil {
		t.Fatalf("second Up() failed: %v", err)
	}

	if err := db.QueryRow(
		"SELECT COUNT(*) FROM schema_migrations WHERE version IN ($1, $2)",
		version1, version2,
	).Scan(&appliedCount); err != nil {
		t.Fatalf("failed to count applied versions after second Up(): %v", err)
	}
	if appliedCount != 2 {
		t.Fatalf("expected 2 applied versions after second Up(), got %d", appliedCount)
	}

	if err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&rowCount); err != nil {
		t.Fatalf("failed to query seeded rows after second Up(): %v", err)
	}
	if rowCount != 1 {
		t.Fatalf("expected 1 seeded row after second Up(), got %d", rowCount)
	}
}

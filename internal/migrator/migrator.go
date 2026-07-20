package migrator

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type Migrator struct {
	db            *sql.DB
	migrationsDir string
}

func New(db *sql.DB, migrationsDir string) *Migrator {
	return &Migrator{
		db:            db,
		migrationsDir: migrationsDir,
	}
}

func (m *Migrator) ensureMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INT PRIMARY KEY,
			applied_at TIMESTAMPTZ DEFAULT NOW()
		);
	`
	_, err := m.db.Exec(query)
	return err
}

func (m *Migrator) getAppliedVersions() (map[int]bool, error) {
	rows, err := m.db.Query("SELECT version FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	versions := make(map[int]bool)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		versions[version] = true
	}
	return versions, nil
}

func (m *Migrator) Up() error {
	if err := m.ensureMigrationsTable(); err != nil {
		return fmt.Errorf("failed to ensure migrations table: %w", err)
	}

	applied, err := m.getAppliedVersions()
	if err != nil {
		return fmt.Errorf("failed to get applied versions: %w", err)
	}

	files, err := os.ReadDir(m.migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrationFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			migrationFiles = append(migrationFiles, file.Name())
		}
	}
	sort.Strings(migrationFiles)

	appliedCount := 0
	for _, filename := range migrationFiles {
		parts := strings.SplitN(filename, "_", 2)
		if len(parts) < 2 {
			continue
		}

		version, err := strconv.Atoi(parts[0])
		if err != nil {
			return fmt.Errorf("invalid migration version in filename %s: %w", filename, err)
		}

		if applied[version] {
			continue
		}

		content, err := os.ReadFile(filepath.Join(m.migrationsDir, filename))
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		fmt.Printf("Applying migration %d: %s...\n", version, filename)

		tx, err := m.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for %s: %w", filename, err)
		}

		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", filename, err)
		}

		if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", filename, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", filename, err)
		}

		appliedCount++
	}

	if appliedCount == 0 {
		fmt.Println("Database is up to date. No new migrations to apply.")
	} else {
		fmt.Printf("Successfully applied %d migration(s).\n", appliedCount)
	}

	return nil
}

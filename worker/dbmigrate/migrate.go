package dbmigrate

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
)

// RunMigrations reads SQL migration files from the given directory
// and executes them in numeric order against the database.
// Migration files must match the pattern NNN-*.sql (e.g., 001-base.sql).
func RunMigrations(db *sql.DB, migrationsDir string) error {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir %s: %w", migrationsDir, err)
	}

	pat := regexp.MustCompile(`^(\d{3})-.*\.sql$`)
	var migrations []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if pat.MatchString(e.Name()) {
			migrations = append(migrations, e.Name())
		}
	}
	sort.Strings(migrations)

	// Check which migrations have already been executed
	executed := make(map[int]bool)
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='migrations'").Scan(&tableName)
	switch {
	case err == nil:
		rows, err := db.Query("SELECT migration_number FROM migrations")
		if err != nil {
			return fmt.Errorf("query executed migrations: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var n int
			if err := rows.Scan(&n); err != nil {
				return fmt.Errorf("scan migration number: %w", err)
			}
			executed[n] = true
		}
	case errors.Is(err, sql.ErrNoRows):
		slog.Info("db: migrations table not found; running all migrations")
	default:
		return fmt.Errorf("check migrations table: %w", err)
	}

	applied := 0
	for _, m := range migrations {
		match := pat.FindStringSubmatch(m)
		if len(match) != 2 {
			return fmt.Errorf("invalid migration filename: %s", m)
		}
		n, err := strconv.Atoi(match[1])
		if err != nil {
			return fmt.Errorf("parse migration number %s: %w", m, err)
		}
		if executed[n] {
			continue
		}

		content, err := os.ReadFile(filepath.Join(migrationsDir, m))
		if err != nil {
			return fmt.Errorf("read %s: %w", m, err)
		}
		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("exec %s: %w", m, err)
		}
		applied++
		slog.Info("db: applied migration", "file", m, "number", n)
	}

	if applied > 0 {
		slog.Info("db: migrations complete", "applied", applied, "total", len(migrations))
	} else {
		slog.Info("db: all migrations already applied", "total", len(migrations))
	}

	return nil
}

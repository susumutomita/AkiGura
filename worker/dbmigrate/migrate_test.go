package dbmigrate

import (
	"database/sql"
	"os"
	"testing"

	_ "modernc.org/sqlite"
)

const migrationsDir = "../../control-plane/db/migrations"

func TestRunMigrations(t *testing.T) {
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		t.Skip("migrations directory not found (expected in monorepo layout)")
	}

	t.Run("空のデータベースに全マイグレーションを適用すべき", func(t *testing.T) {
		db, err := sql.Open("sqlite", ":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		if err := RunMigrations(db, migrationsDir); err != nil {
			t.Fatalf("RunMigrations failed: %v", err)
		}

		// Verify migrations table exists and has entries
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM migrations").Scan(&count); err != nil {
			t.Fatalf("query migrations count: %v", err)
		}
		if count == 0 {
			t.Error("migrations table should have entries after running migrations")
		}

		// Verify key tables exist
		for _, table := range []string{"municipalities", "grounds", "slots", "scrape_jobs"} {
			var name string
			err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
			if err != nil {
				t.Errorf("table %s should exist after migrations: %v", table, err)
			}
		}
	})

	t.Run("既に適用済みのマイグレーションをスキップすべき", func(t *testing.T) {
		db, err := sql.Open("sqlite", ":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		// Run twice -- second run should be a no-op
		if err := RunMigrations(db, migrationsDir); err != nil {
			t.Fatalf("first RunMigrations failed: %v", err)
		}
		if err := RunMigrations(db, migrationsDir); err != nil {
			t.Fatalf("second RunMigrations (idempotent) failed: %v", err)
		}
	})

	t.Run("存在しないディレクトリでエラーを返すべき", func(t *testing.T) {
		db, err := sql.Open("sqlite", ":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		err = RunMigrations(db, "/nonexistent/path")
		if err == nil {
			t.Error("RunMigrations should fail for nonexistent directory")
		}
	})
}

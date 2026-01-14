package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"akigura.dev/worker"
	"akigura.dev/worker/notifier"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

var (
	flagDBPath         = flag.String("db", "../control-plane/db.sqlite3", "database path (for local SQLite)")
	flagScraperPath    = flag.String("scraper", "./scraper_wrapper.py", "scraper wrapper path")
	flagPythonPath     = flag.String("python", "python3", "python interpreter path")
	flagInterval       = flag.Duration("interval", 15*time.Minute, "scrape interval")
	flagJobInterval    = flag.Duration("job-interval", 30*time.Second, "pending job check interval")
	flagNotifyInterval = flag.Duration("notify-interval", 1*time.Minute, "notification check interval")
	flagOnce           = flag.Bool("once", false, "run once and exit")
	flagNotifyOnly     = flag.Bool("notify-only", false, "only process notifications")
	flagJobMode        = flag.Bool("job-mode", false, "process pending jobs from database")
)

func main() {
	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	db, err := openDB()
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cancel()
	}()

	sender := notifier.NewSender(db)

	if *flagNotifyOnly {
		// Only send pending notifications
		sent, failed, err := sender.ProcessPending(ctx)
		fmt.Printf("Notifications: sent=%d, failed=%d\n", sent, failed)
		return err
	}

	w := worker.NewWorker(db, *flagScraperPath, *flagPythonPath)

	if *flagOnce {
		// Run scraper once for all municipalities
		if err := w.ProcessAllFacilities(ctx); err != nil {
			return err
		}
		// Then send notifications
		sent, failed, _ := sender.ProcessPending(ctx)
		fmt.Printf("Notifications: sent=%d, failed=%d\n", sent, failed)
		return nil
	}

	if *flagJobMode {
		// Job mode: process pending jobs from database
		slog.Info("starting job processor", "job_interval", *flagJobInterval, "notify_interval", *flagNotifyInterval)

		// Start notification sender in background
		go sender.StartSender(ctx, *flagNotifyInterval)

		// Process pending jobs periodically
		ticker := time.NewTicker(*flagJobInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				slog.Info("job processor stopped")
				return nil
			case <-ticker.C:
				if err := w.ProcessPendingJobs(ctx); err != nil {
					slog.Error("failed to process pending jobs", "error", err)
				}
			}
		}
	}

	// Default: Start notification sender in background
	go sender.StartSender(ctx, *flagNotifyInterval)

	// Run scraper scheduler (blocks)
	w.StartScheduler(ctx, *flagInterval)
	return nil
}

// openDB opens a database connection.
// If TURSO_DATABASE_URL is set, connects to Turso.
// Otherwise, opens a local SQLite file.
func openDB() (*sql.DB, error) {
	tursoURL := os.Getenv("TURSO_DATABASE_URL")
	tursoToken := os.Getenv("TURSO_AUTH_TOKEN")

	if tursoURL != "" {
		return openTurso(tursoURL, tursoToken)
	}
	return openLocalSQLite(*flagDBPath)
}

func openTurso(url, token string) (*sql.DB, error) {
	connStr := url
	if token != "" {
		if strings.Contains(url, "?") {
			connStr = url + "&authToken=" + token
		} else {
			connStr = url + "?authToken=" + token
		}
	}

	db, err := sql.Open("libsql", connStr)
	if err != nil {
		return nil, fmt.Errorf("open turso: %w", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping turso: %w", err)
	}

	slog.Info("db: connected to Turso", "url", url)
	return db, nil
}

func openLocalSQLite(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON;"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	if _, err := db.Exec("PRAGMA journal_mode=wal;"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set WAL: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=1000;"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set busy_timeout: %w", err)
	}
	slog.Info("db: connected to local SQLite", "path", path)
	return db, nil
}

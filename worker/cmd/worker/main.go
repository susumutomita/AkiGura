package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"akigura.dev/worker"
	"akigura.dev/worker/notifier"
	_ "modernc.org/sqlite"
)

var (
	flagDBPath       = flag.String("db", "../control-plane/db.sqlite3", "database path")
	flagScraperPath  = flag.String("scraper", "./scraper_wrapper.py", "scraper wrapper path")
	flagPythonPath   = flag.String("python", "python3", "python interpreter path")
	flagInterval     = flag.Duration("interval", 15*time.Minute, "scrape interval")
	flagNotifyInterval = flag.Duration("notify-interval", 1*time.Minute, "notification check interval")
	flagOnce         = flag.Bool("once", false, "run once and exit")
	flagNotifyOnly   = flag.Bool("notify-only", false, "only process notifications")
)

func main() {
	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	db, err := sql.Open("sqlite", *flagDBPath)
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
		// Run scraper once
		if err := w.ProcessAllFacilities(ctx); err != nil {
			return err
		}
		// Then send notifications
		sent, failed, _ := sender.ProcessPending(ctx)
		fmt.Printf("Notifications: sent=%d, failed=%d\n", sent, failed)
		return nil
	}

	// Start notification sender in background
	go sender.StartSender(ctx, *flagNotifyInterval)

	// Run scraper scheduler (blocks)
	w.StartScheduler(ctx, *flagInterval)
	return nil
}

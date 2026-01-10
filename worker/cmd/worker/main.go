package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"akigura.dev/worker"
	_ "modernc.org/sqlite"
	"database/sql"
)

var (
	flagDBPath      = flag.String("db", "../control-plane/db.sqlite3", "database path")
	flagScraperPath = flag.String("scraper", "./scraper_wrapper.py", "scraper wrapper path")
	flagInterval    = flag.Duration("interval", 15*time.Minute, "scrape interval")
	flagOnce        = flag.Bool("once", false, "run once and exit")
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

	w := worker.NewWorker(db, *flagScraperPath, "python3")

	if *flagOnce {
		// Run once and exit
		return w.ProcessAllFacilities(context.Background())
	}

	// Run scheduler
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		cancel()
	}()

	w.StartScheduler(ctx, *flagInterval)
	return nil
}

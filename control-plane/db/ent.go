package db

import (
	"context"
	"fmt"
	"log/slog"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"srv.exe.dev/ent"
)

// OpenEnt opens an Ent client connection.
// If TURSO_DATABASE_URL is set, connects to Turso.
// Otherwise, opens a local SQLite file at path.
func OpenEnt(ctx context.Context, path string) (*ent.Client, error) {
	// Use existing Open function to get sql.DB
	db, err := Open(path)
	if err != nil {
		return nil, err
	}

	// Create Ent driver
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := ent.NewClient(ent.Driver(drv))

	// Auto-migrate schema
	if err := client.Schema.Create(ctx); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	slog.Info("db: ent client initialized")
	return client, nil
}

package pg

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

// Pg uses the pgx driver to create a new postgres connection using the provided connection string.
func Pg(ctx context.Context, connString string) (*sql.DB, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}
	db, err := sql.Open("pgx", stdlib.RegisterConnConfig(config.ConnConfig))
	if err != nil {
		return nil, fmt.Errorf("failed to create database pool: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	return db, nil
}

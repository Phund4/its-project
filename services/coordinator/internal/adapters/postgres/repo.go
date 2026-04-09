package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Repo struct {
	db *sql.DB
}

func New(ctx context.Context, dsn string, maxOpenConns, maxIdleConns int, connMaxIdleTime time.Duration) (*Repo, error) {
	db, err := sql.Open(postgresDriver, dsn)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToOpenDB, err)
	}

	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxIdleTime(connMaxIdleTime)

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToPingDB, err)
	}

	return &Repo{db: db}, nil
}

func (r *Repo) Close() error {
	if err := r.db.Close(); err != nil {
		return fmt.Errorf("%w: %w", ErrFailedToCloseDB, err)
	}
	return nil
}

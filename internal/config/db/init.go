package db

import (
	"context"
	"fmt"
	"log"

	"github.com/RoGogDBD/wb/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	var pool *pgxpool.Pool

	if err := config.GetRetryIntervals(ctx, func() error {
		var err error
		pool, err = pgxpool.New(ctx, dsn)
		if err != nil {
			return err
		}
		if err := pool.Ping(ctx); err != nil {
			pool.Close()
			return err
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to connect to DB: %w", err)
	}

	log.Println("Connected to PostgreSQL")

	if err := config.GetRetryIntervals(ctx, func() error {
		return RunMigrations(dsn)
	}); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to run migrations after retries: %w", err)
	}

	return pool, nil
}

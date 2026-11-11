package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dsn := os.Getenv("DB_URL")
	if dsn == "" {
		slog.Error("get dsn", "error", "DB_URL is required")
		os.Exit(1)
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		slog.Error("pool connection", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	dbCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	if err = pool.Ping(dbCtx); err != nil {
		slog.Error("db ping", "error", err)
		os.Exit(1)
	}

	slog.Info("pool succesfully connected")
}

package database

import (
	"context"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ConnectDB() (*pgxpool.Pool, error) {

	// 🔥 MUST be set in Render environment
	dsn := os.Getenv("DATABASE_URL")

	if dsn == "" {
		panic("DATABASE_URL is not set")
	}

	// Create connection pool (SAFE + SIMPLE for Render)
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, err
	}

	// Test connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	// Connection pool tuning (production-safe)
	config := pool.Config()
	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnIdleTime = 5 * time.Minute

	return pool, nil
}
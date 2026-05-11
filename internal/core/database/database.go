package database

import (
	"context"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ConnectDB() (*pgxpool.Pool, error) {

	// 🔥 MUST come from Render env
	dsn := os.Getenv("DATABASE_URL")

	// Safety check (helps debugging)
	if dsn == "" {
		panic("DATABASE_URL is not set")
	}

	// Create pool (SIMPLE + RELIABLE for Render)
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, err
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	// Optional tuning (production safe defaults)
	pool.Config().MaxConns = 10
	pool.Config().MinConns = 2
	pool.Config().MaxConnIdleTime = 5 * time.Minute

	return pool, nil
}
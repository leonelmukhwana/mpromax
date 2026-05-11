package database

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Add 'url string' here so main.go can pass the Render URL in
func ConnectDB(url string) (*pgxpool.Pool, error) {
	// 1. Use the url passed from main.go
	if url == "" {
		return nil, fmt.Errorf("database connection string is empty")
	}

	// 2. Parse the config
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("unable to parse config: %v", err)
	}

	// 3. Apply your tuning
	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnIdleTime = 5 * time.Minute

	// 4. Create the pool
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("unable to create pool: %v", err)
	}

	// 5. Test connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("database ping failed: %v", err)
	}

	return pool, nil
}
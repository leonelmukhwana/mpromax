package services

import (
	"context"
	"log"
	"rest_api/internal/repository"
	"time"
)

func StartCleanupCron(repo repository.AuthRepository) {
	// Run this in a separate goroutine
	go func() {
		// Check every 24 hours
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				err := repo.ClearExpiredBlacklist(ctx)
				if err != nil {
					log.Printf("Cron Error: Failed to clear blacklist: %v", err)
				} else {
					log.Println("Cron Success: Expired tokens cleared from blacklist")
				}
				cancel()
			}
		}
	}()
}

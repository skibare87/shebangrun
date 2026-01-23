package jobs

import (
	"database/sql"
	"log"
	"time"
)

// CheckExpiredSubscriptions downgrades users with expired subscriptions to Free tier
func CheckExpiredSubscriptions(db *sql.DB) {
	log.Println("Checking for expired subscriptions...")
	
	// Find users with expired subscriptions (not managed by admin)
	result, err := db.Exec(`
		UPDATE users 
		SET tier_id = 1, subscription_expires_at = NULL
		WHERE subscription_expires_at IS NOT NULL 
		  AND subscription_expires_at < NOW()
		  AND subscription_managed_by != 'admin'
		  AND tier_id != 1
	`)
	
	if err != nil {
		log.Printf("Error checking expired subscriptions: %v", err)
		return
	}
	
	rows, _ := result.RowsAffected()
	if rows > 0 {
		log.Printf("Downgraded %d users to Free tier due to expired subscriptions", rows)
	}
}

// StartSubscriptionChecker runs the expiration check hourly
func StartSubscriptionChecker(db *sql.DB) {
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		// Run immediately on startup
		CheckExpiredSubscriptions(db)
		
		// Then run hourly
		for range ticker.C {
			CheckExpiredSubscriptions(db)
		}
	}()
	
	log.Println("Subscription checker started (runs hourly)")
}

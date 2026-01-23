package database

import (
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RunMigrations executes all SQL migration files in order
func RunMigrations(db *sql.DB, migrationsPath string) error {
	// Create migrations tracking table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %v", err)
	}
	
	// Get list of migration files
	files, err := os.ReadDir(migrationsPath)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %v", err)
	}
	
	// Sort files
	var migrations []fs.DirEntry
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".sql") {
			migrations = append(migrations, f)
		}
	}
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Name() < migrations[j].Name()
	})
	
	// Apply each migration
	for _, migration := range migrations {
		version := strings.TrimSuffix(migration.Name(), ".sql")
		
		// Check if already applied
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", version).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check migration status: %v", err)
		}
		
		if count > 0 {
			log.Printf("Migration %s already applied, skipping", version)
			continue
		}
		
		// Read migration file
		content, err := os.ReadFile(filepath.Join(migrationsPath, migration.Name()))
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %v", migration.Name(), err)
		}
		
		// Execute migration
		log.Printf("Applying migration: %s", version)
		_, err = db.Exec(string(content))
		if err != nil {
			return fmt.Errorf("failed to apply migration %s: %v", version, err)
		}
		
		// Mark as applied
		_, err = db.Exec("INSERT INTO schema_migrations (version) VALUES (?)", version)
		if err != nil {
			return fmt.Errorf("failed to mark migration as applied: %v", err)
		}
		
		log.Printf("Migration %s applied successfully", version)
	}
	
	return nil
}

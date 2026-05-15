package db

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

var (
	DB       *sql.DB
	mu       sync.RWMutex
	isSQLite bool
)

func IsSQLite() bool {
	mu.RLock()
	defer mu.RUnlock()
	return isSQLite
}

func DriverName() string {
	if IsSQLite() {
		return "sqlite3"
	}
	return "postgres"
}

func Connect(databaseURL string) error {
	mu.Lock()
	defer mu.Unlock()

	url := strings.TrimSpace(databaseURL)

	if url == "" || strings.HasPrefix(url, "sqlite") {
		isSQLite = true
		return connectSQLite(url)
	}

	isSQLite = false
	return connectPostgres(url)
}

func connectSQLite(dsn string) error {
	if dsn == "" || dsn == "sqlite" {
		dsn = "lucid.db"
	}
	dsn = strings.TrimPrefix(dsn, "sqlite://")

	var err error
	DB, err = sql.Open("sqlite3", dsn)
	if err != nil {
		return fmt.Errorf("sqlite open: %w", err)
	}

	DB.SetMaxOpenConns(1)
	DB.SetMaxIdleConns(1)
	DB.SetConnMaxLifetime(0)

	if _, err := DB.Exec("PRAGMA journal_mode=WAL"); err != nil {
		log.Printf("Warning: could not set WAL mode: %v", err)
	}
	if _, err := DB.Exec("PRAGMA foreign_keys=ON"); err != nil {
		log.Printf("Warning: could not enable foreign keys: %v", err)
	}

	if err = retryPing(DB, 3); err != nil {
		return fmt.Errorf("sqlite ping: %w", err)
	}

	log.Printf("Connected to SQLite: %s", dsn)
	return nil
}

func connectPostgres(databaseURL string) error {
	var err error
	DB, err = sql.Open("postgres", databaseURL)
	if err != nil {
		return fmt.Errorf("postgres open: %w", err)
	}

	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(5)
	DB.SetConnMaxLifetime(30 * time.Minute)
	DB.SetConnMaxIdleTime(5 * time.Minute)

	if err = retryPing(DB, 5); err != nil {
		return fmt.Errorf("postgres ping: %w", err)
	}

	log.Println("Connected to PostgreSQL")
	return nil
}

func retryPing(db *sql.DB, maxRetries int) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		err = db.Ping()
		if err == nil {
			return nil
		}
		wait := time.Duration(1<<uint(i)) * 500 * time.Millisecond
		log.Printf("DB ping failed (attempt %d/%d): %v — retrying in %v", i+1, maxRetries, err, wait)
		time.Sleep(wait)
	}
	return err
}

func RunMigrations() error {
	return Migrate()
}

func Close() {
	mu.Lock()
	defer mu.Unlock()
	if DB != nil {
		DB.Close()
		log.Println("Database connection closed")
	}
}



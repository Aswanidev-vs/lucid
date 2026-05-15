package db

import (
	"log"
)

func Migrate() error {
	if IsSQLite() {
		migrateSQLite()
	} else {
		migratePostgres()
	}
	return nil
}

func migrateSQLite() {
	// Enable WAL and foreign keys
	DB.Exec("PRAGMA journal_mode=WAL")
	DB.Exec("PRAGMA foreign_keys=ON")


	// Users table
	userQuery := `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			bio TEXT DEFAULT '',
			avatar_url TEXT DEFAULT '',
			dreamer_type TEXT DEFAULT 'Explorer',
			is_public_profile INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`
	if _, err := DB.Exec(userQuery); err != nil {
		log.Fatalf("Failed to create users table: %v", err)
	}

	// Dream categories
	categoryQuery := `
		CREATE TABLE IF NOT EXISTS dream_categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			emoji TEXT DEFAULT '💭',
			description TEXT DEFAULT ''
		)
	`
	if _, err := DB.Exec(categoryQuery); err != nil {
		log.Fatalf("Failed to create dream_categories table: %v", err)
	}

	// Dreams table
	dreamQuery := `
		CREATE TABLE IF NOT EXISTS dreams (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			category_id INTEGER REFERENCES dream_categories(id) ON DELETE SET NULL,
			mood TEXT,
			visibility TEXT NOT NULL DEFAULT 'private',
			is_lucid INTEGER DEFAULT 0,
			is_recurring INTEGER DEFAULT 0,
			is_nightmare INTEGER DEFAULT 0,
			lucidity_level INTEGER DEFAULT 0 CHECK (lucidity_level >= 0 AND lucidity_level <= 10),
			dream_tags TEXT DEFAULT '',
			like_count INTEGER DEFAULT 0,
			comment_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`
	if _, err := DB.Exec(dreamQuery); err != nil {
		log.Fatalf("Failed to create dreams table: %v", err)
	}

	// Comments table
	commentQuery := `
		CREATE TABLE IF NOT EXISTS dream_comments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			dream_id INTEGER REFERENCES dreams(id) ON DELETE CASCADE,
			user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
			content TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`
	if _, err := DB.Exec(commentQuery); err != nil {
		log.Fatalf("Failed to create dream_comments table: %v", err)
	}

	// Likes table
	likeQuery := `
		CREATE TABLE IF NOT EXISTS dream_likes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			dream_id INTEGER REFERENCES dreams(id) ON DELETE CASCADE,
			user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(dream_id, user_id)
		)
	`
	if _, err := DB.Exec(likeQuery); err != nil {
		log.Fatalf("Failed to create dream_likes table: %v", err)
	}

	// Trending themes table
	trendQuery := `
		CREATE TABLE IF NOT EXISTS trending_themes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			theme TEXT UNIQUE NOT NULL,
			mention_count INTEGER DEFAULT 1,
			last_mentioned DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`
	if _, err := DB.Exec(trendQuery); err != nil {
		log.Fatalf("Failed to create trending_themes table: %v", err)
	}

	// Seed categories (SQLite uses INSERT OR IGNORE)
	seedCategories := `
		INSERT OR IGNORE INTO dream_categories (name, emoji, description) VALUES
			('Lucid', '🌙', 'Dreams where you are aware you are dreaming'),
			('Recurring', '🔄', 'Dreams that repeat with similar themes'),
			('Nightmare', '😱', 'Frightening or disturbing dreams'),
			('Flying', '🕊', 'Dreams involving flight or levitation'),
			('Falling', '🌀', 'Dreams of falling from heights'),
			('Water', '🌊', 'Dreams featuring water, oceans, or rain'),
			('Chase', '🏃', 'Dreams of being chased or pursued'),
			('Romantic', '💕', 'Romantic or intimate dreams'),
			('Prophetic', '🔮', 'Dreams that feel prophetic or precognitive'),
			('Other', '💭', 'Other types of dreams')
	`
	if _, err := DB.Exec(seedCategories); err != nil {
		log.Printf("Warning: Could not seed categories: %v", err)
	}

	// Indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_dreams_user_id ON dreams(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_dreams_visibility ON dreams(visibility)",
		"CREATE INDEX IF NOT EXISTS idx_dreams_created_at ON dreams(created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_dream_comments_dream_id ON dream_comments(dream_id)",
		"CREATE INDEX IF NOT EXISTS idx_dream_likes_dream_id ON dream_likes(dream_id)",
		"CREATE INDEX IF NOT EXISTS idx_dream_likes_user_id ON dream_likes(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_trending_themes_mention ON trending_themes(mention_count DESC)",
	}
	for _, idx := range indexes {
		if _, err := DB.Exec(idx); err != nil {
			log.Printf("Warning: index creation skipped: %v", err)
		}
	}

	log.Println("SQLite migration completed successfully")
}

func migratePostgres() {
	// Users table
	userQuery := `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username VARCHAR(50) UNIQUE NOT NULL,
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			bio TEXT DEFAULT '',
			avatar_url TEXT DEFAULT '',
			dreamer_type VARCHAR(50) DEFAULT 'Explorer',
			is_public_profile BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`
	if _, err := DB.Exec(userQuery); err != nil {
		log.Fatalf("Failed to create users table: %v", err)
	}

	// Dream categories
	categoryQuery := `
		CREATE TABLE IF NOT EXISTS dream_categories (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) UNIQUE NOT NULL,
			emoji VARCHAR(10) DEFAULT '💭',
			description TEXT DEFAULT ''
		)
	`
	if _, err := DB.Exec(categoryQuery); err != nil {
		log.Fatalf("Failed to create dream_categories table: %v", err)
	}

	// Dreams table
	dreamQuery := `
		CREATE TABLE IF NOT EXISTS dreams (
			id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			category_id INTEGER REFERENCES dream_categories(id) ON DELETE SET NULL,
			mood TEXT,
			visibility TEXT NOT NULL DEFAULT 'private',
			is_lucid BOOLEAN DEFAULT FALSE,
			is_recurring BOOLEAN DEFAULT FALSE,
			is_nightmare BOOLEAN DEFAULT FALSE,
			lucidity_level INTEGER DEFAULT 0 CHECK (lucidity_level >= 0 AND lucidity_level <= 10),
			dream_tags TEXT[] DEFAULT '{}',
			like_count INTEGER DEFAULT 0,
			comment_count INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`
	if _, err := DB.Exec(dreamQuery); err != nil {
		log.Fatalf("Failed to create dreams table: %v", err)
	}

	// Comments table
	commentQuery := `
		CREATE TABLE IF NOT EXISTS dream_comments (
			id SERIAL PRIMARY KEY,
			dream_id INTEGER REFERENCES dreams(id) ON DELETE CASCADE,
			user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
			content TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`
	if _, err := DB.Exec(commentQuery); err != nil {
		log.Fatalf("Failed to create dream_comments table: %v", err)
	}

	// Likes table
	likeQuery := `
		CREATE TABLE IF NOT EXISTS dream_likes (
			id SERIAL PRIMARY KEY,
			dream_id INTEGER REFERENCES dreams(id) ON DELETE CASCADE,
			user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(dream_id, user_id)
		)
	`
	if _, err := DB.Exec(likeQuery); err != nil {
		log.Fatalf("Failed to create dream_likes table: %v", err)
	}

	// Trending themes table
	trendQuery := `
		CREATE TABLE IF NOT EXISTS trending_themes (
			id SERIAL PRIMARY KEY,
			theme TEXT UNIQUE NOT NULL,
			mention_count INTEGER DEFAULT 1,
			last_mentioned TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`
	if _, err := DB.Exec(trendQuery); err != nil {
		log.Fatalf("Failed to create trending_themes table: %v", err)
	}

	// Seed categories (PostgreSQL uses ON CONFLICT DO NOTHING)
	seedCategories := `
		INSERT INTO dream_categories (name, emoji, description) VALUES
			('Lucid', '🌙', 'Dreams where you are aware you are dreaming'),
			('Recurring', '🔄', 'Dreams that repeat with similar themes'),
			('Nightmare', '😱', 'Frightening or disturbing dreams'),
			('Flying', '🕊️', 'Dreams involving flight or levitation'),
			('Falling', '🌀', 'Dreams of falling from heights'),
			('Water', '🌊', 'Dreams featuring water, oceans, or rain'),
			('Chase', '🏃', 'Dreams of being chased or pursued'),
			('Romantic', '💕', 'Romantic or intimate dreams'),
			('Prophetic', '🔮', 'Dreams that feel prophetic or precognitive'),
			('Other', '💭', 'Other types of dreams')
		ON CONFLICT (name) DO NOTHING
	`
	if _, err := DB.Exec(seedCategories); err != nil {
		log.Printf("Warning: Could not seed categories (may already exist): %v", err)
	}

	// Indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_dreams_user_id ON dreams(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_dreams_visibility ON dreams(visibility)",
		"CREATE INDEX IF NOT EXISTS idx_dreams_created_at ON dreams(created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_dream_comments_dream_id ON dream_comments(dream_id)",
		"CREATE INDEX IF NOT EXISTS idx_dream_likes_dream_id ON dream_likes(dream_id)",
		"CREATE INDEX IF NOT EXISTS idx_dream_likes_user_id ON dream_likes(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_trending_themes_mention ON trending_themes(mention_count DESC)",
	}
	for _, idx := range indexes {
		if _, err := DB.Exec(idx); err != nil {
			log.Printf("Warning: index creation skipped: %v", err)
		}
	}

	log.Println("PostgreSQL migration completed successfully")
}

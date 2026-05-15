package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Aswanidev-vs/lucid/db"
	"github.com/Aswanidev-vs/lucid/internal/middleware"
	"github.com/Aswanidev-vs/lucid/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/microcosm-cc/bluemonday"
)

// Security utilities
var sanitizer = bluemonday.UGCPolicy()

// Input validation
func validateDreamInput(title, content, mood string) error {
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)
	mood = strings.TrimSpace(mood)

	if title == "" {
		return errors.New("dream title is required")
	}
	if utf8.RuneCountInString(title) > 200 {
		return errors.New("dream title must be less than 200 characters")
	}
	if content == "" {
		return errors.New("dream content is required")
	}
	if utf8.RuneCountInString(content) > 10000 {
		return errors.New("dream content must be less than 10,000 characters")
	}
	validMoods := map[string]bool{
		"": true, "Calm": true, "Haunted": true, "Inspired": true,
		"Happy": true, "Anxious": true,
	}
	if mood != "" && !validMoods[mood] {
		return errors.New("invalid mood selection")
	}
	return nil
}

func sanitizeInput(input string) string {
	return sanitizer.Sanitize(input)
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r)

	// Get category list for the landing page
	catRows, err := db.DB.Query("SELECT id, name, emoji, description FROM dream_categories ORDER BY id")
	categories := []model.DreamCategory{}
	if err == nil {
		defer catRows.Close()
		for catRows.Next() {
			var c model.DreamCategory
			if err := catRows.Scan(&c.ID, &c.Name, &c.Emoji, &c.Description); err == nil {
				categories = append(categories, c)
			}
		}
	}

	// Get trending themes
	trending := getTrendingThemes()

	tmpl, err := ParseTemplate("index.html")
	if err != nil {
		log.Printf("Template parsing error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, map[string]interface{}{
		"IsLoggedIn": claims != nil,
		"Username":   getUsername(claims),
		"Categories": categories,
		"Trending":   trending,
	}); err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func CreateDream(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r)
	userID := 0
	if claims != nil {
		userID = claims.UserID
	}

	if err := r.ParseForm(); err != nil {
		log.Printf("Form parsing error: %v", err)
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	title := r.FormValue("title")
	content := r.FormValue("content")
	mood := r.FormValue("mood")
	visibility := "private"
	if r.FormValue("visibility") == "public" || r.FormValue("is_public") == "on" {
		visibility = "public"
	}

	categoryIDStr := r.FormValue("category_id")
	var categoryID *int
	if categoryIDStr != "" {
		if id, err := strconv.Atoi(categoryIDStr); err == nil && id > 0 {
			categoryID = &id
		}
	}

	isLucid := r.FormValue("is_lucid") == "on"
	isRecurring := r.FormValue("is_recurring") == "on"
	isNightmare := r.FormValue("is_nightmare") == "on"

	lucidityLevelStr := r.FormValue("lucidity_level")
	lucidityLevel := 0
	if lucidityLevelStr != "" {
		if l, err := strconv.Atoi(lucidityLevelStr); err == nil && l >= 0 && l <= 10 {
			lucidityLevel = l
		}
	}

	tagsFormValue := r.FormValue("dream_tags")
	var tags []string
	if tagsFormValue != "" {
		for _, tag := range strings.Split(tagsFormValue, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tags = append(tags, tag)
			}
		}
	}

	if err := validateDreamInput(title, content, mood); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	title = sanitizeInput(title)
	content = sanitizeInput(content)
	mood = sanitizeInput(mood)

	// Build tags string (SQLite: comma-separated, PG: {tag1,tag2})
	var tagsStr string
	if db.IsSQLite() {
		tagsStr = strings.Join(tags, ",")
	} else {
		tagsStr = "{" + strings.Join(tags, ",") + "}"
	}

	query := `
		INSERT INTO dreams (user_id, title, content, category_id, mood, visibility, 
			is_lucid, is_recurring, is_nightmare, lucidity_level, dream_tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := db.DB.Exec(query, userID, title, content, categoryID, mood, visibility,
		isLucid, isRecurring, isNightmare, lucidityLevel, tagsStr)
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Failed to save dream. Please try again.", http.StatusInternalServerError)
		return
	}

	// Track trending themes based on content keywords
	trackTrendingThemes(title + " " + content)

	http.Redirect(w, r, "/dreams", http.StatusSeeOther)
}

func GetDreams(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r)
	userID := 0
	if claims != nil {
		userID = claims.UserID
	}

	rows, err := db.DB.Query(`
		SELECT d.id, d.user_id, d.title, d.content, d.mood, d.visibility,
			d.is_lucid, d.is_recurring, d.is_nightmare, d.lucidity_level,
			d.dream_tags, d.like_count, d.comment_count, d.created_at,
			u.username,
			COALESCE(dc.name, '') as category_name,
			COALESCE(dc.emoji, '') as category_emoji
		FROM dreams d
		JOIN users u ON d.user_id = u.id
		LEFT JOIN dream_categories dc ON d.category_id = dc.id
		WHERE d.user_id = $1
		ORDER BY d.created_at DESC
	`, userID)
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Failed to load dreams", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var dreams []model.Dream
	for rows.Next() {
		var dream model.Dream
		var tags []byte
		err := rows.Scan(
			&dream.ID, &dream.UserID, &dream.Title, &dream.Content, &dream.Mood, &dream.Visibility,
			&dream.IsLucid, &dream.IsRecurring, &dream.IsNightmare, &dream.LucidityLevel,
			&tags, &dream.LikeCount, &dream.CommentCount, &dream.CreatedAt,
			&dream.Username, &dream.CategoryName, &dream.CategoryEmoji,
		)
		if err != nil {
			log.Printf("Row scan error: %v", err)
			continue
		}
		dream.DreamTags = parseTags(tags)
		dreams = append(dreams, dream)
	}

	tmpl, err := ParseTemplate("templates/dream.html")
	if err != nil {
		log.Printf("Template parsing error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, map[string]interface{}{
		"Dreams":     dreams,
		"IsLoggedIn": claims != nil,
		"Username":   getUsername(claims),
	}); err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func NewDreamPage(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r)

	// Get categories for the form
	catRows, err := db.DB.Query("SELECT id, name, emoji, description FROM dream_categories ORDER BY id")
	categories := []model.DreamCategory{}
	if err == nil {
		defer catRows.Close()
		for catRows.Next() {
			var c model.DreamCategory
			if err := catRows.Scan(&c.ID, &c.Name, &c.Emoji, &c.Description); err == nil {
				categories = append(categories, c)
			}
		}
	}

	tmpl, err := ParseTemplate("templates/new.html")
	if err != nil {
		log.Printf("Template parsing error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, map[string]interface{}{
		"Categories": categories,
		"IsLoggedIn": claims != nil,
		"Username":   getUsername(claims),
	}); err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func GetDream(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r)
	userID := 0
	if claims != nil {
		userID = claims.UserID
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid dream ID", http.StatusBadRequest)
		return
	}

	var dream model.Dream
	var tags []byte
	query := `
		SELECT d.id, d.user_id, d.title, d.content, d.mood, d.visibility,
			d.is_lucid, d.is_recurring, d.is_nightmare, d.lucidity_level,
			d.dream_tags, d.like_count, d.comment_count, d.created_at, d.updated_at,
			u.username,
			COALESCE(dc.name, '') as category_name,
			COALESCE(dc.emoji, '') as category_emoji
		FROM dreams d
		JOIN users u ON d.user_id = u.id
		LEFT JOIN dream_categories dc ON d.category_id = dc.id
		WHERE d.id = $1
	`
	row := db.DB.QueryRow(query, id)
	err = row.Scan(
		&dream.ID, &dream.UserID, &dream.Title, &dream.Content, &dream.Mood, &dream.Visibility,
		&dream.IsLucid, &dream.IsRecurring, &dream.IsNightmare, &dream.LucidityLevel,
		&tags, &dream.LikeCount, &dream.CommentCount, &dream.CreatedAt, &dream.UpdatedAt,
		&dream.Username, &dream.CategoryName, &dream.CategoryEmoji,
	)
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Dream not found", http.StatusNotFound)
		return
	}
	dream.DreamTags = parseTags(tags)

	// Check if private dream belongs to current user
	if dream.Visibility != "public" && dream.UserID != userID {
		http.Error(w, "Dream not found", http.StatusNotFound)
		return
	}

	// Get comments
	commentRows, err := db.DB.Query(`
		SELECT c.id, c.dream_id, c.user_id, c.content, c.created_at,
			u.username, COALESCE(u.avatar_url, '') as avatar_url
		FROM dream_comments c
		JOIN users u ON c.user_id = u.id
		WHERE c.dream_id = $1
		ORDER BY c.created_at DESC
		LIMIT 20
	`, dream.ID)
	comments := []model.Comment{}
	if err == nil {
		defer commentRows.Close()
		for commentRows.Next() {
			var c model.Comment
			if err := commentRows.Scan(&c.ID, &c.DreamID, &c.UserID, &c.Content, &c.CreatedAt, &c.Username, &c.AvatarURL); err == nil {
				comments = append(comments, c)
			}
		}
	}

	// Check if current user liked this dream
	if userID > 0 {
		var liked bool
		db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM dream_likes WHERE dream_id=$1 AND user_id=$2)", dream.ID, userID).Scan(&liked)
		dream.IsLikedByMe = liked
	}

	tmpl, err := ParseTemplate("templates/view.html")
	if err != nil {
		log.Printf("Template parsing error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, map[string]interface{}{
		"Dream":      dream,
		"Comments":   comments,
		"IsOwner":    dream.UserID == userID,
		"IsLoggedIn": claims != nil,
		"Username":   getUsername(claims),
		"UserID":     userID,
	}); err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func EditDreamPage(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r)
	userID := 0
	if claims != nil {
		userID = claims.UserID
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid dream ID", http.StatusBadRequest)
		return
	}

	var dream model.Dream
	query := "SELECT id, user_id, title, content, mood, visibility FROM dreams WHERE id = $1"
	row := db.DB.QueryRow(query, id)
	err = row.Scan(&dream.ID, &dream.UserID, &dream.Title, &dream.Content, &dream.Mood, &dream.Visibility)
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Dream not found", http.StatusNotFound)
		return
	}

	// Only owner can edit
	if dream.UserID != userID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Get categories for form
	catRows, err := db.DB.Query("SELECT id, name, emoji, description FROM dream_categories ORDER BY id")
	categories := []model.DreamCategory{}
	if err == nil {
		defer catRows.Close()
		for catRows.Next() {
			var c model.DreamCategory
			if err := catRows.Scan(&c.ID, &c.Name, &c.Emoji, &c.Description); err == nil {
				categories = append(categories, c)
			}
		}
	}

	tmpl, err := ParseTemplate("templates/edit.html")
	if err != nil {
		log.Printf("Template parsing error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, map[string]interface{}{
		"Dream":      dream,
		"Categories": categories,
		"IsLoggedIn": claims != nil,
		"Username":   getUsername(claims),
	}); err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func UpdateDream(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r)
	userID := 0
	if claims != nil {
		userID = claims.UserID
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid dream ID", http.StatusBadRequest)
		return
	}

	// Check ownership
	var ownerID int
	db.DB.QueryRow("SELECT user_id FROM dreams WHERE id=$1", id).Scan(&ownerID)
	if ownerID != userID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Printf("Form parsing error: %v", err)
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	title := r.FormValue("title")
	content := r.FormValue("content")
	mood := r.FormValue("mood")
	visibility := "private"
	if r.FormValue("visibility") == "public" || r.FormValue("is_public") == "on" {
		visibility = "public"
	}

	if err := validateDreamInput(title, content, mood); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	title = sanitizeInput(title)
	content = sanitizeInput(content)
	mood = sanitizeInput(mood)

	query := "UPDATE dreams SET title = $1, content = $2, mood = $3, visibility = $4 WHERE id = $5"
	_, err = db.DB.Exec(query, title, content, mood, visibility, id)
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Failed to update dream. Please try again.", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/dreams/"+idStr, http.StatusSeeOther)
}

func GetPublicDreams(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r)

	rows, err := db.DB.Query(`
		SELECT d.id, d.user_id, d.title, d.content, d.mood, d.visibility,
			d.is_lucid, d.is_recurring, d.is_nightmare, d.lucidity_level,
			d.dream_tags, d.like_count, d.comment_count, d.created_at,
			u.username,
			COALESCE(dc.name, '') as category_name,
			COALESCE(dc.emoji, '') as category_emoji
		FROM dreams d
		JOIN users u ON d.user_id = u.id
		LEFT JOIN dream_categories dc ON d.category_id = dc.id
		WHERE d.visibility = 'public'
		ORDER BY d.created_at DESC
	`)
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Failed to load dreams", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var dreams []model.Dream
	for rows.Next() {
		var dream model.Dream
		var tags []byte
		err := rows.Scan(
			&dream.ID, &dream.UserID, &dream.Title, &dream.Content, &dream.Mood, &dream.Visibility,
			&dream.IsLucid, &dream.IsRecurring, &dream.IsNightmare, &dream.LucidityLevel,
			&tags, &dream.LikeCount, &dream.CommentCount, &dream.CreatedAt,
			&dream.Username, &dream.CategoryName, &dream.CategoryEmoji,
		)
		if err != nil {
			log.Printf("Row scan error: %v", err)
			continue
		}
		dream.DreamTags = parseTags(tags)
		dreams = append(dreams, dream)
	}

	tmpl, err := ParseTemplate("templates/public.html")
	if err != nil {
		log.Printf("Template parsing error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, map[string]interface{}{
		"Dreams":     dreams,
		"IsLoggedIn": claims != nil,
		"Username":   getUsername(claims),
	}); err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func DeleteDream(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r)
	userID := 0
	if claims != nil {
		userID = claims.UserID
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid dream ID", http.StatusBadRequest)
		return
	}

	// Check ownership
	var ownerID int
	db.DB.QueryRow("SELECT user_id FROM dreams WHERE id=$1", id).Scan(&ownerID)
	if ownerID != userID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	query := "DELETE FROM dreams WHERE id = $1"
	result, err := db.DB.Exec(query, id)
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Failed to delete dream", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Rows affected check error: %v", err)
		http.Error(w, "Failed to delete dream", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "Dream not found", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, "/dreams", http.StatusSeeOther)
}

// Track trending themes from dream content
func trackTrendingThemes(text string) {
	themeKeywords := map[string][]string{
		"Flying":  {"fly", "flying", "flight", "soar", "soaring", "wings", "sky"},
		"Falling": {"fall", "falling", "fell", "gravity", "drop"},
		"Water":   {"water", "ocean", "sea", "river", "lake", "swim", "swimming", "rain", "flood", "wave"},
		"Chase":   {"chase", "chasing", "run", "running", "escape", "flee", "pursue"},
		"Teeth":   {"teeth", "tooth", "falling out", "rotten"},
		"Test":    {"exam", "test", "school", "unprepared", "late"},
		"Death":   {"death", "die", "dying", "dead", "funeral", "grave"},
		"Romance": {"romance", "romantic", "kiss", "kissing", "love", "date", "partner"},
		"Animal":  {"animal", "dog", "cat", "horse", "beast", "creature"},
		"House":   {"house", "home", "room", "door", "hallway", "stairs", "building"},
	}

	textLower := strings.ToLower(text)
	now := time.Now()

	for theme, keywords := range themeKeywords {
		for _, keyword := range keywords {
			if strings.Contains(textLower, keyword) {
				if db.IsSQLite() {
					_, err := db.DB.Exec(`
						INSERT INTO trending_themes (theme, mention_count, last_mentioned)
						VALUES ($1, 1, $2)
						ON CONFLICT(theme) DO UPDATE SET
							mention_count = mention_count + 1,
							last_mentioned = $2
					`, theme, now)
					if err != nil {
						log.Printf("Error updating trending theme: %v", err)
					}
				} else {
					_, err := db.DB.Exec(`
						INSERT INTO trending_themes (theme, mention_count, last_mentioned)
						VALUES ($1, 1, $2)
						ON CONFLICT (theme) DO UPDATE SET
							mention_count = trending_themes.mention_count + 1,
							last_mentioned = $2
					`, theme, now)
					if err != nil {
						log.Printf("Error updating trending theme: %v", err)
					}
				}
				break
			}
		}
	}
}

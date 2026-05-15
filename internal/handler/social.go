package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/Aswanidev-vs/lucid/db"
	"github.com/Aswanidev-vs/lucid/internal/middleware"
	"github.com/Aswanidev-vs/lucid/internal/model"
	"github.com/go-chi/chi/v5"
)

// ShowFeed renders the social feed page
func ShowFeed(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r)
	loggedInUserID := 0
	if claims != nil {
		loggedInUserID = claims.UserID
	}

	// Get trending themes
	trending := getTrendingThemes()

	// Get public dreams for feed
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
		WHERE d.visibility = 'public'
		ORDER BY d.created_at DESC
		LIMIT 50
	`
	rows, err := db.DB.Query(query)
	if err != nil {
		log.Printf("Database error fetching feed: %v", err)
		http.Error(w, "Failed to load feed", http.StatusInternalServerError)
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
			&tags, &dream.LikeCount, &dream.CommentCount, &dream.CreatedAt, &dream.UpdatedAt,
			&dream.Username, &dream.CategoryName, &dream.CategoryEmoji,
		)
		if err != nil {
			log.Printf("Row scan error: %v", err)
			continue
		}
		dream.DreamTags = parseTags(tags)
		dreams = append(dreams, dream)
	}

	// Check if user has liked these dreams
	if loggedInUserID > 0 && len(dreams) > 0 {
		likedDreams := getUserLikedDreams(loggedInUserID)
		for i := range dreams {
			if _, exists := likedDreams[dreams[i].ID]; exists {
				dreams[i].IsLikedByMe = true
			}
		}
	}

	// Get dreamer types for leaderboard
	var leaders []model.User
	leaderRows, err := db.DB.Query(`
		SELECT u.id, u.username, u.dreamer_type, u.avatar_url,
			COUNT(d.id) as dream_count
		FROM users u
		LEFT JOIN dreams d ON d.user_id = u.id AND d.visibility = 'public'
		GROUP BY u.id, u.username, u.dreamer_type, u.avatar_url
		ORDER BY dream_count DESC
		LIMIT 5
	`)
	if err == nil {
		defer leaderRows.Close()
		for leaderRows.Next() {
			var u model.User
			if err := leaderRows.Scan(&u.ID, &u.Username, &u.DreamerType, &u.AvatarURL, &u.DreamCount); err == nil {
				leaders = append(leaders, u)
			}
		}
	}

	tmpl, err := ParseTemplate("templates/feed.html")
	if err != nil {
		log.Printf("Template parsing error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, map[string]interface{}{
		"Dreams":     dreams,
		"Trending":   trending,
		"Leaders":    leaders,
		"IsLoggedIn": claims != nil,
		"Username":   getUsername(claims),
		"UserID":     loggedInUserID,
	}); err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// HandleLikeDream toggles a like on a dream
func HandleLikeDream(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r)
	if claims == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(model.APIResponse{Success: false, Error: "Not authenticated"})
		return
	}

	idStr := chi.URLParam(r, "id")
	dreamID, err := strconv.Atoi(idStr)
	if err != nil || dreamID <= 0 {
		http.Error(w, "Invalid dream ID", http.StatusBadRequest)
		return
	}

	// Check if already liked
	var exists bool
	db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM dream_likes WHERE dream_id=$1 AND user_id=$2)", dreamID, claims.UserID).Scan(&exists)

	if exists {
		// Unlike
		_, err = db.DB.Exec("DELETE FROM dream_likes WHERE dream_id=$1 AND user_id=$2", dreamID, claims.UserID)
		if err != nil {
			log.Printf("Database error unliking: %v", err)
			http.Error(w, "Failed to unlike", http.StatusInternalServerError)
			return
		}
		db.DB.Exec("UPDATE dreams SET like_count = (SELECT COUNT(*) FROM dream_likes WHERE dream_id=$1) WHERE id=$1", dreamID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(model.APIResponse{Success: true, Data: map[string]interface{}{"liked": false}})
		return
	}

	// Like
	_, err = db.DB.Exec("INSERT INTO dream_likes (dream_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", dreamID, claims.UserID)
	if err != nil {
		log.Printf("Database error liking: %v", err)
		http.Error(w, "Failed to like", http.StatusInternalServerError)
		return
	}
	db.DB.Exec("UPDATE dreams SET like_count = (SELECT COUNT(*) FROM dream_likes WHERE dream_id=$1) WHERE id=$1", dreamID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model.APIResponse{Success: true, Data: map[string]interface{}{"liked": true}})
}

// HandleAddComment adds a comment to a dream
func HandleAddComment(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r)
	if claims == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(model.APIResponse{Success: false, Error: "Not authenticated"})
		return
	}

	idStr := chi.URLParam(r, "id")
	dreamID, err := strconv.Atoi(idStr)
	if err != nil || dreamID <= 0 {
		http.Error(w, "Invalid dream ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	content := strings.TrimSpace(r.FormValue("content"))
	if content == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(model.APIResponse{Success: false, Error: "Comment cannot be empty"})
		return
	}
	if utf8.RuneCountInString(content) > 2000 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(model.APIResponse{Success: false, Error: "Comment must be less than 2000 characters"})
		return
	}

	content = sanitizeInput(content)

	_, err = db.DB.Exec("INSERT INTO dream_comments (dream_id, user_id, content) VALUES ($1, $2, $3)",
		dreamID, claims.UserID, content)
	if err != nil {
		log.Printf("Database error adding comment: %v", err)
		http.Error(w, "Failed to add comment", http.StatusInternalServerError)
		return
	}

	db.DB.Exec("UPDATE dreams SET comment_count = (SELECT COUNT(*) FROM dream_comments WHERE dream_id=$1) WHERE id=$1", dreamID)

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(model.APIResponse{Success: true, Message: "Comment added"})
		return
	}

	http.Redirect(w, r, "/dreams/"+idStr, http.StatusSeeOther)
}

// HandleGetComments retrieves comments for a dream
func HandleGetComments(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	dreamID, err := strconv.Atoi(idStr)
	if err != nil || dreamID <= 0 {
		http.Error(w, "Invalid dream ID", http.StatusBadRequest)
		return
	}

	rows, err := db.DB.Query(`
		SELECT c.id, c.dream_id, c.user_id, c.content, c.created_at,
			u.username, COALESCE(u.avatar_url, '') as avatar_url
		FROM dream_comments c
		JOIN users u ON c.user_id = u.id
		WHERE c.dream_id = $1
		ORDER BY c.created_at DESC
		LIMIT 50
	`, dreamID)
	if err != nil {
		log.Printf("Database error fetching comments: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(model.APIResponse{Success: false, Error: "Failed to load comments"})
		return
	}
	defer rows.Close()

	var comments []model.Comment
	for rows.Next() {
		var c model.Comment
		if err := rows.Scan(&c.ID, &c.DreamID, &c.UserID, &c.Content, &c.CreatedAt, &c.Username, &c.AvatarURL); err != nil {
			continue
		}
		comments = append(comments, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model.APIResponse{Success: true, Data: comments})
}

// HandleTrending returns trending dream themes
func HandleTrending(w http.ResponseWriter, r *http.Request) {
	themes := getTrendingThemes()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model.APIResponse{Success: true, Data: themes})
}

// Helper functions

func getTrendingThemes() []model.TrendingTheme {
	rows, err := db.DB.Query(`
		SELECT id, theme, mention_count, last_mentioned
		FROM trending_themes
		ORDER BY mention_count DESC, last_mentioned DESC
		LIMIT 10
	`)
	if err != nil {
		log.Printf("Database error fetching trending: %v", err)
		return nil
	}
	defer rows.Close()

	var themes []model.TrendingTheme
	for rows.Next() {
		var t model.TrendingTheme
		if err := rows.Scan(&t.ID, &t.Theme, &t.MentionCount, &t.LastMentioned); err != nil {
			continue
		}
		themes = append(themes, t)
	}
	return themes
}

func getUserLikedDreams(userID int) map[int]bool {
	liked := make(map[int]bool)
	rows, err := db.DB.Query("SELECT dream_id FROM dream_likes WHERE user_id=$1", userID)
	if err != nil {
		return liked
	}
	defer rows.Close()
	for rows.Next() {
		var dreamID int
		if rows.Scan(&dreamID) == nil {
			liked[dreamID] = true
		}
	}
	return liked
}

func parseTags(tags []byte) []string {
	if tags == nil {
		return []string{}
	}
	tagStr := string(tags)
	if tagStr == "{}" || tagStr == "" {
		return []string{}
	}
	tagStr = strings.Trim(tagStr, "{}")
	parts := strings.Split(tagStr, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, "\"")
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

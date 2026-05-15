package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/Aswanidev-vs/lucid/db"
	"github.com/Aswanidev-vs/lucid/internal/middleware"
	"github.com/Aswanidev-vs/lucid/internal/model"
	"github.com/go-chi/chi/v5"
)

// ShowProfile renders the user profile page
func ShowProfile(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r)
	loggedInUserID := 0
	if claims != nil {
		loggedInUserID = claims.UserID
	}

	idStr := chi.URLParam(r, "id")
	profileID, err := strconv.Atoi(idStr)
	if err != nil || profileID <= 0 {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Fetch user profile
	var user model.User
	err = db.DB.QueryRow(`
		SELECT id, username, email, bio, avatar_url, dreamer_type, is_public_profile, created_at
		FROM users WHERE id = $1
	`, profileID).Scan(&user.ID, &user.Username, &user.Email, &user.Bio, &user.AvatarURL,
		&user.DreamerType, &user.IsPublicProfile, &user.CreatedAt)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Count dreams
	db.DB.QueryRow("SELECT COUNT(*) FROM dreams WHERE user_id=$1", profileID).Scan(&user.DreamCount)

	// Count total likes
	db.DB.QueryRow("SELECT COALESCE(SUM(like_count), 0) FROM dreams WHERE user_id=$1", profileID).Scan(&user.TotalLikes)

	// Get user's dreams (public only for others, all for owner)
	var dreamRows []model.Dream
	if loggedInUserID == profileID {
		// Owner: see all dreams
		rows, err := db.DB.Query(`
			SELECT d.id, d.title, d.content, d.mood, d.visibility,
				d.is_lucid, d.is_recurring, d.is_nightmare, d.lucidity_level,
				d.like_count, d.comment_count, d.created_at,
				COALESCE(dc.name, '') as category_name,
				COALESCE(dc.emoji, '') as category_emoji
			FROM dreams d
			LEFT JOIN dream_categories dc ON d.category_id = dc.id
			WHERE d.user_id = $1
			ORDER BY d.created_at DESC
		`, profileID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var d model.Dream
				if err := rows.Scan(&d.ID, &d.Title, &d.Content, &d.Mood, &d.Visibility,
					&d.IsLucid, &d.IsRecurring, &d.IsNightmare, &d.LucidityLevel,
					&d.LikeCount, &d.CommentCount, &d.CreatedAt,
					&d.CategoryName, &d.CategoryEmoji); err == nil {
					dreamRows = append(dreamRows, d)
				}
			}
		}
	} else {
		// Others: see only public dreams
		rows, err := db.DB.Query(`
			SELECT d.id, d.title, d.content, d.mood, d.visibility,
				d.is_lucid, d.is_recurring, d.is_nightmare, d.lucidity_level,
				d.like_count, d.comment_count, d.created_at,
				COALESCE(dc.name, '') as category_name,
				COALESCE(dc.emoji, '') as category_emoji
			FROM dreams d
			LEFT JOIN dream_categories dc ON d.category_id = dc.id
			WHERE d.user_id = $1 AND d.visibility = 'public'
			ORDER BY d.created_at DESC
		`, profileID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var d model.Dream
				if err := rows.Scan(&d.ID, &d.Title, &d.Content, &d.Mood, &d.Visibility,
					&d.IsLucid, &d.IsRecurring, &d.IsNightmare, &d.LucidityLevel,
					&d.LikeCount, &d.CommentCount, &d.CreatedAt,
					&d.CategoryName, &d.CategoryEmoji); err == nil {
					dreamRows = append(dreamRows, d)
				}
			}
		}
	}

	// Get categories for chart
	var categoryCounts []map[string]interface{}
	catRows, err := db.DB.Query(`
		SELECT COALESCE(dc.name, 'Uncategorized') as name,
			COALESCE(dc.emoji, '💭') as emoji,
			COUNT(*) as count
		FROM dreams d
		LEFT JOIN dream_categories dc ON d.category_id = dc.id
		WHERE d.user_id = $1 AND d.visibility = 'public'
		GROUP BY dc.name, dc.emoji
		ORDER BY count DESC
	`, profileID)
	if err == nil {
		defer catRows.Close()
		for catRows.Next() {
			var name, emoji string
			var count int
			if err := catRows.Scan(&name, &emoji, &count); err == nil {
				categoryCounts = append(categoryCounts, map[string]interface{}{
					"name":  name,
					"emoji": emoji,
					"count": count,
				})
			}
		}
	}

	tmpl, err := ParseTemplate("templates/profile.html")
	if err != nil {
		log.Printf("Template parsing error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Parse import notification from query params
	importedStr := r.URL.Query().Get("imported")
	skippedStr := r.URL.Query().Get("skipped")
	imported := 0
	skipped := 0
	if importedStr != "" {
		imported, _ = strconv.Atoi(importedStr)
	}
	if skippedStr != "" {
		skipped, _ = strconv.Atoi(skippedStr)
	}

	if err := tmpl.Execute(w, map[string]interface{}{
		"Profile":        user,
		"Dreams":         dreamRows,
		"CategoryCounts": categoryCounts,
		"IsOwner":        loggedInUserID == profileID,
		"IsLoggedIn":     claims != nil,
		"Username":       getUsername(claims),
		"UserID":         loggedInUserID,
		"Imported":       imported,
		"Skipped":        skipped,
	}); err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// UpdateProfile updates the user's profile
func UpdateProfile(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r)
	if claims == nil {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	bio := r.FormValue("bio")
	dreamerType := r.FormValue("dreamer_type")

	_, err := db.DB.Exec("UPDATE users SET bio=$1, dreamer_type=$2 WHERE id=$3",
		bio, dreamerType, claims.UserID)
	if err != nil {
		log.Printf("Database error updating profile: %v", err)
		http.Error(w, "Failed to update profile", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/profile/"+strconv.Itoa(claims.UserID), http.StatusSeeOther)
}

// ImportUserData imports dreams from a JSON export file
func ImportUserData(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r)
	if claims == nil {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Failed to parse upload: "+err.Error(), http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("import_file")
	if err != nil {
		http.Error(w, "No file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	var importData model.ExportData
	if err := json.NewDecoder(file).Decode(&importData); err != nil {
		http.Error(w, "Invalid JSON file: "+err.Error(), http.StatusBadRequest)
		return
	}

	imported := 0
	skipped := 0
	for _, d := range importData.Dreams {
		// Map category name to ID
		var categoryID *int
		if d.CategoryName != "" {
			var cid int
			err := db.DB.QueryRow("SELECT id FROM dream_categories WHERE name=$1", d.CategoryName).Scan(&cid)
			if err == nil {
				categoryID = &cid
			}
		}

		// Build tags array
		tagsStr := "{}"
		if len(d.DreamTags) > 0 {
			tagsStr = "{" + strings.Join(d.DreamTags, ",") + "}"
		}

		_, err := db.DB.Exec(`
			INSERT INTO dreams (user_id, title, content, category_id, mood, visibility,
				is_lucid, is_recurring, is_nightmare, lucidity_level, dream_tags)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`, claims.UserID, d.Title, d.Content, categoryID, d.Mood, d.Visibility,
			d.IsLucid, d.IsRecurring, d.IsNightmare, d.LucidityLevel, tagsStr)
		if err != nil {
			log.Printf("Import error for dream '%s': %v", d.Title, err)
			skipped++
			continue
		}

		// Track trending themes from imported content
		trackTrendingThemes(d.Title + " " + d.Content)
		imported++
	}

	http.Redirect(w, r, "/profile/"+strconv.Itoa(claims.UserID)+"?imported="+strconv.Itoa(imported)+"&skipped="+strconv.Itoa(skipped), http.StatusSeeOther)
}

// ExportUserData exports all user data as JSON
func ExportUserData(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r)
	if claims == nil {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	var export model.ExportData

	// Get user info
	err := db.DB.QueryRow(`
		SELECT id, username, email, bio, avatar_url, dreamer_type, is_public_profile, created_at
		FROM users WHERE id = $1
	`, claims.UserID).Scan(&export.User.ID, &export.User.Username, &export.User.Email,
		&export.User.Bio, &export.User.AvatarURL, &export.User.DreamerType,
		&export.User.IsPublicProfile, &export.User.CreatedAt)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Get all dreams
	dreamRows, err := db.DB.Query(`
		SELECT id, user_id, title, content, COALESCE(category_id, 0), mood, visibility,
			is_lucid, is_recurring, is_nightmare, lucidity_level, dream_tags,
			like_count, comment_count, created_at, updated_at
		FROM dreams WHERE user_id = $1
		ORDER BY created_at DESC
	`, claims.UserID)
	if err == nil {
		defer dreamRows.Close()
		for dreamRows.Next() {
			var d model.Dream
			var catID int
			var tags []byte
			if err := dreamRows.Scan(&d.ID, &d.UserID, &d.Title, &d.Content, &catID,
				&d.Mood, &d.Visibility, &d.IsLucid, &d.IsRecurring, &d.IsNightmare,
				&d.LucidityLevel, &tags, &d.LikeCount, &d.CommentCount, &d.CreatedAt, &d.UpdatedAt); err == nil {
				if catID > 0 {
					id := catID
					d.CategoryID = &id
				}
				d.DreamTags = parseTags(tags)
				export.Dreams = append(export.Dreams, d)
			}
		}
	}

	// Get comments
	commentRows, err := db.DB.Query(`
		SELECT id, dream_id, user_id, content, created_at
		FROM dream_comments WHERE user_id = $1
		ORDER BY created_at DESC
	`, claims.UserID)
	if err == nil {
		defer commentRows.Close()
		for commentRows.Next() {
			var c model.Comment
			if err := commentRows.Scan(&c.ID, &c.DreamID, &c.UserID, &c.Content, &c.CreatedAt); err == nil {
				export.Comments = append(export.Comments, c)
			}
		}
	}

	// Get likes
	likeRows, err := db.DB.Query(`
		SELECT id, dream_id, user_id, created_at
		FROM dream_likes WHERE user_id = $1
		ORDER BY created_at DESC
	`, claims.UserID)
	if err == nil {
		defer likeRows.Close()
		for likeRows.Next() {
			var l model.Like
			if err := likeRows.Scan(&l.ID, &l.DreamID, &l.UserID, &l.CreatedAt); err == nil {
				export.Likes = append(export.Likes, l)
			}
		}
	}

	// Count stats
	export.TotalDreams = len(export.Dreams)
	for _, d := range export.Dreams {
		if d.Visibility == "public" {
			export.PublicDreams++
		} else {
			export.PrivateDreams++
		}
	}

	// Set headers for file download
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=lucid_dreams_export_"+claims.Username+".json")

	if err := json.NewEncoder(w).Encode(export); err != nil {
		log.Printf("Error encoding export: %v", err)
	}
}

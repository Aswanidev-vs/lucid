package handler

import (
	"errors"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/Aswanidev-vs/lucid/db"
	"github.com/go-chi/chi/v5"
	"github.com/microcosm-cc/bluemonday"
)

// Security utilities
var sanitizer = bluemonday.UGCPolicy()

// Input validation
func validateDreamInput(title, content, mood string) error {
	// Trim whitespace
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)
	mood = strings.TrimSpace(mood)

	// Title validation
	if title == "" {
		return errors.New("dream title is required")
	}
	if utf8.RuneCountInString(title) > 200 {
		return errors.New("dream title must be less than 200 characters")
	}

	// Content validation
	if content == "" {
		return errors.New("dream content is required")
	}
	if utf8.RuneCountInString(content) > 10000 {
		return errors.New("dream content must be less than 10,000 characters")
	}

	// Mood validation (optional)
	validMoods := map[string]bool{
		"": true, "Calm": true, "Haunted": true, "Inspired": true,
		"Happy": true, "Anxious": true,
	}
	if mood != "" && !validMoods[mood] {
		return errors.New("invalid mood selection")
	}

	return nil
}

// Sanitize user input to prevent XSS
func sanitizeInput(input string) string {
	return sanitizer.Sanitize(input)
}

type Dream struct {
	Id        int
	Title     string
	Content   string
	Mood      string
	IsPublic  bool
	CreatedAt string
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		log.Printf("Template parsing error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, nil); err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
func CreateDream(w http.ResponseWriter, r *http.Request) {
	// Parse form data
	if err := r.ParseForm(); err != nil {
		log.Printf("Form parsing error: %v", err)
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	title := r.FormValue("title")
	content := r.FormValue("content")
	mood := r.FormValue("mood")
	isPublic := r.FormValue("is_public") == "on"

	// Validate input
	if err := validateDreamInput(title, content, mood); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Sanitize input to prevent XSS
	title = sanitizeInput(title)
	content = sanitizeInput(content)
	mood = sanitizeInput(mood)

	query := `
	INSERT INTO dreams(title,content,mood,is_public) VALUES($1,$2,$3,$4)
	`
	_, err := db.DB.Exec(query, title, content, mood, isPublic)
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Failed to save dream. Please try again.", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/dreams", http.StatusSeeOther)
}

func GetDreams(w http.ResponseWriter, r *http.Request) {
	rows, err := db.DB.Query("SELECT id,title,content,mood,is_public,created_at from dreams ORDER BY created_at DESC")
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Failed to load dreams", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var dreams []Dream
	for rows.Next() {
		var dream Dream
		err := rows.Scan(
			&dream.Id,
			&dream.Title,
			&dream.Content,
			&dream.Mood,
			&dream.IsPublic,
			&dream.CreatedAt,
		)
		if err != nil {
			log.Printf("Row scan error: %v", err)
			continue
		}
		dreams = append(dreams, dream)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Rows iteration error: %v", err)
		http.Error(w, "Failed to load dreams", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles("templates/dream.html")
	if err != nil {
		log.Printf("Template parsing error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, dreams); err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
func NewDreamPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/new.html")
	if err != nil {
		log.Printf("Template parsing error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, nil); err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func GetDream(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid dream ID", http.StatusBadRequest)
		return
	}

	var dream Dream
	query := "SELECT id, title, content, mood, is_public, created_at FROM dreams WHERE id = $1"
	row := db.DB.QueryRow(query, id)

	err = row.Scan(&dream.Id, &dream.Title, &dream.Content, &dream.Mood, &dream.IsPublic, &dream.CreatedAt)
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Dream not found", http.StatusNotFound)
		return
	}

	tmpl, err := template.ParseFiles("templates/view.html")
	if err != nil {
		log.Printf("Template parsing error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, dream); err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func EditDreamPage(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid dream ID", http.StatusBadRequest)
		return
	}

	var dream Dream
	query := "SELECT id, title, content, mood, is_public, created_at FROM dreams WHERE id = $1"
	row := db.DB.QueryRow(query, id)

	err = row.Scan(&dream.Id, &dream.Title, &dream.Content, &dream.Mood, &dream.IsPublic, &dream.CreatedAt)
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Dream not found", http.StatusNotFound)
		return
	}

	tmpl, err := template.ParseFiles("templates/edit.html")
	if err != nil {
		log.Printf("Template parsing error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, dream); err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func UpdateDream(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid dream ID", http.StatusBadRequest)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		log.Printf("Form parsing error: %v", err)
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	title := r.FormValue("title")
	content := r.FormValue("content")
	mood := r.FormValue("mood")
	isPublic := r.FormValue("is_public") == "on"

	// Validate input
	if err := validateDreamInput(title, content, mood); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Sanitize input to prevent XSS
	title = sanitizeInput(title)
	content = sanitizeInput(content)
	mood = sanitizeInput(mood)

	query := "UPDATE dreams SET title = $1, content = $2, mood = $3, is_public = $4 WHERE id = $5"
	_, err = db.DB.Exec(query, title, content, mood, isPublic, id)
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Failed to update dream. Please try again.", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/dreams/"+idStr, http.StatusSeeOther)
}

func GetPublicDreams(w http.ResponseWriter, r *http.Request) {
	rows, err := db.DB.Query("SELECT id,title,content,mood,is_public,created_at from dreams WHERE is_public = true ORDER BY created_at DESC")
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Failed to load dreams", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var dreams []Dream
	for rows.Next() {
		var dream Dream
		err := rows.Scan(
			&dream.Id,
			&dream.Title,
			&dream.Content,
			&dream.Mood,
			&dream.IsPublic,
			&dream.CreatedAt,
		)
		if err != nil {
			log.Printf("Row scan error: %v", err)
			continue
		}
		dreams = append(dreams, dream)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Rows iteration error: %v", err)
		http.Error(w, "Failed to load dreams", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles("templates/public.html")
	if err != nil {
		log.Printf("Template parsing error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, dreams); err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func DeleteDream(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid dream ID", http.StatusBadRequest)
		return
	}

	query := "DELETE FROM dreams WHERE id = $1"
	result, err := db.DB.Exec(query, id)
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Failed to delete dream", http.StatusInternalServerError)
		return
	}

	// Check if any row was affected
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

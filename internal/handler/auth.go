package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/Aswanidev-vs/lucid/db"
	"github.com/Aswanidev-vs/lucid/internal/auth"
	"github.com/Aswanidev-vs/lucid/internal/middleware"
	"github.com/Aswanidev-vs/lucid/internal/model"
	"golang.org/x/crypto/bcrypt"
)

// ShowLoginPage renders the login page
func ShowLoginPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := ParseTemplate("templates/login.html")
	if err != nil {
		log.Printf("Template parsing error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Check if already logged in
	claims := middleware.GetUserClaims(r)
	if err := tmpl.Execute(w, map[string]interface{}{
		"IsLoggedIn": claims != nil,
		"Username":   getUsername(claims),
	}); err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// ShowSignupPage renders the signup page
func ShowSignupPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := ParseTemplate("templates/signup.html")
	if err != nil {
		log.Printf("Template parsing error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	claims := middleware.GetUserClaims(r)
	if err := tmpl.Execute(w, map[string]interface{}{
		"IsLoggedIn": claims != nil,
		"Username":   getUsername(claims),
	}); err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// HandleSignup processes user registration
func HandleSignup(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	// Validate input
	if err := validateSignup(username, email, password, confirmPassword); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if username or email already exists
	var exists bool
	err := db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username=$1 OR email=$2)", username, email).Scan(&exists)
	if err != nil {
		log.Printf("Database error checking user existence: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if exists {
		http.Error(w, "Username or email already taken", http.StatusConflict)
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Password hashing error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Insert user
	var userID int
	err = db.DB.QueryRow(
		"INSERT INTO users (username, email, password_hash, dreamer_type) VALUES ($1, $2, $3, $4) RETURNING id",
		username, email, string(hashedPassword), "Explorer",
	).Scan(&userID)
	if err != nil {
		log.Printf("Database error creating user: %v", err)
		http.Error(w, "Failed to create account. Please try again.", http.StatusInternalServerError)
		return
	}

	// Generate JWT token
	token, err := auth.GenerateToken(userID, username, email)
	if err != nil {
		log.Printf("Token generation error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Check if request accepts JSON (API call)
	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(model.AuthResponse{
			Token:   token,
			Message: "Account created successfully!",
			User: model.User{
				ID:          userID,
				Username:    username,
				Email:       email,
				DreamerType: "Explorer",
			},
		})
		return
	}

	// Set cookie and redirect for browser requests
	middleware.SetAuthCookie(w, token)
	http.Redirect(w, r, "/feed", http.StatusSeeOther)
}

// HandleLogin processes user login
func HandleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")

	if email == "" || password == "" {
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	// Find user by email
	var user model.User
	err := db.DB.QueryRow(
		"SELECT id, username, email, password_hash, bio, avatar_url, dreamer_type, is_public_profile FROM users WHERE email=$1",
		email,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.Bio, &user.AvatarURL, &user.DreamerType, &user.IsPublicProfile)
	if err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	// Generate JWT token
	token, err := auth.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		log.Printf("Token generation error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(model.AuthResponse{
			Token:   token,
			Message: "Welcome back, " + user.Username + "!",
			User:    user,
		})
		return
	}

	middleware.SetAuthCookie(w, token)
	http.Redirect(w, r, "/feed", http.StatusSeeOther)
}

// HandleLogout logs the user out
func HandleLogout(w http.ResponseWriter, r *http.Request) {
	middleware.ClearAuthCookie(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// GetCurrentUser returns the currently authenticated user info
func GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r)
	if claims == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(model.APIResponse{
			Success: false,
			Error:   "Not authenticated",
		})
		return
	}

	var user model.User
	err := db.DB.QueryRow(
		"SELECT id, username, email, bio, avatar_url, dreamer_type, is_public_profile, created_at FROM users WHERE id=$1",
		claims.UserID,
	).Scan(&user.ID, &user.Username, &user.Email, &user.Bio, &user.AvatarURL,
		&user.DreamerType, &user.IsPublicProfile, &user.CreatedAt)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(model.APIResponse{
			Success: false,
			Error:   "User not found",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model.APIResponse{
		Success: true,
		Data:    user,
	})
}

// validation helpers
func validateSignup(username, email, password, confirmPassword string) error {
	username = strings.TrimSpace(username)
	email = strings.TrimSpace(email)

	if username == "" {
		return errors.New("username is required")
	}
	if utf8.RuneCountInString(username) < 3 {
		return errors.New("username must be at least 3 characters")
	}
	if utf8.RuneCountInString(username) > 50 {
		return errors.New("username must be less than 50 characters")
	}

	if email == "" {
		return errors.New("email is required")
	}
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return errors.New("invalid email address")
	}

	if password == "" {
		return errors.New("password is required")
	}
	if utf8.RuneCountInString(password) < 6 {
		return errors.New("password must be at least 6 characters")
	}
	if utf8.RuneCountInString(password) > 128 {
		return errors.New("password must be less than 128 characters")
	}

	if password != confirmPassword {
		return errors.New("passwords do not match")
	}

	return nil
}

func getUsername(claims *auth.Claims) string {
	if claims == nil {
		return ""
	}
	return claims.Username
}

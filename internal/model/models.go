package model

import "time"

// User represents a registered user
type User struct {
	ID              int       `json:"id"`
	Username        string    `json:"username"`
	Email           string    `json:"email"`
	PasswordHash    string    `json:"-"`
	Bio             string    `json:"bio"`
	AvatarURL       string    `json:"avatar_url"`
	DreamerType     string    `json:"dreamer_type"`
	IsPublicProfile bool      `json:"is_public_profile"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	// Computed fields
	DreamCount int `json:"dream_count,omitempty"`
	TotalLikes int `json:"total_likes,omitempty"`
}

// Dream represents a dream entry
type Dream struct {
	ID            int       `json:"id"`
	UserID        int       `json:"user_id"`
	Title         string    `json:"title"`
	Content       string    `json:"content"`
	CategoryID    *int      `json:"category_id,omitempty"`
	Mood          string    `json:"mood"`
	Visibility    string    `json:"visibility"`
	IsLucid       bool      `json:"is_lucid"`
	IsRecurring   bool      `json:"is_recurring"`
	IsNightmare   bool      `json:"is_nightmare"`
	LucidityLevel int       `json:"lucidity_level"`
	DreamTags     []string  `json:"dream_tags"`
	LikeCount     int       `json:"like_count"`
	CommentCount  int       `json:"comment_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	// Joined fields
	Username      string `json:"username,omitempty"`
	CategoryName  string `json:"category_name,omitempty"`
	CategoryEmoji string `json:"category_emoji,omitempty"`
	IsLikedByMe   bool   `json:"is_liked_by_me,omitempty"`
}

// Comment represents a comment on a dream
type Comment struct {
	ID        int       `json:"id"`
	DreamID   int       `json:"dream_id"`
	UserID    int       `json:"user_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	// Joined fields
	Username  string `json:"username,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

// Like represents a like on a dream
type Like struct {
	ID        int       `json:"id"`
	DreamID   int       `json:"dream_id"`
	UserID    int       `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

// DreamCategory represents a dream category
type DreamCategory struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Emoji       string `json:"emoji"`
	Description string `json:"description"`
}

// TrendingTheme represents a trending dream theme
type TrendingTheme struct {
	ID            int       `json:"id"`
	Theme         string    `json:"theme"`
	MentionCount  int       `json:"mention_count"`
	LastMentioned time.Time `json:"last_mentioned"`
}

// ExportData represents all user data for export
type ExportData struct {
	User          User      `json:"user"`
	Dreams        []Dream   `json:"dreams"`
	Comments      []Comment `json:"comments"`
	Likes         []Like    `json:"likes"`
	TotalDreams   int       `json:"total_dreams"`
	PublicDreams  int       `json:"public_dreams"`
	PrivateDreams int       `json:"private_dreams"`
}

// AuthResponse is the response after authentication
type AuthResponse struct {
	Token   string `json:"token"`
	User    User   `json:"user"`
	Message string `json:"message"`
}

// APIResponse is a generic API response
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// LoginRequest represents login credentials
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// SignupRequest represents signup data
type SignupRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

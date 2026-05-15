package middleware

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/Aswanidev-vs/lucid/internal/auth"
)

type contextKey string

const UserContextKey contextKey = "user"

func AuthRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			http.Error(w, "Unauthorized: missing token", http.StatusUnauthorized)
			return
		}

		claims, err := auth.ValidateToken(token)
		if err != nil {
			http.Error(w, "Unauthorized: invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token != "" {
			claims, err := auth.ValidateToken(token)
			if err == nil {
				ctx := context.WithValue(r.Context(), UserContextKey, claims)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func GetUserClaims(r *http.Request) *auth.Claims {
	claims, ok := r.Context().Value(UserContextKey).(*auth.Claims)
	if !ok {
		return nil
	}
	return claims
}

func GetUserID(r *http.Request) int {
	claims := GetUserClaims(r)
	if claims == nil {
		return 0
	}
	return claims.UserID
}

func extractToken(r *http.Request) string {
	bearer := r.Header.Get("Authorization")
	if bearer != "" {
		if strings.HasPrefix(bearer, "Bearer ") {
			return strings.TrimPrefix(bearer, "Bearer ")
		}
		return bearer
	}

	cookie, err := r.Cookie("token")
	if err == nil {
		return cookie.Value
	}

	token := r.URL.Query().Get("token")
	if token != "" {
		return token
	}

	return ""
}

func SetAuthCookie(w http.ResponseWriter, token string) {
	secure := os.Getenv("APP_ENV") == "production"
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		MaxAge:   72 * 60 * 60,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func ClearAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   os.Getenv("APP_ENV") == "production",
		SameSite: http.SameSiteLaxMode,
	})
}

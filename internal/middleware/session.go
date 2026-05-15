package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"sync"
	"time"
)

type sessionKey string

const SessionKey sessionKey = "session_id"

type SessionStore struct {
	mu     sync.RWMutex
	data   map[string]map[string]interface{}
}

var GlobalSessionStore = &SessionStore{
	data: make(map[string]map[string]interface{}),
}

func SessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionID := ""

		if cookie, err := r.Cookie("session"); err == nil && cookie.Value != "" {
			sessionID = cookie.Value
			if !GlobalSessionStore.Exists(sessionID) {
				sessionID = ""
			}
		}

		if sessionID == "" {
			b := make([]byte, 16)
			rand.Read(b)
			sessionID = hex.EncodeToString(b)
			GlobalSessionStore.Create(sessionID)

			secure := os.Getenv("APP_ENV") == "production"
			http.SetCookie(w, &http.Cookie{
				Name:     "session",
				Value:    sessionID,
				Path:     "/",
				MaxAge:   86400 * 30,
				HttpOnly: true,
				Secure:   secure,
				SameSite: http.SameSiteLaxMode,
			})
		}

		GlobalSessionStore.Touch(sessionID)
		ctx := context.WithValue(r.Context(), SessionKey, sessionID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetSessionID(r *http.Request) string {
	if v, ok := r.Context().Value(SessionKey).(string); ok {
		return v
	}
	return ""
}

func (s *SessionStore) Create(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[id] = map[string]interface{}{
		"created_at": time.Now(),
		"last_seen":  time.Now(),
	}
}

func (s *SessionStore) Exists(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.data[id]
	return ok
}

func (s *SessionStore) Touch(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d, ok := s.data[id]; ok {
		d["last_seen"] = time.Now()
	}
}

func (s *SessionStore) Get(id string, key string) interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if d, ok := s.data[id]; ok {
		return d[key]
	}
	return nil
}

func (s *SessionStore) Set(id string, key string, val interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d, ok := s.data[id]; ok {
		d[key] = val
	}
}

func (s *SessionStore) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, id)
}

func (s *SessionStore) Cleanup(maxAge time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for id, d := range s.data {
		if lastSeen, ok := d["last_seen"].(time.Time); ok {
			if now.Sub(lastSeen) > maxAge {
				delete(s.data, id)
			}
		}
	}
}

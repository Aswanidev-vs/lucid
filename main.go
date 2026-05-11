package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Aswanidev-vs/lucid/db"
	"github.com/Aswanidev-vs/lucid/internal/handler"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists (for local development)
	// Render uses environment variables from the dashboard, not .env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found — using system environment variables")
	}

	db.Connect()
	db.Table()

	r := chi.NewRouter()

	// Security middleware
	r.Use(middleware.Logger)                    // Request logging
	r.Use(middleware.Recoverer)                 // Panic recovery
	r.Use(middleware.RealIP)                    // Get real IP from headers
	r.Use(middleware.Timeout(30 * time.Second)) // Request timeout

	// Security headers
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			next.ServeHTTP(w, r)
		})
	})

	r.Get("/", handler.HomeHandler)
	r.Get("/dreams", handler.GetDreams)
	r.Get("/dreams/public", handler.GetPublicDreams)
	r.Get("/dreams/new", handler.NewDreamPage)
	r.Post("/dreams/new", handler.CreateDream)
	r.Get("/dreams/{id}", handler.GetDream)
	r.Get("/dreams/{id}/edit", handler.EditDreamPage)
	r.Post("/dreams/{id}", handler.UpdateDream)
	r.Post("/dreams/{id}/delete", handler.DeleteDream)
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("templates/static/"))))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server is running at http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

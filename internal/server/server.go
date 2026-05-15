package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Aswanidev-vs/lucid/config"
	"github.com/Aswanidev-vs/lucid/db"
	"github.com/Aswanidev-vs/lucid/internal/handler"
	"github.com/Aswanidev-vs/lucid/internal/middleware"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"
)

func Start(cfg *config.Config) error {
	if err := db.Connect(cfg.DBURL); err != nil {
		return err
	}
	defer db.Close()

	if err := db.RunMigrations(); err != nil {
		return err
	}

	r := chi.NewRouter()

	// Middleware stack
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Timeout(30 * time.Second))
	r.Use(middleware.RequestID)
	r.Use(middleware.SessionMiddleware)
	r.Use(middleware.StructuredLogger)
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.SecurityHeaders)

	// CORS
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   cfg.CORS.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "Accept", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           86400,
	})
	r.Use(corsHandler.Handler)

	// Static files
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("templates/static/"))))

	// Health check
	r.Get("/healthz", handler.HealthCheck)

	// Public routes (optional auth)
	r.Group(func(r chi.Router) {
		r.Use(middleware.OptionalAuth)
		r.Get("/", handler.HomeHandler)
		r.Get("/login", handler.ShowLoginPage)
		r.Get("/signup", handler.ShowSignupPage)
		r.Post("/login", handler.HandleLogin)
		r.Post("/signup", handler.HandleSignup)
		r.Get("/dreams/{id}", handler.GetDream)
		r.Get("/trending", handler.HandleTrending)
		r.Get("/dreams/{id}/comments", handler.HandleGetComments)
		r.Get("/profile/{id}", handler.ShowProfile)
	})

	// Authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthRequired)

		r.Get("/feed", handler.ShowFeed)
		r.Get("/dreams/public", handler.GetPublicDreams)
		r.Post("/logout", handler.HandleLogout)
		r.Get("/api/me", handler.GetCurrentUser)

		r.Get("/dreams", handler.GetDreams)
		r.Get("/dreams/new", handler.NewDreamPage)
		r.Post("/dreams/new", handler.CreateDream)
		r.Get("/dreams/{id}/edit", handler.EditDreamPage)
		r.Post("/dreams/{id}", handler.UpdateDream)
		r.Post("/dreams/{id}/delete", handler.DeleteDream)

		r.Post("/dreams/{id}/like", handler.HandleLikeDream)
		r.Post("/dreams/{id}/comment", handler.HandleAddComment)

		r.Post("/profile/update", handler.UpdateProfile)
		r.Get("/export", handler.ExportUserData)
		r.Post("/import", handler.ImportUserData)
	})

	// API routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.OptionalAuth)
		r.Get("/api/trending", handler.HandleTrending)
		r.Get("/api/dreams/{id}/comments", handler.HandleGetComments)
	})

	srv := &http.Server{
		Addr:         cfg.AppPort(),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Session cleanup every 30 minutes
	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				middleware.GlobalSessionStore.Cleanup(72 * time.Hour)
			}
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("%s running on %s", cfg.AppName, cfg.AppPort())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return srv.Shutdown(ctx)
}

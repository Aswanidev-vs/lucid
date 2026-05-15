package main

import (
	"log"

	"github.com/Aswanidev-vs/lucid/config"
	"github.com/Aswanidev-vs/lucid/internal/auth"
	"github.com/Aswanidev-vs/lucid/internal/server"
)

func main() {
	cfg := config.Load()

	auth.SetSecret(cfg.JWTSecret())

	if err := server.Start(cfg); err != nil {
		log.Fatalf("Server exited with error: %v", err)
	}
}

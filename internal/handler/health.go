package handler

import (
	"encoding/json"
	"net/http"

	"github.com/Aswanidev-vs/lucid/db"
)

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	dbStatus := "ok"
	if err := db.DB.Ping(); err != nil {
		dbStatus = "error: " + err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "ok",
		"database": dbStatus,
		"version":  "1.0.0",
	})
}

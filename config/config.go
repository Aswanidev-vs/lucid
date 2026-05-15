package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port    string
	Env     string // "development" or "production"
	DBURL   string
	JWTKey  string
	CORS    CORSConfig
	Cookie  CookieConfig
	AppName string
}

type CORSConfig struct {
	AllowedOrigins []string
}

type CookieConfig struct {
	Secure   bool
	HTTPOnly bool
	SameSite string
}

func Load() *Config {
	env := getEnv("APP_ENV", "development")
	return &Config{
		Port:    getEnv("PORT", "8080"),
		Env:     env,
		DBURL:   getEnv("DATABASE_URL", ""),
		JWTKey:  getEnv("JWT_SECRET", ""),
		AppName: "Lucid",
		CORS: CORSConfig{
			AllowedOrigins: parseOrigins(getEnv("CORS_ORIGINS", "http://localhost:8080")),
		},
		Cookie: CookieConfig{
			Secure:   env == "production",
			HTTPOnly: true,
			SameSite: getEnv("COOKIE_SAMESITE", "Lax"),
		},
	}
}

func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

func (c *Config) JWTSecret() []byte {
	if c.JWTKey == "" {
		return []byte("lucid-dev-secret-change-in-prod")
	}
	return []byte(c.JWTKey)
}

func (c *Config) AppPort() string {
	if p, err := strconv.Atoi(c.Port); err == nil && p > 0 && p <= 65535 {
		return ":" + c.Port
	}
	return ":8080"
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseOrigins(s string) []string {
	if s == "" || s == "*" {
		return []string{"*"}
	}
	var origins []string
	current := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == ',' || s[i] == ' ' {
			if len(current) > 0 {
				origins = append(origins, string(current))
				current = current[:0]
			}
			continue
		}
		current = append(current, s[i])
	}
	if len(current) > 0 {
		origins = append(origins, string(current))
	}
	return origins
}

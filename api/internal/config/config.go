package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port            string
	DBPath          string
	CookieSecure    bool
	SessionTTLHours int
	StaticDir       string
}

func Load() *Config {
	port := getEnv("PORT", "8080")
	dbPath := getEnv("DB_PATH", "./data/expense.db")

	cookieSecure := true
	if val := os.Getenv("COOKIE_SECURE"); val != "" {
		if parsed, err := strconv.ParseBool(val); err == nil {
			cookieSecure = parsed
		}
	}

	sessionTTL := 720
	if val := os.Getenv("SESSION_TTL_HOURS"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			sessionTTL = parsed
		}
	}

	staticDir := getEnv("STATIC_DIR", "../web/dist/web/browser")

	return &Config{
		Port:            port,
		DBPath:          dbPath,
		CookieSecure:    cookieSecure,
		SessionTTLHours: sessionTTL,
		StaticDir:       staticDir,
	}
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

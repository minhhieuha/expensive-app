package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	os.Clearenv()
	cfg := Load()

	if cfg.Port != "8080" {
		t.Errorf("expected Port=8080, got %s", cfg.Port)
	}
	if cfg.DBPath != "./data/expense.db" {
		t.Errorf("expected DBPath=./data/expense.db, got %s", cfg.DBPath)
	}
	if !cfg.CookieSecure {
		t.Errorf("expected CookieSecure=true, got %v", cfg.CookieSecure)
	}
	if cfg.SessionTTLHours != 720 {
		t.Errorf("expected SessionTTLHours=720, got %d", cfg.SessionTTLHours)
	}
	if cfg.StaticDir != "../web/dist/web/browser" {
		t.Errorf("expected StaticDir=../web/dist/web/browser, got %s", cfg.StaticDir)
	}
}

func TestLoadCustom(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("DB_PATH", ":memory:")
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("SESSION_TTL_HOURS", "24")
	t.Setenv("STATIC_DIR", "/public")

	cfg := Load()

	if cfg.Port != "9090" {
		t.Errorf("expected Port=9090, got %s", cfg.Port)
	}
	if cfg.DBPath != ":memory:" {
		t.Errorf("expected DBPath=:memory:, got %s", cfg.DBPath)
	}
	if cfg.CookieSecure {
		t.Errorf("expected CookieSecure=false, got %v", cfg.CookieSecure)
	}
	if cfg.SessionTTLHours != 24 {
		t.Errorf("expected SessionTTLHours=24, got %d", cfg.SessionTTLHours)
	}
	if cfg.StaticDir != "/public" {
		t.Errorf("expected StaticDir=/public, got %s", cfg.StaticDir)
	}
}

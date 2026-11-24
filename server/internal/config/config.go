// Package config reads all environment variables in one place.
package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port          string
	FrontendURL   string
	AllowedOrigin map[string]bool // CORS allow list; the frontend URL is allowed by default
	JWTSecret     []byte
	DBPath        string
	BotNumber     string // the WhatsApp number users add to their groups
	CookieSecure  bool
	DevMode       bool // enables dev-only endpoints (simulate a group without WhatsApp)
}

func Load() Config {
	_ = godotenv.Load() // .env is optional; real env vars win

	cfg := Config{
		Port:         get("PORT", "8081"),
		FrontendURL:  get("FRONTEND_URL", "http://localhost:3000"),
		DBPath:       get("DB_PATH", "internal/db/.data/botty.db"),
		BotNumber:    get("BOT_WHATSAPP_NUMBER", "+00 00000 00000"), // TODO: change - set the real bot number in .env
		CookieSecure: get("COOKIE_SECURE", "false") == "true",
		DevMode:      get("DEV_MODE", "false") == "true",
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// TODO: change - never run with this default outside local dev
		log.Println("WARNING: JWT_SECRET not set, using an insecure dev default")
		secret = "dev-insecure-secret-change-me"
	}
	cfg.JWTSecret = []byte(secret)

	// frontend URL is allowed by default, ALLOWED_ORIGINS (comma separated) adds more
	cfg.AllowedOrigin = map[string]bool{cfg.FrontendURL: true}
	for _, o := range strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",") {
		if o = strings.TrimSpace(o); o != "" {
			cfg.AllowedOrigin[o] = true
		}
	}
	return cfg
}

func get(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

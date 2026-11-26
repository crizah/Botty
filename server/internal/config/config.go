// Package config reads all environment variables in one place.
package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port           string
	FrontendURL    string
	AllowedOrigin  map[string]bool // CORS allow list; the frontend URL is allowed by default
	JWTSecret      []byte
	DBPath         string
	BotNumber      string // the WhatsApp number users add to their groups
	CookieSecure   bool
	DevMode        bool   // enables dev-only endpoints (simulate a group without WhatsApp)
	WAEnabled      bool   // connect to WhatsApp (prints a QR on first run)
	WASessionDB    string // whatsmeow's own session store; keep on a persistent volume
	WAWorkers      int    // message workers; a group always maps to the same one
	ResendKey      string // Resend API key for invite emails
	MailFrom       string
	SarvamKey      string // API key for Prism + the naming model; tag handling is off without it
	SarvamURL      string
	NamingModel    string  // the small chat model that names new tasks
	MatchThreshold float64 // min Prism probability to file under an existing task
	DupThreshold   float64 // min Prism probability to treat a message as an exact repeat
	TagMaxTasks    int     // how many recent open tasks are offered to the classifier
	ChainDepth     int     // most messages of a reply chain sent to the classifier
	MsgRetention   int     // days recent group messages are kept for reply chains
}

func Load() Config {
	_ = godotenv.Load() // .env is optional; real env vars win

	cfg := Config{
		Port:           get("PORT", "8081"),
		FrontendURL:    get("FRONTEND_URL", "http://localhost:3000"),
		DBPath:         get("DB_PATH", "internal/db/.data/botty.db"),
		BotNumber:      get("BOT_WHATSAPP_NUMBER", "+00 00000 00000"), // TODO: change - set the real bot number in .env
		CookieSecure:   get("COOKIE_SECURE", "false") == "true",
		DevMode:        get("DEV_MODE", "false") == "true",
		WAEnabled:      get("WA_ENABLED", "false") == "true",
		WASessionDB:    get("WA_SESSION_DB", "internal/db/.data/whatsapp.db"),
		WAWorkers:      atoi(get("WA_WORKERS", "8"), 8),
		ResendKey:      os.Getenv("RESEND_API_KEY"),
		MailFrom:       get("MAIL_FROM", "Botty <noreply@crizah.space>"),
		SarvamKey:      os.Getenv("SARVAM_API_KEY"),
		SarvamURL:      get("SARVAM_BASE_URL", "https://api.sarvam.ai"),
		NamingModel:    get("SARVAM_NAMING_MODEL", "sarvam-105b-conversations"),
		MatchThreshold: atof(get("TAG_MATCH_THRESHOLD", "0.6"), 0.6), // TODO: change - tune on real scores
		DupThreshold:   atof(get("DUP_MATCH_THRESHOLD", "0.7"), 0.7), // TODO: change - tune on real scores
		TagMaxTasks:    atoi(get("TAG_MAX_TASKS", "50"), 50),
		ChainDepth:     atoi(get("TAG_CHAIN_DEPTH", "3"), 3),
		MsgRetention:   atoi(get("MSG_RETENTION_DAYS", "7"), 7),
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

func atoi(s string, def int) int {
	if n, err := strconv.Atoi(s); err == nil && n > 0 {
		return n
	}
	return def
}

func atof(s string, def float64) float64 {
	if f, err := strconv.ParseFloat(s, 64); err == nil && f > 0 && f < 1 {
		return f
	}
	return def
}

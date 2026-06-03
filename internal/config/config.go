package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                 string
	DatabaseURL          string
	JWTSecret            string
	CORSOrigins          []string
	CookieSecure         bool
	CloudinaryAPIKey       string
	CloudinaryAPISecret    string
	CloudinaryCloudName    string
	CloudinaryUploadPreset string
	CloudinaryUploadFolder string
	CloudinaryUploadHook   string
	TTSVoice               string
	TTSLanguage            string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Port:                 envOr("PORT", "8080"),
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		JWTSecret:            os.Getenv("JWT_SECRET"),
		CORSOrigins:          splitCSV(envOr("CORS_ORIGINS", "http://localhost:3000")),
		CookieSecure:         envOr("COOKIE_SECURE", "false") == "true",
		CloudinaryAPIKey:       os.Getenv("CLOUDINARY_API_KEY"),
		CloudinaryAPISecret:    os.Getenv("CLOUDINARY_API_SECRET"),
		CloudinaryCloudName:    os.Getenv("CLOUDINARY_CLOUD_NAME"),
		CloudinaryUploadPreset: os.Getenv("CLOUDINARY_UPLOAD_PRESET"),
		CloudinaryUploadFolder: envOr("CLOUDINARY_UPLOAD_FOLDER", "sistem-antrian/videos"),
		CloudinaryUploadHook:   os.Getenv("CLOUDINARY_UPLOAD_HOOK"),
		TTSVoice:               os.Getenv("TTS_VOICE"),
		TTSLanguage:            os.Getenv("TTS_LANGUAGE"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
